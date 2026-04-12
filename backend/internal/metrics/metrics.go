package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path"},
	)

	ActiveWebsockets = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_active_connections",
			Help: "Current active WebSocket connections",
		},
	)

	BidsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "auction_bids_total",
			Help: "Total bids placed",
		},
	)

	DepositsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_deposits_total",
			Help: "Total deposit transactions",
		},
		[]string{"status"},
	)

	WithdrawalsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_withdrawals_total",
			Help: "Total withdrawal transactions",
		},
		[]string{"status"},
	)

	CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Cache hit/miss counter",
		},
		[]string{"result"},
	)
)

func Init() {
	prometheus.MustRegister(
		HttpRequestsTotal,
		HttpRequestDuration,
		ActiveWebsockets,
		BidsTotal,
		DepositsTotal,
		WithdrawalsTotal,
		CacheHits,
	)
}

// Middleware records HTTP request metrics.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())

		HttpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HttpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// Handler returns the Prometheus metrics HTTP handler.
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
