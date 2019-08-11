package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "apitest",
			Subsystem: "requests",
			Name:      "total",
			Help:      "The total number of processed requests",
		},
		[]string{"name", "hostname", "path", "method"})
	requestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "apitest",
			Subsystem: "requests",
			Name:      "errors_total",
			Help:      "The total number of requests that had at least one assertion error",
		},
		[]string{"name", "hostname", "path", "method"})
	requestDurations = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "apitest",
			Subsystem: "requests",
			Name:      "duration",
			Help:      "The duration for requests made, in seconds",
		},
		[]string{"name", "hostname", "path", "method"})
)

// recordError records an error.
// only one error (or none) should be recorded per request (e.g. even if there are
// multiple assertion errors)
func recordError(name string, hostname string, path string, method string) {
	requestErrors.WithLabelValues(name, hostname, path, method).Inc()
}

// recordRequest records a request made.
func recordRequest(name string, hostname string, path string, method string) {
	requestsProcessed.WithLabelValues(name, hostname, path, method).Inc()
}

// recordDuration records a duration in ms.
func recordDuration(name string, hostname string, path string, method string, duration float64) {
	requestDurations.WithLabelValues(name, hostname, path, method).Observe(duration)
}

// NewMetricsHandler returns the metrics http endpoint
func NewMetricsHandler() http.Handler {
	return promhttp.Handler()
}
