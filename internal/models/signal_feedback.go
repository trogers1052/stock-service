package models

import "time"

// SignalFeedback represents a user's feedback on a trading signal.
type SignalFeedback struct {
	ID                 int       `json:"id"`
	Symbol             string    `json:"symbol"`
	Signal             string    `json:"signal"`
	Action             string    `json:"action"`
	Confidence         float64   `json:"confidence,omitempty"`
	RulesTriggered     []string  `json:"rules_triggered,omitempty"`
	RegimeID           string    `json:"regime_id,omitempty"`
	DecisionConfidence float64   `json:"decision_confidence,omitempty"`
	FeedbackTimestamp  time.Time `json:"feedback_timestamp"`
	CreatedAt          time.Time `json:"created_at"`
}

// FeedbackSummary holds aggregate counts of feedback actions.
type FeedbackSummary struct {
	Total   int `json:"total"`
	Traded  int `json:"traded"`
	Skipped int `json:"skipped"`
}

// RuleAccuracy holds per-rule, per-regime accuracy metrics computed from signal_feedback.
type RuleAccuracy struct {
	RuleName     string  `json:"rule_name"`
	RegimeID     string  `json:"regime_id"`
	SignalCount  int     `json:"signal_count"`
	TradedCount  int     `json:"traded_count"`
	SkippedCount int     `json:"skipped_count"`
	TradeRate    float64 `json:"trade_rate"`
	Multiplier   float64 `json:"multiplier"`
}
