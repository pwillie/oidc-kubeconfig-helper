package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

type (
	// PrometheusMiddlewareConfig defines the config for PrometheusMiddleware middleware.
	PrometheusMiddlewareConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper middleware.Skipper
	}
)

var (
	// DefaultPrometheusMiddlewareConfig is the default PrometheusMiddleware middleware config.
	DefaultPrometheusMiddlewareConfig = PrometheusMiddlewareConfig{
		Skipper: middleware.DefaultSkipper,
	}

	reqInFlightGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "A gauge of requests currently being served by the wrapped handler.",
	})

	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "A counter for requests to the wrapped handler.",
		},
		[]string{"code", "method"},
	)

	// duration is partitioned by the HTTP method and handler. It uses custom
	// buckets based on the expected request duration.
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_milliseconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method"},
	)
)

func init() {
	// Register all of the metrics in the standard registry.
	prometheus.MustRegister(reqInFlightGauge, reqCounter, reqDuration)
}

// PrometheusMiddleware returns a middleware that logs HTTP requests.
func PrometheusMiddleware() echo.MiddlewareFunc {
	return PrometheusMiddlewareWithConfig(DefaultPrometheusMiddlewareConfig)
}

// PrometheusMiddlewareWithConfig returns a PrometheusMiddleware middleware with config.
func PrometheusMiddlewareWithConfig(config PrometheusMiddlewareConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultPrometheusMiddlewareConfig.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}
			start := time.Now()
			reqInFlightGauge.Inc()
			if err = next(c); err != nil {
				c.Error(err)
			}
			method := c.Request().Method
			status := strconv.Itoa(c.Response().Status)
			reqCounter.WithLabelValues(status, method).Inc()
			stop := time.Now()
			l := stop.Sub(start)
			reqDuration.WithLabelValues(status, method).Observe(float64(l) / 1000000)
			reqInFlightGauge.Dec()
			return
		}
	}
}
