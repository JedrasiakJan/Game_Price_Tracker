package database

import (
	"database/sql"
	"fmt"
)

type TriggeredAlert struct {
	DiscordUserID string
	GameTitle     string
	TargetPrice   float64
	SalePrice     float64
}

type UserAlert struct {
	GameID       string
	GameTitle    string
	TargetPrice  float64
	CurrentPrice float64
}

func CreateAlert(db *sql.DB, chatID string, gameID string, title string, targetPrice float64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	var userID int
	userQuery := `
	INSERT INTO users (chat_id, platform)
	VALUES ($1, 'discord')
	ON CONFLICT (chat_id) DO UPDATE SET platform = 'discord'
	RETURNING id;`

	err = tx.QueryRow(userQuery, chatID).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	gameQuery := `
	INSERT INTO games (game_id, title, normal_price_usd, store_id)
	VALUES ($1, $2, 0.00, '0')
	ON CONFLICT (game_id) DO NOTHING;`
	_, err = tx.Exec(gameQuery, gameID, title)
	if err != nil {
		return fmt.Errorf("failed to ensure game exists: %w", err)
	}

	updateAlertQuery := `
		UPDATE alerts 
		SET price_drop = $3 
		WHERE user_id = $1 AND game_id = $2 AND is_active = TRUE;`

	res, err := tx.Exec(updateAlertQuery, userID, gameID, targetPrice)
	if err != nil {
		return fmt.Errorf("failed to update existing alert: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		insertAlertQuery := `
			INSERT INTO alerts (user_id, game_id, price_drop, is_active)
			VALUES ($1, $2, $3, TRUE);`
		_, err = tx.Exec(insertAlertQuery, userID, gameID, targetPrice)
		if err != nil {
			return fmt.Errorf("failed to insert new alert: %w", err)
		}
	}

	return tx.Commit()
}

func GetActiveAlertGameIDs(db *sql.DB) ([]string, error) {
	query := `SELECT DISTINCT game_id FROM alerts WHERE is_active = TRUE;`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func CheckAlertsInDB(db *sql.DB, gameID string, currentPrice float64) ([]TriggeredAlert, error) {
	query := `
		SELECT u.chat_id, g.title, a.price_drop
		FROM alerts a
		JOIN users u ON a.user_id = u.id
		JOIN games g ON a.game_id = g.game_id
		WHERE a.game_id = $1 AND a.is_active = TRUE AND $2 <= a.price_drop;`

	rows, err := db.Query(query, gameID, currentPrice)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggered []TriggeredAlert
	for rows.Next() {
		var ta TriggeredAlert
		err := rows.Scan(&ta.DiscordUserID, &ta.GameTitle, &ta.TargetPrice)
		if err != nil {
			return nil, err
		}
		ta.SalePrice = currentPrice
		triggered = append(triggered, ta)
	}
	return triggered, nil
}

func DisableAlert(db *sql.DB, discordUserID string, gameID string) error {
	query := `
		UPDATE alerts 
		SET is_active = FALSE 
		WHERE game_id = $1 AND user_id = (SELECT id FROM users WHERE chat_id = $2);`
	_, err := db.Exec(query, gameID, discordUserID)
	return err
}

func GetActiveAlertForUsers(db *sql.DB, chatID string) ([]UserAlert, error) {
	query := `
	SELECT a.game_id, g.title, a.price_drop,
	       COALESCE((
	           SELECT sale_price_usd 
	           FROM game_price_history 
	           WHERE game_id = g.game_id 
	           ORDER BY timestamp DESC LIMIT 1
	       ), 0.00) AS current_price
	FROM alerts a
	JOIN users u ON a.user_id = u.id
	JOIN games g ON a.game_id = g.game_id
	WHERE u.chat_id = $1 AND a.is_active = True;`

	rows, err := db.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []UserAlert
	for rows.Next() {
		var ua UserAlert
		if err := rows.Scan(&ua.GameID, &ua.GameTitle, &ua.TargetPrice, &ua.CurrentPrice); err != nil {
			return nil, err
		}
		alerts = append(alerts, ua)
	}
	return alerts, nil
}

func DeactivateAlertManually(db *sql.DB, chatID string, gameID string) (bool, error) {
	query := `
	UPDATE alerts
	SET is_active = FALSE
	WHERE game_id = $1
		AND user_id = (SELECT id FROM users WHERE chat_id = $2)
		AND is_active = TRUE`
	res, err := db.Exec(query, gameID, chatID)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}
