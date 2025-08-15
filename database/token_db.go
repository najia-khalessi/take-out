package database

import (
	"take-out/monitoring"
	"time"
)

// AddTokenToBlacklist adds a JTI to the blacklist.
func AddTokenToBlacklist(jti string, expiresAt time.Time) error {
	return monitoring.RecordDBTime("AddTokenToBlacklist", func() error {
		query := `INSERT INTO token_blacklist (jti, expires_at) VALUES ($1, $2)`
		_, err := DB.Exec(query, jti, expiresAt)
		return err
	})
}

// IsTokenBlacklisted checks if a JTI is in the blacklist.
func IsTokenBlacklisted(jti string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM token_blacklist WHERE jti = $1)`
	err := monitoring.RecordDBTime("IsTokenBlacklisted", func() error {
		return DB.QueryRow(query, jti).Scan(&exists)
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}
