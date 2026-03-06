package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/trogers1052/stock-alert-system/internal/models"
)

// UpsertBacktestTier inserts or updates a backtest tier record.
func (db *DB) UpsertBacktestTier(tier *models.BacktestTier) error {
	query := `
		INSERT INTO backtest_tiers (
			symbol, tier, composite_score, gates_passed, gates_total,
			regime_pass, allowed_regimes, sharpe, total_return, win_rate,
			profit_factor, max_drawdown, trade_count,
			confidence_multiplier, position_size_multiplier,
			blacklisted, ranking_date, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (symbol) DO UPDATE SET
			tier = EXCLUDED.tier,
			composite_score = EXCLUDED.composite_score,
			gates_passed = EXCLUDED.gates_passed,
			gates_total = EXCLUDED.gates_total,
			regime_pass = EXCLUDED.regime_pass,
			allowed_regimes = EXCLUDED.allowed_regimes,
			sharpe = EXCLUDED.sharpe,
			total_return = EXCLUDED.total_return,
			win_rate = EXCLUDED.win_rate,
			profit_factor = EXCLUDED.profit_factor,
			max_drawdown = EXCLUDED.max_drawdown,
			trade_count = EXCLUDED.trade_count,
			confidence_multiplier = EXCLUDED.confidence_multiplier,
			position_size_multiplier = EXCLUDED.position_size_multiplier,
			blacklisted = EXCLUDED.blacklisted,
			ranking_date = EXCLUDED.ranking_date,
			notes = EXCLUDED.notes,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	_, err := db.conn.Exec(query,
		tier.Symbol, tier.Tier, tier.CompositeScore, tier.GatesPassed, tier.GatesTotal,
		tier.RegimePass, pq.Array(tier.AllowedRegimes), tier.Sharpe, tier.TotalReturn, tier.WinRate,
		tier.ProfitFactor, tier.MaxDrawdown, tier.TradeCount,
		tier.ConfidenceMultiplier, tier.PositionSizeMultiplier,
		tier.Blacklisted, tier.RankingDate, tier.Notes, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert backtest tier: %w", err)
	}
	tier.CreatedAt = now
	tier.UpdatedAt = now
	return nil
}

// GetBacktestTier retrieves a single backtest tier by symbol.
func (db *DB) GetBacktestTier(symbol string) (*models.BacktestTier, error) {
	query := `
		SELECT symbol, tier, composite_score, gates_passed, gates_total,
		       regime_pass, allowed_regimes, sharpe, total_return, win_rate,
		       profit_factor, max_drawdown, trade_count,
		       confidence_multiplier, position_size_multiplier,
		       blacklisted, ranking_date, notes, created_at, updated_at
		FROM backtest_tiers
		WHERE symbol = $1
	`
	t, err := db.scanBacktestTier(db.conn.QueryRow(query, symbol))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get backtest tier: %w", err)
	}
	return t, nil
}

// GetAllBacktestTiers retrieves all backtest tiers ordered by composite score descending.
func (db *DB) GetAllBacktestTiers() ([]*models.BacktestTier, error) {
	query := `
		SELECT symbol, tier, composite_score, gates_passed, gates_total,
		       regime_pass, allowed_regimes, sharpe, total_return, win_rate,
		       profit_factor, max_drawdown, trade_count,
		       confidence_multiplier, position_size_multiplier,
		       blacklisted, ranking_date, notes, created_at, updated_at
		FROM backtest_tiers
		ORDER BY composite_score DESC
	`
	return db.scanBacktestTiers(db.conn.Query(query))
}

// GetBacktestTiersByTier retrieves all backtest tiers with a specific tier grade.
func (db *DB) GetBacktestTiersByTier(tier string) ([]*models.BacktestTier, error) {
	query := `
		SELECT symbol, tier, composite_score, gates_passed, gates_total,
		       regime_pass, allowed_regimes, sharpe, total_return, win_rate,
		       profit_factor, max_drawdown, trade_count,
		       confidence_multiplier, position_size_multiplier,
		       blacklisted, ranking_date, notes, created_at, updated_at
		FROM backtest_tiers
		WHERE tier = $1
		ORDER BY composite_score DESC
	`
	return db.scanBacktestTiers(db.conn.Query(query, tier))
}

func (db *DB) scanBacktestTier(row *sql.Row) (*models.BacktestTier, error) {
	var t models.BacktestTier
	var allowedRegimes []string
	var sharpe, totalReturn, winRate, profitFactor, maxDrawdown sql.NullFloat64
	var notes sql.NullString

	err := row.Scan(
		&t.Symbol, &t.Tier, &t.CompositeScore, &t.GatesPassed, &t.GatesTotal,
		&t.RegimePass, pq.Array(&allowedRegimes), &sharpe, &totalReturn, &winRate,
		&profitFactor, &maxDrawdown, &t.TradeCount,
		&t.ConfidenceMultiplier, &t.PositionSizeMultiplier,
		&t.Blacklisted, &t.RankingDate, &notes, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.AllowedRegimes = allowedRegimes
	if sharpe.Valid {
		t.Sharpe = &sharpe.Float64
	}
	if totalReturn.Valid {
		t.TotalReturn = &totalReturn.Float64
	}
	if winRate.Valid {
		t.WinRate = &winRate.Float64
	}
	if profitFactor.Valid {
		t.ProfitFactor = &profitFactor.Float64
	}
	if maxDrawdown.Valid {
		t.MaxDrawdown = &maxDrawdown.Float64
	}
	if notes.Valid {
		t.Notes = notes.String
	}

	return &t, nil
}

func (db *DB) scanBacktestTiers(rows *sql.Rows, err error) ([]*models.BacktestTier, error) {
	if err != nil {
		return nil, fmt.Errorf("failed to query backtest tiers: %w", err)
	}
	defer rows.Close()

	var tiers []*models.BacktestTier
	for rows.Next() {
		var t models.BacktestTier
		var allowedRegimes []string
		var sharpe, totalReturn, winRate, profitFactor, maxDrawdown sql.NullFloat64
		var notes sql.NullString

		err := rows.Scan(
			&t.Symbol, &t.Tier, &t.CompositeScore, &t.GatesPassed, &t.GatesTotal,
			&t.RegimePass, pq.Array(&allowedRegimes), &sharpe, &totalReturn, &winRate,
			&profitFactor, &maxDrawdown, &t.TradeCount,
			&t.ConfidenceMultiplier, &t.PositionSizeMultiplier,
			&t.Blacklisted, &t.RankingDate, &notes, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backtest tier: %w", err)
		}

		t.AllowedRegimes = allowedRegimes
		if sharpe.Valid {
			t.Sharpe = &sharpe.Float64
		}
		if totalReturn.Valid {
			t.TotalReturn = &totalReturn.Float64
		}
		if winRate.Valid {
			t.WinRate = &winRate.Float64
		}
		if profitFactor.Valid {
			t.ProfitFactor = &profitFactor.Float64
		}
		if maxDrawdown.Valid {
			t.MaxDrawdown = &maxDrawdown.Float64
		}
		if notes.Valid {
			t.Notes = notes.String
		}

		tiers = append(tiers, &t)
	}

	return tiers, nil
}
