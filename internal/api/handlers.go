package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/trogers1052/stock-alert-system/internal/database"
	"github.com/trogers1052/stock-alert-system/internal/kafka"
	"github.com/trogers1052/stock-alert-system/internal/models"
	"github.com/trogers1052/stock-alert-system/internal/redis"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	db       *database.DB
	producer *kafka.Producer
	redis    *redis.Client
}

// NewHandler creates a new Handler
func NewHandler(db *database.DB, producer *kafka.Producer, redisClient *redis.Client) *Handler {
	return &Handler{
		db:       db,
		producer: producer,
		redis:    redisClient,
	}
}

// GetAllStocks handles GET /stocks
func (h *Handler) GetAllStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.db.GetAllStocks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stocks)
}

// GetStock handles GET /stocks/{symbol}
func (h *Handler) GetStock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	stock, err := h.db.GetStock(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, stock)
}

// AddStock handles POST /stocks
func (h *Handler) AddStock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbol string `json:"symbol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	monitoredStock := &models.MonitoredStock{
		Symbol:  req.Symbol,
		Enabled: true,
	}
	if err := h.db.CreateMonitoredStock(monitoredStock); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the stock to return and publish event
	stock, err := h.db.GetStock(req.Symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish Kafka event
	if h.producer != nil {
		if err := h.producer.PublishStockAdded(r.Context(), stock); err != nil {
			// Log error but don't fail the request
			// In production, you'd use a proper logger here
		}
	}

	respondJSON(w, http.StatusCreated, stock)
}

// RemoveStock handles DELETE /stocks/{symbol}
func (h *Handler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	if err := h.db.DeleteMonitoredStock(symbol); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish Kafka event
	if h.producer != nil {
		if err := h.producer.PublishStockRemoved(r.Context(), symbol); err != nil {
			// Log error but don't fail the request
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSectors handles GET /api/v1/stocks/sectors
func (h *Handler) GetSectors(w http.ResponseWriter, r *http.Request) {
	sectorMap, err := h.db.GetSectorMap()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, sectorMap)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services":  map[string]string{},
	}
	services := health["services"].(map[string]string)
	allHealthy := true

	// Check database
	if h.db != nil {
		if err := h.db.Ping(); err != nil {
			services["postgres"] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			services["postgres"] = "healthy"
		}
	} else {
		services["postgres"] = "not configured"
		allHealthy = false
	}

	// Check Redis
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			services["redis"] = "unhealthy: " + err.Error()
		} else {
			services["redis"] = "healthy"
		}
	} else {
		services["redis"] = "not configured"
	}

	// Check Kafka producer
	if h.producer != nil {
		services["kafka"] = "configured"
	} else {
		services["kafka"] = "not configured"
	}

	if !allHealthy {
		health["status"] = "degraded"
	}

	respondJSON(w, http.StatusOK, health)
}

// CreateFeedback handles POST /api/v1/feedback
func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbol     string  `json:"symbol"`
		Signal     string  `json:"signal"`
		Action     string  `json:"action"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Symbol == "" || req.Signal == "" || req.Action == "" {
		http.Error(w, "symbol, signal, and action are required", http.StatusBadRequest)
		return
	}

	fb := &models.SignalFeedback{
		Symbol:            req.Symbol,
		Signal:            req.Signal,
		Action:            req.Action,
		Confidence:        req.Confidence,
		FeedbackTimestamp: time.Now(),
	}

	if err := h.db.CreateSignalFeedback(fb); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, fb)
}

// GetFeedback handles GET /api/v1/feedback
func (h *Handler) GetFeedback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var sinceDate *time.Time
	if sinceDays := query.Get("since_days"); sinceDays != "" {
		days, err := strconv.Atoi(sinceDays)
		if err != nil || days < 0 {
			http.Error(w, "invalid since_days parameter", http.StatusBadRequest)
			return
		}
		t := time.Now().AddDate(0, 0, -days)
		sinceDate = &t
	}

	symbol := query.Get("symbol")

	limit := 1000
	if limitStr := query.Get("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 0 {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = l
	}

	entries, err := h.db.GetSignalFeedback(limit, sinceDate, symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, entries)
}

// GetFeedbackSummary handles GET /api/v1/feedback/summary
func (h *Handler) GetFeedbackSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.db.GetFeedbackSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// GetAllTiers handles GET /api/v1/tiers
func (h *Handler) GetAllTiers(w http.ResponseWriter, r *http.Request) {
	tierFilter := r.URL.Query().Get("tier")

	var tiers []*models.BacktestTier
	var err error

	if tierFilter != "" {
		tiers, err = h.db.GetBacktestTiersByTier(tierFilter)
	} else {
		tiers, err = h.db.GetAllBacktestTiers()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, tiers)
}

// GetTier handles GET /api/v1/tiers/{symbol}
func (h *Handler) GetTier(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	tier, err := h.db.GetBacktestTier(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tier == nil {
		http.Error(w, "tier not found: "+symbol, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, tier)
}

// UpsertTier handles PUT /api/v1/tiers
func (h *Handler) UpsertTier(w http.ResponseWriter, r *http.Request) {
	var tier models.BacktestTier
	if err := json.NewDecoder(r.Body).Decode(&tier); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if tier.Symbol == "" || tier.Tier == "" {
		http.Error(w, "symbol and tier are required", http.StatusBadRequest)
		return
	}

	if err := h.db.UpsertBacktestTier(&tier); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache in Redis
	if h.redis != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		td := &redis.TierData{
			Symbol:                 tier.Symbol,
			Tier:                   tier.Tier,
			CompositeScore:         tier.CompositeScore,
			ConfidenceMultiplier:   tier.ConfidenceMultiplier,
			PositionSizeMultiplier: tier.PositionSizeMultiplier,
			Blacklisted:            tier.Blacklisted,
			AllowedRegimes:         tier.AllowedRegimes,
		}
		_ = h.redis.SetTierData(ctx, td, 24*time.Hour)
	}

	respondJSON(w, http.StatusOK, tier)
}

// BulkUpsertTiers handles PUT /api/v1/tiers/bulk
func (h *Handler) BulkUpsertTiers(w http.ResponseWriter, r *http.Request) {
	var tiers []models.BacktestTier
	if err := json.NewDecoder(r.Body).Decode(&tiers); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(tiers) == 0 {
		http.Error(w, "at least one tier is required", http.StatusBadRequest)
		return
	}

	var errors []string
	succeeded := 0
	for i := range tiers {
		if tiers[i].Symbol == "" || tiers[i].Tier == "" {
			errors = append(errors, "entry missing symbol or tier")
			continue
		}

		if err := h.db.UpsertBacktestTier(&tiers[i]); err != nil {
			errors = append(errors, tiers[i].Symbol+": "+err.Error())
			continue
		}

		// Cache in Redis
		if h.redis != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			td := &redis.TierData{
				Symbol:                 tiers[i].Symbol,
				Tier:                   tiers[i].Tier,
				CompositeScore:         tiers[i].CompositeScore,
				ConfidenceMultiplier:   tiers[i].ConfidenceMultiplier,
				PositionSizeMultiplier: tiers[i].PositionSizeMultiplier,
				Blacklisted:            tiers[i].Blacklisted,
				AllowedRegimes:         tiers[i].AllowedRegimes,
			}
			_ = h.redis.SetTierData(ctx, td, 24*time.Hour)
			cancel()
		}

		succeeded++
	}

	result := map[string]interface{}{
		"succeeded": succeeded,
		"total":     len(tiers),
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}

	respondJSON(w, http.StatusOK, result)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
