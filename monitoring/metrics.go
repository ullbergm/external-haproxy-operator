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

// MetricDescription is an exported struct that defines the metric description (Name, Help)
// as a new type named MetricDescription.
type MetricDescription struct {
	Name string
	Help string
	Type string
}

// metricsDescription is a map of string keys (metrics) to MetricDescription values (Name, Help).
var metricDescription = map[string]MetricDescription{
	"HAProxyClientErrorCountTotal": {
		Name: "haproxy_client_errors_count_total",
		Help: "Total number of errors from the HAProxy client.",
		Type: "Counter",
	},
}

// ListMetrics will create a slice with the metrics available in metricDescription
func ListMetrics() []MetricDescription {
	v := make([]MetricDescription, 0, len(metricDescription))
	// Insert value (Name, Help) for each metric
	for _, value := range metricDescription {
		v = append(v, value)
	}

	return v
}
