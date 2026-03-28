package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	responsepkg "git.neolidy.top/neo/fuckcmdb/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestErrorHandlingMiddleware_WritesHTTPResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(errorHandlingMiddleware())
	router.GET("/boom", func(c *gin.Context) {
		_ = c.Error(http.ErrAbortHandler)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/boom", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp responsepkg.HttpResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)
	require.Equal(t, http.ErrAbortHandler.Error(), resp.Message)
}
