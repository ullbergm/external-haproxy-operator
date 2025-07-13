package monitoring

// "github.com/prometheus/client_golang/prometheus"
// "sigs.k8s.io/controller-runtime/pkg/metrics"

// BackendBackendsGauge is a Prometheus gauge metric to track the number of Backend instances.
var (
// BackendBackendsGauge = prometheus.NewGauge(
//
//	prometheus.GaugeOpts{
//		Name: "backend_backends_count",
//		Help: "Current number of backend instances.",
//	},
//
// )
)

// RegisterMetrics registers the BackendBackendsGauge metric with the controller-runtime metrics registry.
func RegisterMetrics() {
	// metrics.Registry.MustRegister(BackendBackendsGauge)
}
