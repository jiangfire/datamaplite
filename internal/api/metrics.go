package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/jiangfire/datamaplite/pkg/response"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// metricsRegistry 使用独立的 registry，避免测试时重复注册 panic
	metricsRegistry = prometheus.NewRegistry()

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	activeRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of HTTP requests currently being processed",
		},
	)
)

func init() {
	metricsRegistry.MustRegister(httpRequestsTotal, httpRequestDuration, activeRequests)
}

// MetricsMiddleware 收集 HTTP 指标
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 使用路由模板作为 path label，避免高 cardinality
		path := c.FullPath()
		if path == "" {
			// 对静态资源使用固定 label
			path = "/static"
		}

		activeRequests.Inc()
		defer activeRequests.Dec()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// MetricsAuthMiddleware 验证 /metrics 端点的 API Key
func MetricsAuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "metrics endpoint not configured"})
			c.Abort()
			return
		}

		key := c.Query("api_key")
		if key == "" {
			key = c.GetHeader("X-Metrics-Key")
		}

		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "api key required"})
			c.Abort()
			return
		}

		if key != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RegisterMetricsRoutes 注册 /metrics 路由（带 API Key 认证）
func RegisterMetricsRoutes(engine *gin.Engine, apiKey string) {
	metricsGroup := engine.Group("/metrics")
	metricsGroup.Use(MetricsAuthMiddleware(apiKey))
	metricsGroup.GET("", gin.WrapH(promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{})))
}

// HealthChecker 健康检查器
type HealthChecker struct {
	store store.Store
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(st store.Store) *HealthChecker {
	return &HealthChecker{store: st}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// Check 执行健康检查
func (h *HealthChecker) Check(ctx context.Context) *HealthResponse {
	resp := &HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Checks:    make(map[string]string),
	}

	// 检查数据库连接（使用轻量级 Ping）
	if h.store != nil {
		// 使用超时 context 避免健康检查 hang 住
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := h.store.Ping(pingCtx); err != nil {
			resp.Status = "degraded"
			resp.Checks["database"] = "unhealthy: " + err.Error()
		} else {
			resp.Checks["database"] = "ok"
		}
	} else {
		resp.Checks["database"] = "not_configured"
	}

	return resp
}

// RegisterHealthRoutes 注册健康检查路由
func (h *HealthChecker) RegisterHealthRoutes(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) {
		resp := h.Check(c.Request.Context())
		status := http.StatusOK
		if resp.Status == "degraded" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, response.Success(resp))
	})

	engine.GET("/ready", func(c *gin.Context) {
		resp := h.Check(c.Request.Context())
		if resp.Status == "degraded" {
			c.JSON(http.StatusServiceUnavailable, response.Success(resp))
			return
		}
		c.JSON(http.StatusOK, response.Success(resp))
	})
}
