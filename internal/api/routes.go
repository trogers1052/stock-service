package api

import (
	"github.com/gorilla/mux"
)

// SetupRoutes configures all API routes
func SetupRoutes(handler *Handler, apiKey string) *mux.Router {
	r := mux.NewRouter()

	// Global Prometheus metrics middleware — must be registered on the
	// top-level router so it captures every request including /health.
	r.Use(PrometheusMiddleware)

	// Health check (unauthenticated)
	r.HandleFunc("/health", handler.HealthCheck).Methods("GET")

	// API routes (authenticated when API_KEY is set)
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(APIKeyAuth(apiKey))

	// Stock routes
	api.HandleFunc("/stocks", handler.GetAllStocks).Methods("GET")
	api.HandleFunc("/stocks", handler.AddStock).Methods("POST")
	api.HandleFunc("/stocks/sectors", handler.GetSectors).Methods("GET")
	api.HandleFunc("/stocks/{symbol}", handler.GetStock).Methods("GET")
	api.HandleFunc("/stocks/{symbol}", handler.RemoveStock).Methods("DELETE")

	// Signal feedback routes
	api.HandleFunc("/feedback", handler.CreateFeedback).Methods("POST")
	api.HandleFunc("/feedback", handler.GetFeedback).Methods("GET")
	api.HandleFunc("/feedback/summary", handler.GetFeedbackSummary).Methods("GET")
	api.HandleFunc("/feedback/accuracy", handler.GetRuleAccuracy).Methods("GET")
	api.HandleFunc("/feedback/unresolved", handler.GetUnresolvedSignals).Methods("GET")
	api.HandleFunc("/feedback/outcome-quality", handler.GetRuleOutcomeQuality).Methods("GET")
	api.HandleFunc("/feedback/{id}", handler.UpdateFeedback).Methods("PUT")
	api.HandleFunc("/feedback/{id}/outcome", handler.UpdateSignalOutcome).Methods("PUT")

	// Tier ranking routes
	api.HandleFunc("/tiers", handler.GetAllTiers).Methods("GET")
	api.HandleFunc("/tiers/bulk", handler.BulkUpsertTiers).Methods("PUT")
	api.HandleFunc("/tiers/{symbol}", handler.GetTier).Methods("GET")
	api.HandleFunc("/tiers", handler.UpsertTier).Methods("PUT")

	return r
}
