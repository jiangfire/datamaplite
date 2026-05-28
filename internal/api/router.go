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
	syncScheduleHandler *SyncScheduleHandler
	dashboardHandler    *DashboardHandler
	authService         *service.AuthService
}

// NewRouter 创建路由注册器
func NewRouter(sourceHandler *SourceHandler, schemaHandler *SchemaHandler, termHandler *TermHandler, authHandler *AuthHandler, dqHandler *DQHandler, tagHandler *TagHandler, alertHandler *AlertHandler, notificationHandler *NotificationHandler, syncScheduleHandler *SyncScheduleHandler, dashboardHandler *DashboardHandler, authService *service.AuthService) *Router {
	return &Router{
		sourceHandler:       sourceHandler,
		schemaHandler:       schemaHandler,
		termHandler:         termHandler,
		authHandler:         authHandler,
		dqHandler:           dqHandler,
		tagHandler:          tagHandler,
		alertHandler:        alertHandler,
		notificationHandler: notificationHandler,
		syncScheduleHandler: syncScheduleHandler,
		dashboardHandler:    dashboardHandler,
		authService:         authService,
	}
}

// Register 注册所有路由
func (r *Router) Register(engine *gin.Engine) {
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
			// 仪表盘统计
			authorized.GET("/dashboard/stats", r.dashboardHandler.GetStats)
			authorized.GET("/dashboard/change-trend", r.dashboardHandler.GetChangeTrend)

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
				sources.GET("/:id", r.sourceHandler.GetSource)
				sources.GET("/:id/schema", r.sourceHandler.GetSchemaTree)
				sources.GET("/:id/changes", r.sourceHandler.ListSchemaChanges)
				sources.POST("", AdminMiddleware(), r.sourceHandler.CreateSource)
				sources.PUT("/:id", AdminMiddleware(), r.sourceHandler.UpdateSource)
				sources.DELETE("/:id", AdminMiddleware(), r.sourceHandler.DeleteSource)
				sources.POST("/test-connection", AdminMiddleware(), r.sourceHandler.TestConnection)
				sources.POST("/:id/test", AdminMiddleware(), r.sourceHandler.TestConnection)
				sources.POST("/:id/sync", AdminMiddleware(), r.sourceHandler.TriggerSync)
			}

			// 全局搜索
			authorized.GET("/columns/search", r.schemaHandler.SearchColumns)

			// Schema浏览器
			columns := authorized.Group("/columns")
			{
				columns.GET("/:id", r.schemaHandler.GetColumnDetail)
				columns.GET("/:id/mappings", r.schemaHandler.GetColumnMappings)
				columns.POST("/:id/mappings", AdminMiddleware(), r.schemaHandler.CreateColumnMapping)
				columns.DELETE("/:id/mappings/:mappingId", AdminMiddleware(), r.schemaHandler.DeleteColumnMapping)
				columns.GET("/:id/lineage", r.schemaHandler.GetLineage)
				columns.GET("/:id/impact", r.schemaHandler.GetImpactAnalysis)
				columns.POST("/:id/term", AdminMiddleware(), r.termHandler.AssignTermToColumn)
				columns.POST("/:id/tags", AdminMiddleware(), r.tagHandler.AssignTagsToColumn)
				columns.GET("/:id/tags", r.tagHandler.GetColumnTags)
				columns.DELETE("/:id/tags/:tagId", AdminMiddleware(), r.tagHandler.RemoveTagFromColumn)
			}

			// 业务术语管理
			terms := authorized.Group("/terms")
			{
				terms.GET("", r.termHandler.ListTerms)
				terms.POST("", AdminMiddleware(), r.termHandler.CreateTerm)
				terms.GET("/:id", r.termHandler.GetTerm)
				terms.PUT("/:id", AdminMiddleware(), r.termHandler.UpdateTerm)
				terms.DELETE("/:id", AdminMiddleware(), r.termHandler.DeleteTerm)
			}

			// 数据质量管理
			dq := authorized.Group("/dq")
			{
				dq.GET("/rules", r.dqHandler.ListRules)
				dq.POST("/rules", AdminMiddleware(), r.dqHandler.CreateRule)
				dq.GET("/rules/:id", r.dqHandler.GetRule)
				dq.PUT("/rules/:id", AdminMiddleware(), r.dqHandler.UpdateRule)
				dq.DELETE("/rules/:id", AdminMiddleware(), r.dqHandler.DeleteRule)
				dq.POST("/check", AdminMiddleware(), r.dqHandler.CheckRules)
				dq.GET("/results", r.dqHandler.GetResults)
				dq.GET("/stats", r.dqHandler.GetStats)
			}

			// 标签管理
			tags := authorized.Group("/tags")
			{
				tags.GET("", r.tagHandler.ListTags)
				tags.POST("", AdminMiddleware(), r.tagHandler.CreateTag)
				tags.GET("/:id", r.tagHandler.GetTag)
				tags.PUT("/:id", AdminMiddleware(), r.tagHandler.UpdateTag)
				tags.DELETE("/:id", AdminMiddleware(), r.tagHandler.DeleteTag)
				tags.GET("/:id/columns", r.tagHandler.GetColumnsByTag)
			}

			// DDL生成
			authorized.POST("/ddl/generate", AdminMiddleware(), r.termHandler.GenerateDDL)

			// 告警规则管理
			alerts := authorized.Group("/alerts")
			{
				alerts.GET("/rules", r.alertHandler.ListAlertRules)
				alerts.POST("/rules", AdminMiddleware(), r.alertHandler.CreateAlertRule)
				alerts.GET("/rules/:id", r.alertHandler.GetAlertRule)
				alerts.PUT("/rules/:id", AdminMiddleware(), r.alertHandler.UpdateAlertRule)
				alerts.DELETE("/rules/:id", AdminMiddleware(), r.alertHandler.DeleteAlertRule)
			}

			// 通知管理
			notifications := authorized.Group("/notifications")
			{
				notifications.GET("", r.notificationHandler.ListNotifications)
				notifications.GET("/stats", r.notificationHandler.GetNotificationStats)
				notifications.POST("/read", r.notificationHandler.MarkAsRead)
			}

			// 定时同步调度管理
			if r.syncScheduleHandler != nil {
				r.syncScheduleHandler.RegisterRoutes(authorized)
			}
		}
	}
}
