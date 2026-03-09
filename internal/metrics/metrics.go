// Package metrics defines application-level Prometheus metrics for the
// stock-service. All variables are package-level and registered via promauto
// so they are automatically available at the /metrics endpoint.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP metrics -----------------------------------------------------------

var HTTPRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "stock_http_requests_total",
		Help: "Total number of HTTP requests by method, endpoint, and status code.",
	},
	[]string{"method", "endpoint", "status_code"},
)

var HTTPRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "stock_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "endpoint"},
)

// Kafka consumer metrics -------------------------------------------------

var KafkaConsumed = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "stock_kafka_consumed_total",
		Help: "Total Kafka messages consumed by topic.",
	},
	[]string{"topic"},
)

var KafkaConsumerErrors = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "stock_kafka_consumer_errors_total",
		Help: "Total Kafka consumer errors by topic.",
	},
	[]string{"topic"},
)

// Business-logic metrics -------------------------------------------------

var FeedbackCreated = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "stock_feedback_created_total",
		Help: "Total signal feedback entries created.",
	},
)

var FeedbackUpdated = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "stock_feedback_updated_total",
		Help: "Total signal feedback entries updated.",
	},
)

var TierUpserts = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "stock_tier_upserts_total",
		Help: "Total tier ranking upserts.",
	},
)

// Database write metrics -------------------------------------------------

var DBWriteDuration = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "stock_db_write_duration_seconds",
		Help:    "PostgreSQL write latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
)

var DBWriteErrors = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "stock_db_write_errors_total",
		Help: "Total PostgreSQL write failures.",
	},
)

// Accuracy cache metrics -------------------------------------------------

var AccuracyCacheDuration = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "stock_accuracy_cache_duration_seconds",
		Help:    "Accuracy cache write latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
)

var AccuracyCacheErrors = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "stock_accuracy_cache_errors_total",
		Help: "Total accuracy cache write failures.",
	},
)

// Watchlist metrics ------------------------------------------------------

var WatchlistEvents = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "stock_watchlist_events_total",
		Help: "Total watchlist Kafka events published.",
	},
)
