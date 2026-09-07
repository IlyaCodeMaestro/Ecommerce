package metrics

import (
	"testing"
)

func TestMetricsRegistration(t *testing.T) {
	// Verify that custom Prometheus metrics can be incremented without panic
	HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/products", "200").Inc()
	HTTPRequestDuration.WithLabelValues("GET", "/api/v1/products").Observe(0.012)
	CacheOperationsTotal.WithLabelValues("l1", "hit").Inc()
	KafkaOrdersProducedTotal.Inc()
	KafkaOrdersConsumedTotal.Inc()
	DBPoolAcquiredConns.Set(5)
	DBPoolIdleConns.Set(15)
	AuthAttemptsTotal.WithLabelValues("login", "success").Inc()
	AuthAttemptsTotal.WithLabelValues("login", "failure").Inc()
	CartOperationsTotal.WithLabelValues("add").Inc()
	CartOperationsTotal.WithLabelValues("merge").Inc()
}
