package database

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func InitDB() *sql.DB {
	connStr := "host=localhost user=steam_admin password=steam_secure_password123 dbname=steam_api_prod port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error configuring database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Cannot connect to database (Ping failed): %v", err)
	}
	log.Println("Database connection established using database/sql!")
	createTables(db)
	return db
}

func createTables(db *sql.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				chat_id VARCHAR(255) UNIQUE NOT NULL,
				platform VARCHAR(50) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS alerts (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				game_id VARCHAR(100) NOT NULL,
				game_name VARCHAR(255) NOT NULL,
				price_drop NUMERIC(10, 2) NOT NULL,
				is_active BOOLEAN DEFAULT TRUE,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS game_price_history (
				id SERIAL PRIMARY KEY,
				game_id VARCHAR(100) NOT NULL,
				price_usd NUMERIC(10, 2) NOT NULL,
				timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		
		);`,
	}
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatalf("Error creating table: %v", err)
		}
	}
	log.Println("Database tables checked/created successfully.")
}
