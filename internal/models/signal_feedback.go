package models

import "time"

// SignalFeedback represents a user's feedback on a trading signal.
type SignalFeedback struct {
	ID                int       `json:"id"`
	Symbol            string    `json:"symbol"`
	Signal            string    `json:"signal"`
	Action            string    `json:"action"`
	Confidence        float64   `json:"confidence,omitempty"`
	FeedbackTimestamp time.Time `json:"feedback_timestamp"`
	CreatedAt         time.Time `json:"created_at"`
}

// FeedbackSummary holds aggregate counts of feedback actions.
type FeedbackSummary struct {
	Total   int `json:"total"`
	Traded  int `json:"traded"`
	Skipped int `json:"skipped"`
}
