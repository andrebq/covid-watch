package webapp

import "github.com/prometheus/client_golang/prometheus"

var (
	renderErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "render_error",
		Help: "How many render errors on a given page",
	}, []string{"page"})
)

func initMetrics() {
	prometheus.MustRegister(renderErrors)
}
