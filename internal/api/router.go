package api

import (
	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"github.com/gin-gonic/gin"
)

// Router 路由注册器
type Router struct {
	sourceHandler *SourceHandler
	schemaHandler *SchemaHandler
	termHandler   *TermHandler
}

// NewRouter 创建路由注册器
func NewRouter(sourceHandler *SourceHandler, schemaHandler *SchemaHandler, termHandler *TermHandler) *Router {
	return &Router{
		sourceHandler: sourceHandler,
		schemaHandler: schemaHandler,
		termHandler:   termHandler,
	}
}

// Register 注册所有路由
func (r *Router) Register(engine *gin.Engine) {
	// Health check
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, model.BaseResponse{Success: true, Data: gin.H{"status": "ok"}})
	})

	// API v1
	v1 := engine.Group("/api/v1")
	{
		// 数据源管理
		sources := v1.Group("/sources")
		{
			sources.GET("", r.sourceHandler.ListSources)
			sources.POST("", r.sourceHandler.CreateSource)
			sources.GET("/:id", r.sourceHandler.GetSource)
			sources.PUT("/:id", r.sourceHandler.UpdateSource)
			sources.DELETE("/:id", r.sourceHandler.DeleteSource)
			sources.POST("/:id/test", r.sourceHandler.TestConnection)
			sources.POST("/:id/sync", r.sourceHandler.TriggerSync)
			sources.GET("/:id/schema", r.sourceHandler.GetSchemaTree)
			sources.GET("/:id/changes", r.sourceHandler.ListSchemaChanges)
		}

		// 全局搜索
		v1.GET("/columns/search", r.schemaHandler.SearchColumns)

		// Schema浏览器
		columns := v1.Group("/columns")
		{
			columns.GET("/:id", r.schemaHandler.GetColumnDetail)
			columns.GET("/:id/mappings", r.schemaHandler.GetColumnMappings)
			columns.POST("/:id/mappings", r.schemaHandler.CreateColumnMapping)
			columns.DELETE("/:id/mappings/:mappingId", r.schemaHandler.DeleteColumnMapping)
			columns.GET("/:id/lineage", r.schemaHandler.GetLineage)
			columns.GET("/:id/impact", r.schemaHandler.GetImpactAnalysis)
			columns.POST("/:id/term", r.termHandler.AssignTermToColumn)
		}

		// 业务术语管理
		terms := v1.Group("/terms")
		{
			terms.GET("", r.termHandler.ListTerms)
			terms.POST("", r.termHandler.CreateTerm)
			terms.GET("/:id", r.termHandler.GetTerm)
			terms.PUT("/:id", r.termHandler.UpdateTerm)
			terms.DELETE("/:id", r.termHandler.DeleteTerm)
		}

		// DDL生成
		v1.POST("/ddl/generate", r.termHandler.GenerateDDL)
	}
}
