package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/trogers1052/stock-alert-system/internal/models"
)

// CreateSignalFeedback inserts a new feedback entry.
func (db *DB) CreateSignalFeedback(fb *models.SignalFeedback) error {
	query := `
		INSERT INTO signal_feedback (symbol, signal, action, confidence, feedback_timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	now := time.Now()
	err := db.conn.QueryRow(query,
		fb.Symbol, fb.Signal, fb.Action, fb.Confidence, fb.FeedbackTimestamp, now,
	).Scan(&fb.ID, &fb.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create signal feedback: %w", err)
	}
	return nil
}

// GetSignalFeedback returns feedback entries with optional filters.
func (db *DB) GetSignalFeedback(limit int, sinceDate *time.Time, symbol string) ([]*models.SignalFeedback, error) {
	query := `
		SELECT id, symbol, signal, action, confidence, feedback_timestamp, created_at
		FROM signal_feedback
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if sinceDate != nil {
		query += fmt.Sprintf(" AND feedback_timestamp >= $%d", argIdx)
		args = append(args, *sinceDate)
		argIdx++
	}

	if symbol != "" {
		query += fmt.Sprintf(" AND symbol = $%d", argIdx)
		args = append(args, symbol)
		argIdx++
	}

	query += " ORDER BY feedback_timestamp DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get signal feedback: %w", err)
	}
	defer rows.Close()

	var entries []*models.SignalFeedback
	for rows.Next() {
		var fb models.SignalFeedback
		var confidence sql.NullFloat64
		err := rows.Scan(
			&fb.ID, &fb.Symbol, &fb.Signal, &fb.Action,
			&confidence, &fb.FeedbackTimestamp, &fb.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signal feedback: %w", err)
		}
		if confidence.Valid {
			fb.Confidence = confidence.Float64
		}
		entries = append(entries, &fb)
	}

	return entries, nil
}

// GetFeedbackSummary returns aggregate counts by action.
func (db *DB) GetFeedbackSummary() (*models.FeedbackSummary, error) {
	query := `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE action = 'traded') AS traded,
			COUNT(*) FILTER (WHERE action = 'skipped') AS skipped
		FROM signal_feedback
	`
	var summary models.FeedbackSummary
	err := db.conn.QueryRow(query).Scan(&summary.Total, &summary.Traded, &summary.Skipped)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback summary: %w", err)
	}
	return &summary, nil
}
