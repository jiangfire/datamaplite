package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("no api key configured - reject all", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware(""))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("no key provided - 401", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware("secret-key-123"))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("wrong key via query param - 401", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware("secret-key-123"))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics?api_key=wrong-key", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("correct key via query param - 200", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware("secret-key-123"))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics?api_key=secret-key-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("correct key via header - 200", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware("secret-key-123"))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("X-Metrics-Key", "secret-key-123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("wrong key via header - 401", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware("secret-key-123"))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("X-Metrics-Key", "wrong-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("query param takes precedence over header", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware("secret-key-123"))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics?api_key=secret-key-123", nil)
		req.Header.Set("X-Metrics-Key", "wrong-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("response is JSON error format", func(t *testing.T) {
		router := gin.New()
		router.Use(MetricsAuthMiddleware("secret-key-123"))
		router.GET("/metrics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "error")
	})
}

func TestHealthEndpoints_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("health endpoint accessible without auth", func(t *testing.T) {
		router := gin.New()
		hc := NewHealthChecker(nil)
		hc.RegisterHealthRoutes(router)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ready endpoint accessible without auth", func(t *testing.T) {
		router := gin.New()
		hc := NewHealthChecker(nil)
		hc.RegisterHealthRoutes(router)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
