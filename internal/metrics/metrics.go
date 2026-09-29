package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// RateLimitDecisions counts every Allow call, labelled by endpoint and decision.
	RateLimitDecisions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_decisions_total",
			Help: "Total number of rate limit decisions, labelled by endpoint and result.",
		},
		[]string{"endpoint", "result"}, // result: "allowed" | "denied"
	)

	// RateLimitRemaining tracks the remaining token count as a histogram per endpoint.
	RateLimitRemaining = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rate_limit_remaining_tokens",
			Help:    "Remaining tokens in the bucket at decision time.",
			Buckets: prometheus.LinearBuckets(0, 2, 15),
		},
		[]string{"endpoint"},
	)
)

func init() {
	prometheus.MustRegister(RateLimitDecisions, RateLimitRemaining)
}
