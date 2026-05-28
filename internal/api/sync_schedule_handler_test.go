package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/stretchr/testify/assert"
)

func setupSyncScheduleAuthTest(t *testing.T, role model.UserRole) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	authMiddleware := func(c *gin.Context) {
		c.Set(AuthContextKey, &model.AuthContext{
			UserID:   "test-user",
			Username: "testuser",
			Role:     role,
		})
		c.Next()
	}

	authorized := router.Group("/api/v1")
	authorized.Use(authMiddleware)

	schedules := authorized.Group("/sync/schedules")
	{
		schedules.GET("", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		schedules.GET("/:id", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		schedules.POST("", AdminMiddleware(), func(c *gin.Context) { c.JSON(201, gin.H{"ok": true}) })
		schedules.PUT("/:id", AdminMiddleware(), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		schedules.DELETE("/:id", AdminMiddleware(), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	}

	return router
}

func TestSyncScheduleRoutes_AdminRequired(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		role       model.UserRole
		wantStatus int
	}{
		{"admin can create", http.MethodPost, "/api/v1/sync/schedules", `{"name":"t"}`, model.UserRoleAdmin, http.StatusCreated},
		{"user cannot create", http.MethodPost, "/api/v1/sync/schedules", `{"name":"t"}`, model.UserRoleUser, http.StatusForbidden},
		{"admin can update", http.MethodPut, "/api/v1/sync/schedules/s1", `{"name":"t"}`, model.UserRoleAdmin, http.StatusOK},
		{"user cannot update", http.MethodPut, "/api/v1/sync/schedules/s1", `{"name":"t"}`, model.UserRoleUser, http.StatusForbidden},
		{"admin can delete", http.MethodDelete, "/api/v1/sync/schedules/s1", "", model.UserRoleAdmin, http.StatusOK},
		{"user cannot delete", http.MethodDelete, "/api/v1/sync/schedules/s1", "", model.UserRoleUser, http.StatusForbidden},
		{"user can list", http.MethodGet, "/api/v1/sync/schedules", "", model.UserRoleUser, http.StatusOK},
		{"admin can list", http.MethodGet, "/api/v1/sync/schedules", "", model.UserRoleAdmin, http.StatusOK},
		{"user can get by id", http.MethodGet, "/api/v1/sync/schedules/s1", "", model.UserRoleUser, http.StatusOK},
		{"admin can get by id", http.MethodGet, "/api/v1/sync/schedules/s1", "", model.UserRoleAdmin, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupSyncScheduleAuthTest(t, tt.role)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code, "body: %s", w.Body.String())
		})
	}
}
