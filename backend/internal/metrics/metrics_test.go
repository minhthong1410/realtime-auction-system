package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMetricsInit(t *testing.T) {
	// Init should not panic when called (already registered in init)
	// Just verify metrics are accessible
	assert.NotNil(t, HttpRequestsTotal)
	assert.NotNil(t, HttpRequestDuration)
	assert.NotNil(t, ActiveWebsockets)
	assert.NotNil(t, BidsTotal)
	assert.NotNil(t, DepositsTotal)
	assert.NotNil(t, WithdrawalsTotal)
	assert.NotNil(t, CacheHits)
}

func TestMetricsMiddlewareRecords(t *testing.T) {
	// Create isolated registry for test
	reg := prometheus.NewRegistry()
	testCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "test_http_total"},
		[]string{"method", "path", "status"},
	)
	reg.MustRegister(testCounter)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		testCounter.WithLabelValues(c.Request.Method, c.FullPath(), "200").Inc()
	})
	r.GET("/api/test", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Verify metric was recorded
	mfs, err := reg.Gather()
	require.NoError(t, err)
	found := false
	for _, mf := range mfs {
		if mf.GetName() == "test_http_total" {
			found = true
			assert.Greater(t, len(mf.GetMetric()), 0)
		}
	}
	assert.True(t, found, "metric should be recorded")
}

func TestMetricsHandler(t *testing.T) {
	r := gin.New()
	r.GET("/metrics", Handler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "go_goroutines")
}

func TestBusinessMetricsIncrement(t *testing.T) {
	// Just verify they don't panic when incremented
	BidsTotal.Inc()
	DepositsTotal.WithLabelValues("completed").Inc()
	WithdrawalsTotal.WithLabelValues("pending").Inc()
	CacheHits.WithLabelValues("hit").Inc()
	CacheHits.WithLabelValues("miss").Inc()
	ActiveWebsockets.Set(5)
}
