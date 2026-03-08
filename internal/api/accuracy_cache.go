package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	accuracyCacheKey      = "feedback:accuracy"
	accuracyCacheInterval = 15 * time.Minute
	accuracyCacheTTL      = 30 * time.Minute
	accuracySinceDays     = 90
	accuracyMinSignals    = 10
)

// accuracyEntry is the per-rule data written to Redis.
type accuracyEntry struct {
	TradeRate   float64 `json:"trade_rate"`
	Multiplier  float64 `json:"multiplier"`
	SignalCount int     `json:"signal_count"`
}

// StartAccuracyCacheWriter runs a background goroutine that periodically computes
// per-rule accuracy metrics and writes them to Redis for the decision-engine to consume.
func (h *Handler) StartAccuracyCacheWriter(ctx context.Context) {
	if h.redis == nil {
		log.Println("Accuracy cache writer: Redis unavailable, skipping")
		return
	}

	// Write immediately on startup
	h.writeAccuracyCache(ctx)

	ticker := time.NewTicker(accuracyCacheInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.writeAccuracyCache(ctx)
		case <-ctx.Done():
			log.Println("Accuracy cache writer stopped")
			return
		}
	}
}

func (h *Handler) writeAccuracyCache(ctx context.Context) {
	accuracy, err := h.db.GetRuleAccuracy(accuracySinceDays, accuracyMinSignals)
	if err != nil {
		log.Printf("WARNING: failed to compute rule accuracy: %v", err)
		return
	}

	// Build map: "rule_name:regime_id" -> { trade_rate, multiplier, signal_count }
	data := make(map[string]accuracyEntry, len(accuracy))
	for _, a := range accuracy {
		key := fmt.Sprintf("%s:%s", a.RuleName, a.RegimeID)
		data[key] = accuracyEntry{
			TradeRate:   a.TradeRate,
			Multiplier:  a.Multiplier,
			SignalCount: a.SignalCount,
		}
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("WARNING: failed to marshal accuracy cache: %v", err)
		return
	}

	if err := h.redis.Set(ctx, accuracyCacheKey, string(jsonBytes), accuracyCacheTTL); err != nil {
		log.Printf("WARNING: failed to write accuracy cache to Redis: %v", err)
		return
	}

	log.Printf("Accuracy cache updated: %d rule-regime entries", len(data))
}
