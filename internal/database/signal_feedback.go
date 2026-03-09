package database

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/lib/pq"
	"github.com/trogers1052/stock-alert-system/internal/models"
)

// CreateSignalFeedback inserts a new feedback entry (upsert on duplicate).
func (db *DB) CreateSignalFeedback(fb *models.SignalFeedback) error {
	query := `
		INSERT INTO signal_feedback (
			symbol, signal, action, confidence,
			rules_triggered, regime_id, decision_confidence,
			entry_price, stop_price, target_1, target_2, valid_until,
			feedback_timestamp, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (symbol, signal, feedback_timestamp) DO UPDATE SET
			action = EXCLUDED.action,
			confidence = EXCLUDED.confidence,
			rules_triggered = EXCLUDED.rules_triggered,
			regime_id = EXCLUDED.regime_id,
			decision_confidence = EXCLUDED.decision_confidence,
			entry_price = EXCLUDED.entry_price,
			stop_price = EXCLUDED.stop_price,
			target_1 = EXCLUDED.target_1,
			target_2 = EXCLUDED.target_2,
			valid_until = EXCLUDED.valid_until
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

	// Nullable price fields (zero = not provided)
	var entryPrice, stopPrice, target1, target2 interface{}
	if fb.EntryPrice > 0 {
		entryPrice = fb.EntryPrice
	}
	if fb.StopPrice > 0 {
		stopPrice = fb.StopPrice
	}
	if fb.Target1 > 0 {
		target1 = fb.Target1
	}
	if fb.Target2 > 0 {
		target2 = fb.Target2
	}

	err := db.conn.QueryRow(query,
		fb.Symbol, fb.Signal, fb.Action, fb.Confidence,
		rulesArr, regimeID, decConf,
		entryPrice, stopPrice, target1, target2, fb.ValidUntil,
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
			   entry_price, stop_price, target_1, target_2, valid_until,
			   outcome, outcome_at,
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
		var entryPrice, stopPrice, target1, target2 sql.NullFloat64
		var validUntil, outcomeAt sql.NullTime
		var outcome sql.NullString
		err := rows.Scan(
			&fb.ID, &fb.Symbol, &fb.Signal, &fb.Action,
			&confidence, &rulesTriggered, &regimeID, &decisionConfidence,
			&entryPrice, &stopPrice, &target1, &target2, &validUntil,
			&outcome, &outcomeAt,
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
		if entryPrice.Valid {
			fb.EntryPrice = entryPrice.Float64
		}
		if stopPrice.Valid {
			fb.StopPrice = stopPrice.Float64
		}
		if target1.Valid {
			fb.Target1 = target1.Float64
		}
		if target2.Valid {
			fb.Target2 = target2.Float64
		}
		if validUntil.Valid {
			fb.ValidUntil = &validUntil.Time
		}
		if outcome.Valid {
			fb.Outcome = outcome.String
		}
		if outcomeAt.Valid {
			fb.OutcomeAt = &outcomeAt.Time
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

// GetUnresolvedSignals returns feedback entries that have trade plan data but no outcome yet.
func (db *DB) GetUnresolvedSignals(limit int) ([]*models.SignalFeedback, error) {
	query := `
		SELECT id, symbol, signal, action, confidence,
			   rules_triggered, regime_id, decision_confidence,
			   entry_price, stop_price, target_1, target_2, valid_until,
			   outcome, outcome_at,
			   feedback_timestamp, created_at
		FROM signal_feedback
		WHERE outcome IS NULL
		  AND entry_price IS NOT NULL
		  AND stop_price IS NOT NULL
		ORDER BY feedback_timestamp DESC
		LIMIT $1
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get unresolved signals: %w", err)
	}
	defer rows.Close()

	var entries []*models.SignalFeedback
	for rows.Next() {
		var fb models.SignalFeedback
		var confidence sql.NullFloat64
		var rulesTriggered pq.StringArray
		var regimeID sql.NullString
		var decisionConfidence sql.NullFloat64
		var entryPrice, stopPrice, target1, target2 sql.NullFloat64
		var validUntil, outcomeAt sql.NullTime
		var outcome sql.NullString
		err := rows.Scan(
			&fb.ID, &fb.Symbol, &fb.Signal, &fb.Action,
			&confidence, &rulesTriggered, &regimeID, &decisionConfidence,
			&entryPrice, &stopPrice, &target1, &target2, &validUntil,
			&outcome, &outcomeAt,
			&fb.FeedbackTimestamp, &fb.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unresolved signal: %w", err)
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
		if entryPrice.Valid {
			fb.EntryPrice = entryPrice.Float64
		}
		if stopPrice.Valid {
			fb.StopPrice = stopPrice.Float64
		}
		if target1.Valid {
			fb.Target1 = target1.Float64
		}
		if target2.Valid {
			fb.Target2 = target2.Float64
		}
		if validUntil.Valid {
			fb.ValidUntil = &validUntil.Time
		}
		if outcome.Valid {
			fb.Outcome = outcome.String
		}
		if outcomeAt.Valid {
			fb.OutcomeAt = &outcomeAt.Time
		}
		entries = append(entries, &fb)
	}

	return entries, nil
}

// UpdateSignalOutcome sets the outcome and outcome_at for a feedback entry.
func (db *DB) UpdateSignalOutcome(id int, outcome string) error {
	query := `UPDATE signal_feedback SET outcome = $1, outcome_at = NOW() WHERE id = $2`
	result, err := db.conn.Exec(query, outcome, id)
	if err != nil {
		return fmt.Errorf("failed to update signal outcome: %w", err)
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

// GetRuleOutcomeQuality computes per-rule, per-regime signal quality metrics
// based on price outcomes. Win = TARGET_1_HIT or TARGET_2_HIT. Loss = STOPPED_OUT.
func (db *DB) GetRuleOutcomeQuality(sinceDays int, minSignals int) ([]*models.RuleOutcomeQuality, error) {
	query := `
		SELECT
			rule_name,
			COALESCE(regime_id, 'ALL') AS regime_id,
			COUNT(*) AS signal_count,
			COUNT(*) FILTER (WHERE outcome IN ('TARGET_1_HIT', 'TARGET_2_HIT')) AS win_count,
			COUNT(*) FILTER (WHERE outcome = 'STOPPED_OUT') AS loss_count,
			COUNT(*) FILTER (WHERE outcome = 'EXPIRED') AS expired_count
		FROM signal_feedback,
			 LATERAL unnest(rules_triggered) AS rule_name
		WHERE rules_triggered IS NOT NULL
		  AND outcome IS NOT NULL
		  AND feedback_timestamp >= NOW() - ($1 || ' days')::INTERVAL
		GROUP BY rule_name, regime_id
		HAVING COUNT(*) >= $2
		ORDER BY rule_name, regime_id
	`

	rows, err := db.conn.Query(query, fmt.Sprintf("%d", sinceDays), minSignals)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule outcome quality: %w", err)
	}
	defer rows.Close()

	var results []*models.RuleOutcomeQuality
	for rows.Next() {
		var rq models.RuleOutcomeQuality
		err := rows.Scan(
			&rq.RuleName, &rq.RegimeID,
			&rq.SignalCount, &rq.WinCount, &rq.LossCount, &rq.ExpiredCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule outcome quality: %w", err)
		}
		// Win rate = wins / (wins + losses). Expired signals are excluded from the ratio.
		decided := rq.WinCount + rq.LossCount
		if decided > 0 {
			rq.WinRate = float64(rq.WinCount) / float64(decided)
		}
		// Multiplier: clamp(0.5, 1.5, win_rate * 1.5)
		// 0% win rate -> 0.5x, 50% -> 0.75x, 67% -> 1.0x, 100% -> 1.5x
		rq.Multiplier = math.Max(0.5, math.Min(1.5, rq.WinRate*1.5))
		results = append(results, &rq)
	}

	return results, nil
}
