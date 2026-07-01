package database

import (
	"database/sql"
	"fmt"
	"steam_api/internal/client"
)

func SaveDeals(db *sql.DB, deals []client.CheapSharkDeal) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	// ON CONFLICT (game_id) DO UPDATE sprawia, że jeśli gra już istnieje, tylko aktualizujemy jej dane
	gameUpsertQuery := `
		INSERT INTO games (game_id, title, normal_price_usd, store_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (game_id) 
		DO UPDATE SET 
			title = EXCLUDED.title, 
			normal_price_usd = EXCLUDED.normal_price_usd, 
			store_id = EXCLUDED.store_id;`
	historyInsertQuery := `
		INSERT INTO game_price_history (game_id, sale_price_usd)
		VALUES ($1, $2);`
	for _, deal := range deals {
		_, err := tx.Exec(gameUpsertQuery, deal.GameID, deal.Title, deal.NormalPrice, deal.StoreID)
		if err != nil {
			return fmt.Errorf("failed to upsert game %s: %w", deal.Title, err)
		}
		_, err = tx.Exec(historyInsertQuery, deal.GameID, deal.SalePrice)
		if err != nil {
			return fmt.Errorf("failed to insert history for game %s: %w", deal.Title, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
