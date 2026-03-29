package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMount_ServesHTMLForRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	source := Mount(router)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEmpty(t, source)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "<html")
}

func TestMount_ReturnsJSON404ForUnknownAPIPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Mount(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "\"error_code\":\"NOT_FOUND\"")
}

func TestMount_Returns404ForMissingStaticAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Mount(router)

	req := httptest.NewRequest(http.MethodGet, "/static/js/missing.js", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
