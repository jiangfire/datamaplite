package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestHandler_JSON(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	router.GET("/test", func(c *gin.Context) {
		handler.JSON(c, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	assert.Empty(t, resp.ErrorCode)
	assert.Empty(t, resp.Message)
}

func TestHandler_JSONWithStatus(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	router.GET("/test", func(c *gin.Context) {
		handler.JSONWithStatus(c, http.StatusCreated, gin.H{"message": "created"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	assert.Empty(t, resp.ErrorCode)
}

func TestHandler_Success(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	router.POST("/test", func(c *gin.Context) {
		handler.Success(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	assert.Nil(t, resp.Data)
}

func TestHandler_Error(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	router.GET("/test", func(c *gin.Context) {
		handler.Error(c, http.StatusBadRequest, "TEST_ERROR", "test error message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "TEST_ERROR", resp.ErrorCode)
	assert.Equal(t, "test error message", resp.Message)
}

func TestHandler_BadRequest(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	router.GET("/test", func(c *gin.Context) {
		handler.BadRequest(c, "invalid input")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestHandler_NotFound(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	router.GET("/test", func(c *gin.Context) {
		handler.NotFound(c, "resource not found")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "NOT_FOUND", resp.ErrorCode)
}

func TestHandler_InternalError(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	router.GET("/test", func(c *gin.Context) {
		handler.InternalError(c, "internal server error")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)
}

func TestHandler_BindJSON_Success(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	type TestRequest struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	router.POST("/test", func(c *gin.Context) {
		var req TestRequest
		if !handler.BindJSON(c, &req) {
			return
		}
		handler.JSON(c, req)
	})

	body := `{"name": "test", "email": "test@example.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
}

func TestHandler_BindJSON_BindingTagValidation(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	type TestRequest struct {
		Name string `json:"name" binding:"required"`
	}

	router.POST("/test", func(c *gin.Context) {
		var req TestRequest
		if !handler.BindJSON(c, &req) {
			return
		}
		handler.JSON(c, req)
	})

	body := `{}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
	assert.Contains(t, resp.Message, "Invalid request body")
}

func TestHandler_BindJSON_InvalidJSON(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	type TestRequest struct {
		Name string `json:"name"`
	}

	router.POST("/test", func(c *gin.Context) {
		var req TestRequest
		if !handler.BindJSON(c, &req) {
			return
		}
		handler.JSON(c, req)
	})

	body := `{"invalid json`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestHandler_BindJSON_ValidationError(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	type TestRequest struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	router.POST("/test", func(c *gin.Context) {
		var req TestRequest
		if !handler.BindJSON(c, &req) {
			return
		}
		handler.JSON(c, req)
	})

	body := `{"name": "", "email": "invalid-email"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
	assert.Contains(t, resp.Message, "Validation failed")
}

func TestHandler_BindQuery_Success(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	type TestQuery struct {
		Page     int    `form:"page"`
		PageSize int    `form:"page_size"`
		Keyword  string `form:"keyword"`
	}

	router.GET("/test", func(c *gin.Context) {
		var q TestQuery
		if !handler.BindQuery(c, &q) {
			return
		}
		handler.JSON(c, q)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?page=1&page_size=10&keyword=test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
}

func TestHandler_BindQuery_InvalidQuery(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	type TestQuery struct {
		Page     int      `query:"page"`
		IsActive *bool    `query:"is_active"`
		Ratio    *float64 `query:"ratio"`
	}

	router.GET("/test", func(c *gin.Context) {
		var q TestQuery
		if !handler.BindQuery(c, &q) {
			return
		}
		handler.JSON(c, q)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?page=invalid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_BindQuery_ValidationError(t *testing.T) {
	router := setupTestRouter()
	handler := NewHandler()

	type TestQuery struct {
		Keyword string `form:"keyword" validate:"required,min=2"`
	}

	router.GET("/test", func(c *gin.Context) {
		var q TestQuery
		if !handler.BindQuery(c, &q) {
			return
		}
		handler.JSON(c, q)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test?keyword=a", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
	assert.Contains(t, resp.Message, "Validation failed")
}
