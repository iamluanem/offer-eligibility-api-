package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"offer-eligibility-api/internal/models"
)

// DB wraps the database connection and provides methods for data access.
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection and initializes the schema.
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates the necessary tables if they don't exist.
func (db *DB) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS offers (
			id TEXT PRIMARY KEY,
			merchant_id TEXT NOT NULL,
			mcc_whitelist TEXT NOT NULL,
			active INTEGER NOT NULL,
			min_txn_count INTEGER NOT NULL,
			lookback_days INTEGER NOT NULL,
			starts_at TEXT NOT NULL,
			ends_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			merchant_id TEXT NOT NULL,
			mcc TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			approved_at TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_id ON transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_merchant_id ON transactions(merchant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mcc ON transactions(mcc)`,
		`CREATE INDEX IF NOT EXISTS idx_approved_at ON transactions(approved_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_approved_at ON transactions(user_id, approved_at)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

// UpsertOffer creates or updates an offer.
func (db *DB) UpsertOffer(offer models.Offer) error {
	mccWhitelistJSON := serializeMCCWhitelist(offer.MCCWhitelist)
	
	query := `INSERT INTO offers (
		id, merchant_id, mcc_whitelist, active, min_txn_count, 
		lookback_days, starts_at, ends_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		merchant_id = excluded.merchant_id,
		mcc_whitelist = excluded.mcc_whitelist,
		active = excluded.active,
		min_txn_count = excluded.min_txn_count,
		lookback_days = excluded.lookback_days,
		starts_at = excluded.starts_at,
		ends_at = excluded.ends_at,
		updated_at = excluded.updated_at`

	_, err := db.conn.Exec(
		query,
		offer.ID,
		offer.MerchantID,
		mccWhitelistJSON,
		offer.Active,
		offer.MinTxnCount,
		offer.LookbackDays,
		offer.StartsAt.Format(time.RFC3339),
		offer.EndsAt.Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to upsert offer: %w", err)
	}

	return nil
}

// InsertTransactions inserts multiple transactions in a single transaction.
func (db *DB) InsertTransactions(transactions []models.Transaction) (int, error) {
	if len(transactions) == 0 {
		return 0, nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO transactions (
		id, user_id, merchant_id, mcc, amount_cents, approved_at
	) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, txn := range transactions {
		_, err := stmt.Exec(
			txn.ID,
			txn.UserID,
			txn.MerchantID,
			txn.MCC,
			txn.AmountCents,
			txn.ApprovedAt.Format(time.RFC3339),
		)
		if err != nil {
			return 0, fmt.Errorf("failed to insert transaction %s: %w", txn.ID, err)
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return inserted, nil
}

// GetActiveOffers returns all active offers at the given time.
func (db *DB) GetActiveOffers(now time.Time) ([]models.Offer, error) {
	query := `SELECT id, merchant_id, mcc_whitelist, active, min_txn_count, 
		lookback_days, starts_at, ends_at
		FROM offers
		WHERE active = 1 
		AND starts_at <= ? 
		AND ends_at >= ?`

	rows, err := db.conn.Query(query, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to query active offers: %w", err)
	}
	defer rows.Close()

	var offers []models.Offer
	for rows.Next() {
		var offer models.Offer
		var mccWhitelistJSON string
		var startsAtStr, endsAtStr string

		err := rows.Scan(
			&offer.ID,
			&offer.MerchantID,
			&mccWhitelistJSON,
			&offer.Active,
			&offer.MinTxnCount,
			&offer.LookbackDays,
			&startsAtStr,
			&endsAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan offer: %w", err)
		}

		offer.MCCWhitelist = deserializeMCCWhitelist(mccWhitelistJSON)

		offer.StartsAt, err = time.Parse(time.RFC3339, startsAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse starts_at: %w", err)
		}

		offer.EndsAt, err = time.Parse(time.RFC3339, endsAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ends_at: %w", err)
		}

		offers = append(offers, offer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating offers: %w", err)
	}

	return offers, nil
}

// CountMatchingTransactions counts transactions that match an offer for a user
// within the lookback window.
func (db *DB) CountMatchingTransactions(
	userID string,
	offer models.Offer,
	now time.Time,
) (int, error) {
	lookbackStart := now.AddDate(0, 0, -offer.LookbackDays)

	// Build the query to match either merchant_id or mcc in whitelist
	query := `SELECT COUNT(*) FROM transactions
		WHERE user_id = ?
		AND approved_at >= ?
		AND approved_at <= ?
		AND (
			merchant_id = ?`

	args := []interface{}{userID, lookbackStart.Format(time.RFC3339), now.Format(time.RFC3339), offer.MerchantID}

	if len(offer.MCCWhitelist) > 0 {
		query += " OR mcc IN ("
		for i, mcc := range offer.MCCWhitelist {
			if i > 0 {
				query += ","
			}
			query += "?"
			args = append(args, mcc)
		}
		query += ")"
	}

	query += ")"

	var count int
	err := db.conn.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count matching transactions: %w", err)
	}

	return count, nil
}

// serializeMCCWhitelist converts a slice of MCC codes to a JSON string.
func serializeMCCWhitelist(mccList []string) string {
	if len(mccList) == 0 {
		return "[]"
	}
	data, err := json.Marshal(mccList)
	if err != nil {
		// Fallback to comma-separated if JSON fails
		return strings.Join(mccList, ",")
	}
	return string(data)
}

// deserializeMCCWhitelist converts a serialized MCC whitelist back to a slice.
func deserializeMCCWhitelist(serialized string) []string {
	if serialized == "" || serialized == "[]" {
		return []string{}
	}
	
	// Try JSON parsing first
	var result []string
	if err := json.Unmarshal([]byte(serialized), &result); err == nil {
		return result
	}
	
	// Fallback to comma-separated format for backward compatibility
	return strings.Split(serialized, ",")
}

