package logger

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

var (
	// counter is a singleton for the counter vec.
	counter *prometheus.CounterVec //nolint:gochecknoglobals
)

// PrometheusHook calls Prometheus statistics at log write.
type PrometheusHook struct{}

// Run implements zerolog.Hook run method.
func (h PrometheusHook) Run(_ *zerolog.Event, level zerolog.Level, _ string) {
	if level != zerolog.NoLevel {
		counter.WithLabelValues(level.String()).Inc()
	}
}

// NewPrometheusHook returns a prometheus hook counting how often a specific log level was used.
func NewPrometheusHook(feedService string) PrometheusHook {
	if counter == nil {
		counter = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "log_statements_total",
				Help:        "Number of log statements, differentiated by log level.",
				ConstLabels: prometheus.Labels{"service": feedService},
			},
			[]string{"level"},
		)
	}

	return PrometheusHook{}
}
