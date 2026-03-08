package database

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/lib/pq"
	"github.com/trogers1052/stock-alert-system/internal/models"
)

// CreateSignalFeedback inserts a new feedback entry.
func (db *DB) CreateSignalFeedback(fb *models.SignalFeedback) error {
	query := `
		INSERT INTO signal_feedback (
			symbol, signal, action, confidence,
			rules_triggered, regime_id, decision_confidence,
			feedback_timestamp, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	now := time.Now()

	// Use pq.Array for TEXT[] column; pass nil if empty
	var rulesArr interface{}
	if len(fb.RulesTriggered) > 0 {
		rulesArr = pq.Array(fb.RulesTriggered)
	}

	// Use sql.NullString for nullable regime_id
	var regimeID interface{}
	if fb.RegimeID != "" {
		regimeID = fb.RegimeID
	}

	// Use sql.NullFloat64 for nullable decision_confidence
	var decConf interface{}
	if fb.DecisionConfidence > 0 {
		decConf = fb.DecisionConfidence
	}

	err := db.conn.QueryRow(query,
		fb.Symbol, fb.Signal, fb.Action, fb.Confidence,
		rulesArr, regimeID, decConf,
		fb.FeedbackTimestamp, now,
	).Scan(&fb.ID, &fb.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create signal feedback: %w", err)
	}
	return nil
}

// UpdateFeedbackAction updates the action field of an existing feedback row.
func (db *DB) UpdateFeedbackAction(id int, action string) error {
	query := `UPDATE signal_feedback SET action = $1 WHERE id = $2`
	result, err := db.conn.Exec(query, action, id)
	if err != nil {
		return fmt.Errorf("failed to update feedback action: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("feedback id %d not found", id)
	}
	return nil
}

// GetSignalFeedback returns feedback entries with optional filters.
func (db *DB) GetSignalFeedback(limit int, sinceDate *time.Time, symbol string) ([]*models.SignalFeedback, error) {
	query := `
		SELECT id, symbol, signal, action, confidence,
			   rules_triggered, regime_id, decision_confidence,
			   feedback_timestamp, created_at
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
		var rulesTriggered pq.StringArray
		var regimeID sql.NullString
		var decisionConfidence sql.NullFloat64
		err := rows.Scan(
			&fb.ID, &fb.Symbol, &fb.Signal, &fb.Action,
			&confidence, &rulesTriggered, &regimeID, &decisionConfidence,
			&fb.FeedbackTimestamp, &fb.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signal feedback: %w", err)
		}
		if confidence.Valid {
			fb.Confidence = confidence.Float64
		}
		fb.RulesTriggered = rulesTriggered
		if regimeID.Valid {
			fb.RegimeID = regimeID.String
		}
		if decisionConfidence.Valid {
			fb.DecisionConfidence = decisionConfidence.Float64
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

// GetRuleAccuracy computes per-rule, per-regime accuracy metrics from enriched feedback.
func (db *DB) GetRuleAccuracy(sinceDays int, minSignals int) ([]*models.RuleAccuracy, error) {
	query := `
		SELECT
			rule_name,
			COALESCE(regime_id, 'ALL') AS regime_id,
			COUNT(*) AS signal_count,
			COUNT(*) FILTER (WHERE action = 'traded') AS traded_count,
			COUNT(*) FILTER (WHERE action = 'skipped') AS skipped_count
		FROM signal_feedback,
			 LATERAL unnest(rules_triggered) AS rule_name
		WHERE rules_triggered IS NOT NULL
		  AND feedback_timestamp >= NOW() - ($1 || ' days')::INTERVAL
		GROUP BY rule_name, regime_id
		HAVING COUNT(*) >= $2
		ORDER BY rule_name, regime_id
	`

	rows, err := db.conn.Query(query, fmt.Sprintf("%d", sinceDays), minSignals)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule accuracy: %w", err)
	}
	defer rows.Close()

	var results []*models.RuleAccuracy
	for rows.Next() {
		var ra models.RuleAccuracy
		err := rows.Scan(
			&ra.RuleName, &ra.RegimeID,
			&ra.SignalCount, &ra.TradedCount, &ra.SkippedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule accuracy: %w", err)
		}
		if ra.SignalCount > 0 {
			ra.TradeRate = float64(ra.TradedCount) / float64(ra.SignalCount)
		}
		// Multiplier: clamp(0.5, 1.5, 0.5 + trade_rate)
		ra.Multiplier = math.Max(0.5, math.Min(1.5, 0.5+ra.TradeRate))
		results = append(results, &ra)
	}

	return results, nil
}
