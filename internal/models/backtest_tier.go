package models

import "time"

// BacktestTier represents tier ranking metadata from backtesting validation.
type BacktestTier struct {
	Symbol                 string    `json:"symbol"`
	Tier                   string    `json:"tier"`
	CompositeScore         float64   `json:"composite_score"`
	GatesPassed            int       `json:"gates_passed"`
	GatesTotal             int       `json:"gates_total"`
	RegimePass             bool      `json:"regime_pass"`
	AllowedRegimes         []string  `json:"allowed_regimes,omitempty"`
	Sharpe                 *float64  `json:"sharpe,omitempty"`
	TotalReturn            *float64  `json:"total_return,omitempty"`
	WinRate                *float64  `json:"win_rate,omitempty"`
	ProfitFactor           *float64  `json:"profit_factor,omitempty"`
	MaxDrawdown            *float64  `json:"max_drawdown,omitempty"`
	TradeCount             int       `json:"trade_count"`
	ConfidenceMultiplier   float64   `json:"confidence_multiplier"`
	PositionSizeMultiplier float64   `json:"position_size_multiplier"`
	Blacklisted            bool      `json:"blacklisted"`
	RankingDate            time.Time `json:"ranking_date"`
	Notes                  string    `json:"notes,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
