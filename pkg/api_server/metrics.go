package apiserver

import (
	"context"
	"net/http"
	"time"

	"github.com/sid-technologies/replayvalidator/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// Prometheus naming. The subsystem and label must be spelled identically across
// every metric here: a mismatch does not error, it produces a second metric
// family that no dashboard queries.
const (
	metricSubsystem = "http_server"
	labelHandler    = "handler"
)

var (
	// inFlight is a gauge of the number of currently in-flight requests.
	inFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "in_flight_requests",
		Subsystem: metricSubsystem,
		Help:      "A gauge of requests currently being served by the wrapped handler.",
	}, []string{labelHandler})

	// counter is a counter of the number of requests to the wrapped handler.
	counter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "requests_total",
			Subsystem: metricSubsystem,
			Help:      "A counter for requests to the wrapped handler.",
		},
		[]string{"code", labelHandler, "method"},
	)

	// duration is partitioned by the HTTP method and handler.
	duration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "request_duration_seconds",
			Subsystem: metricSubsystem,
			Help:      "A histogram of latencies for requests.",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{labelHandler, "method"},
	)

	// requestSize is a histogram of the request size.
	requestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "request_size_bytes",
			Subsystem: metricSubsystem,
			Help:      "A histogram of request sizes for requests.",
			Buckets:   prometheus.ExponentialBucketsRange(128, 1024*1024, 8),
		},
		[]string{labelHandler},
	)

	// responseSize is a histogram of the response size.
	responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "response_size_bytes",
			Subsystem: metricSubsystem,
			Help:      "A histogram of response sizes for requests.",
			Buckets:   prometheus.ExponentialBucketsRange(128, 1024*1024, 8),
		},
		[]string{labelHandler},
	)
)

// InstrumentHandler wraps an http.Handler and provides Prometheus metrics for it.
func InstrumentHandler(name string) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return promhttp.InstrumentHandlerInFlight(inFlight.With(prometheus.Labels{labelHandler: name}),
			promhttp.InstrumentHandlerDuration(duration.MustCurryWith(prometheus.Labels{labelHandler: name}),
				promhttp.InstrumentHandlerCounter(counter.MustCurryWith(prometheus.Labels{labelHandler: name}),
					promhttp.InstrumentHandlerRequestSize(requestSize.MustCurryWith(prometheus.Labels{labelHandler: name}),
						promhttp.InstrumentHandlerResponseSize(responseSize.MustCurryWith(prometheus.Labels{labelHandler: name}), next),
					),
				),
			),
		)
	}
}

// ServeMonitoring serves the monitoring API on a separate port.
func ServeMonitoring(ctx context.Context, done <-chan struct{}, address string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              address,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		Handler:           mux,
	}

	go func() {
		<-done

		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		log.Info().Msg("Shutdown detected, stopping monitoring server")

		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to shutdown monitoring server")
		}
	}()

	return errors.Wrap(srv.ListenAndServe(), "serve monitoring")
}
