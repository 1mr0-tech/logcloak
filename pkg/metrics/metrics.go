package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ProcessedLines = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logcloak_lines_processed_total",
			Help: "Total log lines processed by the masker sidecar.",
		},
		[]string{"pod", "namespace"},
	)

	MaskedLines = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logcloak_lines_masked_total",
			Help: "Log lines where at least one pattern matched.",
		},
		[]string{"pod", "namespace", "pattern"},
	)

	DroppedLines = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logcloak_dropped_lines_total",
			Help: "Log lines dropped (fail-closed); broken down by reason.",
		},
		[]string{"pod", "namespace", "reason"},
	)

	ProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "logcloak_processing_duration_seconds",
			Help:    "Per-line masking latency.",
			Buckets: prometheus.ExponentialBuckets(0.000001, 10, 7),
		},
		[]string{"pod", "namespace"},
	)

	WebhookAdmissions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logcloak_webhook_admissions_total",
			Help: "Webhook admission outcomes.",
		},
		[]string{"result"},
	)

	WebhookErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logcloak_webhook_errors_total",
			Help: "Webhook failures.",
		},
		[]string{"reason"},
	)

	RuleCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "logcloak_rule_cache_size",
		Help: "Number of pod rule sets currently cached.",
	})
)

func MustRegister() {
	prometheus.MustRegister(
		ProcessedLines, MaskedLines, DroppedLines,
		ProcessingDuration, WebhookAdmissions, WebhookErrors,
		RuleCacheSize,
	)
}
