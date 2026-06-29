package database

import (
	"testing"
)

// Każda funkcja testowa w Go musi zaczynać się od słowa "Test" i przyjmować wskaźnik na testing.T
func TestInitDBAndTables(t *testing.T) {
	// 1. Odpalamy naszą funkcję konfigurującą bazę
	db := InitDB()

	// Sprawdzamy, czy nie dostaliśmy pustego wskaźnika
	if db == nil {
		t.Fatal("Funkcja InitDB zwróciła nil, a oczekiwano prawidłowego wskaźnika do bazy danych.")
	}
	// Dobry nawyk: zamykamy połączenie po zakończeniu testu
	defer db.Close()

	// 2. Definiujemy listę tabel, które funkcja createTables powinna była wyryć w Postgresie
	expectedTables := []string{"users", "alerts", "game_price_history"}

	// 3. Sprawdzamy w pętli każdą tabelę, odpytując systemowy słownik Postgresa (information_schema)
	for _, tableName := range expectedTables {
		var exists bool

		// Zapytanie SQL sprawdzające, czy dana tabela istnieje w schemacie publicznym
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			);`

		// Wykonujemy zapytanie. QueryRow i Scan to standardowe metody z manuala do wyciągania jednego wiersza
		err := db.QueryRow(query, tableName).Scan(&exists)
		if err != nil {
			t.Fatalf("Błąd podczas sprawdzania czy tabela %s istnieje: %v", tableName, err)
		}

		// Jeśli baza mówi, że tabela nie istnieje – oblewamy test za pomocą t.Errorf
		if !exists {
			t.Errorf("Test oblewany: Tabela '%s' nie została utworzona w bazie danych!", tableName)
		}
	}
}
