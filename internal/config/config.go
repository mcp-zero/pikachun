package config

import (
	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server         ServerConfig          `mapstructure:"server"`
	Database       DatabaseConfig        `mapstructure:"database"`
	Canal          CanalConfig           `mapstructure:"canal"`
	Log            LogConfig             `mapstructure:"log"`
	DatabaseStorage DatabaseStorageConfig `mapstructure:"database_storage"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

// CanalConfig Canal配置
type CanalConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Charset  string `mapstructure:"charset"`
	ServerID uint32 `mapstructure:"server_id"`

	// binlog 配置
	Binlog BinlogConfig `mapstructure:"binlog"`

	// 监听配置
	Watch WatchConfig `mapstructure:"watch"`

	// 重连配置
	Reconnect ReconnectConfig `mapstructure:"reconnect"`

	// 性能配置
	Performance PerformanceConfig `mapstructure:"performance"`
}

// BinlogConfig binlog 配置
type BinlogConfig struct {
	Filename    string `mapstructure:"filename"`
	Position    uint32 `mapstructure:"position"`
	GTIDEnabled bool   `mapstructure:"gtid_enabled"`
}

// WatchConfig 监听配置
type WatchConfig struct {
	Databases  []string `mapstructure:"databases"`
	Tables     []string `mapstructure:"tables"`
	EventTypes []string `mapstructure:"event_types"`
}

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	MaxAttempts int    `mapstructure:"max_attempts"`
	Interval    string `mapstructure:"interval"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	EventBufferSize int `mapstructure:"event_buffer_size"`
	BatchSize       int `mapstructure:"batch_size"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	Format     string `mapstructure:"format"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

// DatabaseStorageConfig 数据库存储配置
type DatabaseStorageConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// Load 加载配置
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 设置默认值
	setDefaults()

	// 读取环境变量
	viper.AutomaticEnv()

	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，使用默认配置
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "8668")
	viper.SetDefault("database.dsn", "./data/pikachun.db")
	viper.SetDefault("canal.host", "127.0.0.1")
	viper.SetDefault("canal.port", 3307)
	viper.SetDefault("canal.username", "root")
	viper.SetDefault("canal.password", "lidi10")
	viper.SetDefault("canal.charset", "utf8mb4")
	viper.SetDefault("canal.server_id", 1001)

	// binlog 默认配置
	viper.SetDefault("canal.binlog.filename", "")
	viper.SetDefault("canal.binlog.position", 4)
	viper.SetDefault("canal.binlog.gtid_enabled", true)

	// 监听默认配置
	viper.SetDefault("canal.watch.databases", []string{})
	viper.SetDefault("canal.watch.tables", []string{})
	viper.SetDefault("canal.watch.event_types", []string{"INSERT", "UPDATE", "DELETE"})

	// 重连默认配置
	viper.SetDefault("canal.reconnect.max_attempts", 10)
	viper.SetDefault("canal.reconnect.interval", "5s")

	// 性能默认配置
	viper.SetDefault("canal.performance.event_buffer_size", 1000)
	viper.SetDefault("canal.performance.batch_size", 100)

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "./logs/pikachun.log")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_age", 30)
	viper.SetDefault("log.max_backups", 10)

	// 数据库存储默认配置
	viper.SetDefault("database_storage.enabled", true)
}
