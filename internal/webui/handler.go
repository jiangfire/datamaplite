package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	responsepkg "git.neolidy.top/neo/fuckcmdb/pkg/response"
)

const (
	embeddedSource    = "embedded"
	filesystemSource  = "filesystem"
	placeholderSource = "placeholder"
)

//go:embed all:generated
var generatedAssets embed.FS

//go:embed placeholder/index.html
var placeholderAssets embed.FS

// Mount 将前端资源挂载到 Gin 引擎，并返回实际使用的资源来源。
func Mount(engine *gin.Engine) string {
	assets, source := resolveAssets()
	fileServer := http.FileServerFS(assets)

	engine.NoRoute(func(c *gin.Context) {
		if isBackendRoute(c.Request.URL.Path) {
			c.AbortWithStatusJSON(
				http.StatusNotFound,
				responsepkg.Error(http.StatusNotFound, "NOT_FOUND", "route not found"),
			)
			return
		}

		serveSPA(c, assets, fileServer)
	})

	return source
}

func resolveAssets() (fs.FS, string) {
	if hasEmbeddedIndex() {
		sub, err := fs.Sub(generatedAssets, "generated")
		if err == nil {
			return sub, embeddedSource
		}
	}

	if hasFilesystemIndex() {
		return os.DirFS(filepath.Join("web", "dist")), filesystemSource
	}

	sub, err := fs.Sub(placeholderAssets, "placeholder")
	if err != nil {
		panic(err)
	}

	return sub, placeholderSource
}

func hasEmbeddedIndex() bool {
	info, err := fs.Stat(generatedAssets, "generated/index.html")
	return err == nil && !info.IsDir()
}

func hasFilesystemIndex() bool {
	info, err := os.Stat(filepath.Join("web", "dist", "index.html"))
	return err == nil && !info.IsDir()
}

func isBackendRoute(requestPath string) bool {
	return requestPath == "/api" ||
		strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/mcp" ||
		strings.HasPrefix(requestPath, "/mcp/")
}

func serveSPA(c *gin.Context, assets fs.FS, fileServer http.Handler) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if assetPath, ok := resolveAssetPath(assets, c.Request.URL.Path); ok {
		req := c.Request.Clone(c.Request.Context())
		req.URL.Path = "/" + assetPath
		fileServer.ServeHTTP(c.Writer, req)
		c.Abort()
		return
	}

	if looksLikeStaticAsset(c.Request.URL.Path) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	http.ServeFileFS(c.Writer, c.Request, assets, "index.html")
	c.Abort()
}

func resolveAssetPath(assets fs.FS, requestPath string) (string, bool) {
	cleaned := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(requestPath, "/")), "/")
	if cleaned == "." || cleaned == "" {
		return "", false
	}

	info, err := fs.Stat(assets, cleaned)
	if err != nil || info.IsDir() {
		return "", false
	}

	return cleaned, true
}

func looksLikeStaticAsset(requestPath string) bool {
	cleaned := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(requestPath, "/")), "/")
	if cleaned == "." || cleaned == "" {
		return false
	}

	return path.Ext(path.Base(cleaned)) != ""
}
