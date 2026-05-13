package testutil

import (
	"database/sql"
	"fmt"
)

type MessageRow struct {
	ID        string
	SessionID string
	CreatedAt int64
	UpdatedAt int64
	Role      string
	Cost      float64
	Provider  string
	TokensIn  int64
	TokensOut int64
}

func SeedDB(db *sql.DB, rows []MessageRow) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS message (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			data TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	for _, r := range rows {
		data := fmt.Sprintf(
			`{"role":"%s","providerID":"%s","cost":%.6f,"tokens":{"input":%d,"output":%d},"time":{"created":%d}}`,
			r.Role, r.Provider, r.Cost, r.TokensIn, r.TokensOut, r.CreatedAt,
		)
		_, err := db.Exec(
			`INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)`,
			r.ID, r.SessionID, r.CreatedAt, r.UpdatedAt, data,
		)
		if err != nil {
			return fmt.Errorf("failed to insert row %q: %w", r.ID, err)
		}
	}

	return nil
}
