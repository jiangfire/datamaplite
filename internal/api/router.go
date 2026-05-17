package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/service"
)

// Router 路由注册器
type Router struct {
	sourceHandler       *SourceHandler
	schemaHandler       *SchemaHandler
	termHandler         *TermHandler
	authHandler         *AuthHandler
	dqHandler           *DQHandler
	tagHandler          *TagHandler
	alertHandler        *AlertHandler
	notificationHandler *NotificationHandler
	authService         *service.AuthService
}

// NewRouter 创建路由注册器
func NewRouter(sourceHandler *SourceHandler, schemaHandler *SchemaHandler, termHandler *TermHandler, authHandler *AuthHandler, dqHandler *DQHandler, tagHandler *TagHandler, alertHandler *AlertHandler, notificationHandler *NotificationHandler, authService *service.AuthService) *Router {
	return &Router{
		sourceHandler:       sourceHandler,
		schemaHandler:       schemaHandler,
		termHandler:         termHandler,
		authHandler:         authHandler,
		dqHandler:           dqHandler,
		tagHandler:          tagHandler,
		alertHandler:        alertHandler,
		notificationHandler: notificationHandler,
		authService:         authService,
	}
}

// Register 注册所有路由
func (r *Router) Register(engine *gin.Engine) {
	baseHandler := NewHandler()

	// Health check
	engine.GET("/health", func(c *gin.Context) {
		baseHandler.JSON(c, gin.H{"status": "ok"})
	})

	// API v1
	v1 := engine.Group("/api/v1")
	{
		// 认证相关（公开）
		auth := v1.Group("/auth")
		{
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/refresh", r.authHandler.RefreshToken)
		}

		// 需要认证的路由
		authorized := v1.Group("")
		authorized.Use(AuthMiddleware(r.authService), GovernanceAuditMiddleware())
		{
			// 当前用户信息
			authorized.GET("/auth/me", r.authHandler.GetCurrentUser)

			// 用户管理（需要管理员权限）
			authorized.POST("/auth/register", AdminMiddleware(), r.authHandler.Register)
			authorized.GET("/auth/users", AdminMiddleware(), r.authHandler.ListUsers)
			authorized.PUT("/auth/users/:id", AdminMiddleware(), r.authHandler.UpdateUser)
			authorized.DELETE("/auth/users/:id", AdminMiddleware(), r.authHandler.DeleteUser)

			// 数据源管理
			sources := authorized.Group("/sources")
			{
				sources.GET("", r.sourceHandler.ListSources)
				sources.POST("", r.sourceHandler.CreateSource)
				sources.GET("/:id", r.sourceHandler.GetSource)
				sources.PUT("/:id", r.sourceHandler.UpdateSource)
				sources.DELETE("/:id", r.sourceHandler.DeleteSource)
				sources.POST("/test-connection", r.sourceHandler.TestConnection)
				sources.POST("/:id/test", r.sourceHandler.TestConnection)
				sources.POST("/:id/sync", r.sourceHandler.TriggerSync)
				sources.GET("/:id/schema", r.sourceHandler.GetSchemaTree)
				sources.GET("/:id/changes", r.sourceHandler.ListSchemaChanges)
			}

			// 全局搜索
			authorized.GET("/columns/search", r.schemaHandler.SearchColumns)

			// Schema浏览器
			columns := authorized.Group("/columns")
			{
				columns.GET("/:id", r.schemaHandler.GetColumnDetail)
				columns.GET("/:id/mappings", r.schemaHandler.GetColumnMappings)
				columns.POST("/:id/mappings", r.schemaHandler.CreateColumnMapping)
				columns.DELETE("/:id/mappings/:mappingId", r.schemaHandler.DeleteColumnMapping)
				columns.GET("/:id/lineage", r.schemaHandler.GetLineage)
				columns.GET("/:id/impact", r.schemaHandler.GetImpactAnalysis)
				columns.POST("/:id/term", r.termHandler.AssignTermToColumn)
				columns.POST("/:id/tags", r.tagHandler.AssignTagsToColumn)
				columns.GET("/:id/tags", r.tagHandler.GetColumnTags)
				columns.DELETE("/:id/tags/:tagId", r.tagHandler.RemoveTagFromColumn)
			}

			// 业务术语管理
			terms := authorized.Group("/terms")
			{
				terms.GET("", r.termHandler.ListTerms)
				terms.POST("", r.termHandler.CreateTerm)
				terms.GET("/:id", r.termHandler.GetTerm)
				terms.PUT("/:id", r.termHandler.UpdateTerm)
				terms.DELETE("/:id", r.termHandler.DeleteTerm)
			}

			// 数据质量管理
			dq := authorized.Group("/dq")
			{
				dq.GET("/rules", r.dqHandler.ListRules)
				dq.POST("/rules", r.dqHandler.CreateRule)
				dq.GET("/rules/:id", r.dqHandler.GetRule)
				dq.PUT("/rules/:id", r.dqHandler.UpdateRule)
				dq.DELETE("/rules/:id", r.dqHandler.DeleteRule)
				dq.POST("/check", r.dqHandler.CheckRules)
				dq.GET("/results", r.dqHandler.GetResults)
				dq.GET("/stats", r.dqHandler.GetStats)
			}

			// 标签管理
			tags := authorized.Group("/tags")
			{
				tags.GET("", r.tagHandler.ListTags)
				tags.POST("", r.tagHandler.CreateTag)
				tags.GET("/:id", r.tagHandler.GetTag)
				tags.PUT("/:id", r.tagHandler.UpdateTag)
				tags.DELETE("/:id", r.tagHandler.DeleteTag)
				tags.GET("/:id/columns", r.tagHandler.GetColumnsByTag)
			}

			// DDL生成
			authorized.POST("/ddl/generate", r.termHandler.GenerateDDL)

			// 告警规则管理
			alerts := authorized.Group("/alerts")
			{
				alerts.GET("/rules", r.alertHandler.ListAlertRules)
				alerts.POST("/rules", r.alertHandler.CreateAlertRule)
				alerts.GET("/rules/:id", r.alertHandler.GetAlertRule)
				alerts.PUT("/rules/:id", r.alertHandler.UpdateAlertRule)
				alerts.DELETE("/rules/:id", r.alertHandler.DeleteAlertRule)
			}

			// 通知管理
			notifications := authorized.Group("/notifications")
			{
				notifications.GET("", r.notificationHandler.ListNotifications)
				notifications.GET("/stats", r.notificationHandler.GetNotificationStats)
				notifications.POST("/read", r.notificationHandler.MarkAsRead)
			}
		}
	}
}
