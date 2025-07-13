package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// HAProxyClientErrorCountTotal is a Prometheus counter metric to track the number of errors from the HAProxy client.
var (
	HAProxyClientErrorCountTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "haproxy_client_errors_count_total",
			Help: "Total number of errors from the HAProxy client.",
		},
	)
)

// RegisterMetrics registers the HAProxyClientErrorCountTotal metric with the controller-runtime metrics registry.
func RegisterMetrics() {
	metrics.Registry.MustRegister(HAProxyClientErrorCountTotal)
}
