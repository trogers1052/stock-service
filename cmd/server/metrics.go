package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	// Side-effect import: registers all promauto metrics defined in the
	// metrics package so they appear on the /metrics endpoint.
	_ "github.com/trogers1052/stock-alert-system/internal/metrics"
)

func startMetricsServer() {
	port := os.Getenv("METRICS_PORT")
	if port == "" {
		port = "9097"
	}
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":"+port, metricsMux); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()
	log.Printf("Metrics server listening on :%s/metrics", port)
}
