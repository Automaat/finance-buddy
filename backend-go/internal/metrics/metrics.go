// Package metrics holds the app's Prometheus collectors and the /metrics
// handler. Collectors live on a private registry (not the global default) so
// tests and multiple New() calls don't double-register, and so the exposed
// series are exactly the ones we define here.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()

	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fb_http_requests_total",
		Help: "Total HTTP requests by method, route pattern and status code.",
	}, []string{"method", "route", "status"})

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "fb_http_request_duration_seconds",
		Help: "HTTP request latency by method and route pattern.",
		// Default buckets top out at 10s; extend so the legitimately-slow
		// refresh endpoints (quotes refresh self-bounds at 2 min) land in real
		// buckets instead of all collapsing into +Inf.
		Buckets: append(prometheus.DefBuckets, 30, 60, 120),
	}, []string{"method", "route"})

	schedulerRuns = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fb_scheduler_runs_total",
		Help: "Scheduler job executions by job name and result (success|error).",
	}, []string{"scheduler", "result"})
)

func init() {
	registry.MustRegister(
		httpRequests,
		httpDuration,
		schedulerRuns,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

// Handler serves the Prometheus exposition format for the app's registry.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// ObserveRequest records one served request. route is the chi route pattern
// (not the raw path) so cardinality stays bounded.
func ObserveRequest(method, route string, status int, dur time.Duration) {
	if route == "" {
		route = "unmatched"
	}
	httpRequests.WithLabelValues(method, route, statusText(status)).Inc()
	httpDuration.WithLabelValues(method, route).Observe(dur.Seconds())
}

// SchedulerRun records one scheduler execution. result is "success" or "error".
func SchedulerRun(scheduler, result string) {
	schedulerRuns.WithLabelValues(scheduler, result).Inc()
}

// statusText buckets a status code into its class (2xx/4xx/5xx) to keep the
// status label low-cardinality while still distinguishing success from error.
func statusText(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
