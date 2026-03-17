package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
	Scanner  ScannerConfig  `mapstructure:"scanner"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

// ServerConfig HTTP服务配置
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string        `mapstructure:"type"` // postgres 或 sqlite
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`

	// SQLite-specific
	SQLitePath     string `mapstructure:"sqlite_path"`
	SQLiteMaxConns int32  `mapstructure:"sqlite_max_conns"`
	SQLiteMinConns int32  `mapstructure:"sqlite_min_conns"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// ScannerConfig 采集器配置
type ScannerConfig struct {
	MongoDBSampleSize int `mapstructure:"mongodb_sample_size"`
	MaxLineageDepth   int `mapstructure:"max_lineage_depth"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	BcryptCost      int           `mapstructure:"bcrypt_cost"`
}

// Load 加载配置
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/datamap")
	viper.AddConfigPath(".")

	// 设置默认值
	setDefaults()

	// 环境变量覆盖
	viper.SetEnvPrefix("DATAMAP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 显式绑定关键的嵌套环境变量
	viper.BindEnv("auth.jwt_secret")
	viper.BindEnv("server.port")
	viper.BindEnv("database.type")
	viper.BindEnv("database.sqlite_path")

	// 读取配置文件（可选 - 如果不存在则忽略）
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到 - 使用默认值和环境变量
		} else {
			// 配置文件存在但读取失败
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.shutdown_timeout", "10s")

	// Database defaults - 默认使用 SQLite（方便本地开发测试）
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "datamap")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_conns", 25)
	viper.SetDefault("database.min_conns", 5)
	viper.SetDefault("database.max_conn_lifetime", "1h")
	viper.SetDefault("database.max_conn_idle_time", "30m")

	// SQLite-specific defaults
	viper.SetDefault("database.sqlite_path", "./data/datamap.db")
	viper.SetDefault("database.sqlite_max_conns", 25)
	viper.SetDefault("database.sqlite_min_conns", 5)

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")
	viper.SetDefault("log.output", "stdout")

	// Scanner defaults
	viper.SetDefault("scanner.mongodb_sample_size", 1000)
	viper.SetDefault("scanner.max_lineage_depth", 10)

	// Auth defaults - NO default secret (force production to set it)
	viper.SetDefault("auth.jwt_secret", "")
	viper.SetDefault("auth.access_token_ttl", "15m")
	viper.SetDefault("auth.refresh_token_ttl", "168h") // 7 days
	viper.SetDefault("auth.bcrypt_cost", 10)
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Type != "postgres" && c.Database.Type != "sqlite" {
		return fmt.Errorf("invalid database type: %s (must be 'postgres' or 'sqlite')", c.Database.Type)
	}

	// PostgreSQL 需要数据库名
	if c.Database.Type == "postgres" && c.Database.Database == "" {
		return fmt.Errorf("database name is required for PostgreSQL")
	}

	if c.Log.Level != "debug" && c.Log.Level != "info" &&
		c.Log.Level != "warn" && c.Log.Level != "error" {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	// 要求设置 JWT secret
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required (set DATAMAP_AUTH_JWT_SECRET)")
	}

	return nil
}

// GetEncryptionKey 从环境变量获取加密密钥
func GetEncryptionKey() (string, error) {
	key := os.Getenv("DATAMAP_ENCRYPTION_KEY")
	if key == "" {
		return "", fmt.Errorf("DATAMAP_ENCRYPTION_KEY environment variable is not set")
	}
	if len(key) != 32 {
		return "", fmt.Errorf("DATAMAP_ENCRYPTION_KEY must be 32 bytes (got %d)", len(key))
	}
	return key, nil
}

// DSN 返回PostgreSQL连接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode)
}
