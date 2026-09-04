package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by status code, method, and path.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of HTTP request latencies in seconds.",
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1.0},
		},
		[]string{"method", "path"},
	)

	CacheOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache read operations, partitioned by level (l1, l2) and result (hit, miss).",
		},
		[]string{"level", "status"},
	)

	KafkaOrdersProducedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_orders_produced_total",
			Help: "Total number of order events published to Kafka.",
		},
	)

	KafkaOrdersConsumedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_orders_consumed_total",
			Help: "Total number of order events consumed and batch-persisted from Kafka.",
		},
	)

	DBPoolAcquiredConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_acquired_connections",
			Help: "Number of currently acquired connections from the PostgreSQL pool.",
		},
	)

	DBPoolIdleConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Number of idle connections currently in the PostgreSQL pool.",
		},
	)
)
