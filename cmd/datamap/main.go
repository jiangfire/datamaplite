package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/api"
	"git.neolidy.top/neo/fuckcmdb/internal/config"
	"git.neolidy.top/neo/fuckcmdb/internal/crypto"
	"git.neolidy.top/neo/fuckcmdb/internal/scanner"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func initAuthService(cfg config.AuthConfig, store store.Store) *service.AuthService {
	authCfg := &service.AuthConfig{
		JWTSecret:       cfg.JWTSecret,
		AccessTokenTTL:  cfg.AccessTokenTTL,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
		BcryptCost:      cfg.BcryptCost,
	}
	return service.NewAuthService(store, authCfg)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 初始化日志
	logger, err := initLogger(cfg.Log)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("starting DataMap-Lite",
		zap.String("version", "1.0.0"),
		zap.Int("port", cfg.Server.Port),
		zap.String("database_type", cfg.Database.Type),
	)

	// 获取加密密钥
	encryptionKey, err := config.GetEncryptionKey()
	if err != nil {
		logger.Error("failed to get encryption key", zap.Error(err))
		return err
	}

	// 初始化加密器
	cipher, err := crypto.NewCipher(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to init cipher: %w", err)
	}

	// 初始化数据库连接
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := store.New(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to init store: %w", err)
	}
	defer store.Close()
	if err != nil {
		return fmt.Errorf("failed to init store: %w", err)
	}

	// 初始化扫描器注册表
	registry := scanner.NewRegistry()
	registry.Register("mysql", scanner.NewMySQLScanner())
	registry.Register("mongodb", scanner.NewMongoDBScanner(cfg.Scanner.MongoDBSampleSize))

	// 初始化服务层
	sourceService := service.NewSourceService(store, cipher, registry)
	metadataService := service.NewMetadataService(store)
	termService := service.NewTermService(store)
	ddlService := service.NewDDLService(store)
	authService := initAuthService(cfg.Auth, store)
	dqService := service.NewDQService(store)
	tagService := service.NewTagService(store)
	alertService := service.NewAlertService(store, logger)
	notifService := service.NewNotificationService(store, logger)

	// 解决循环依赖：设置告警服务到数据源服务
	sourceService.SetAlertService(alertService)

	// 初始化API层
	sourceHandler := api.NewSourceHandler(sourceService, metadataService)
	schemaHandler := api.NewSchemaHandler(metadataService)
	termHandler := api.NewTermHandler(termService, ddlService)
	authHandler := api.NewAuthHandler(authService)
	dqHandler := api.NewDQHandler(dqService)
	tagHandler := api.NewTagHandler(tagService)
	alertHandler := api.NewAlertHandler(alertService, notifService, logger)
	notifHandler := api.NewNotificationHandler(notifService, logger)
	router := api.NewRouter(sourceHandler, schemaHandler, termHandler, authHandler, dqHandler, tagHandler, alertHandler, notifHandler, authService)

	// 配置Gin
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(loggingMiddleware(logger))
	engine.Use(errorHandlingMiddleware())

	// 注册路由
	router.Register(engine)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器（在goroutine中）
	go func() {
		logger.Info("HTTP server started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Info("server exited")
	return nil
}

func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var loggerConfig zap.Config

	if cfg.Format == "json" {
		loggerConfig = zap.NewProductionConfig()
	} else {
		loggerConfig = zap.NewDevelopmentConfig()
	}

	switch cfg.Level {
	case "debug":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return loggerConfig.Build()
}

func loggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP request",
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		)
	}
}

func errorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			// 只处理第一个错误
			err := c.Errors[0]
			c.JSON(-1, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": err.Error(),
				},
			})
		}
	}
}
