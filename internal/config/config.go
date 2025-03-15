package config

import (
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/nerdneilsfield/shlogin/pkg/network"
	"go.uber.org/zap"
)

var logger = loggerPkg.GetLogger()

//go:embed example_config.toml
var exampleConfigToml embed.FS

type Config struct {
	Port        int     `toml:"port"`
	Host        string  `toml:"host"`
	LogFilePath string  `toml:"log_file_path"`
	Cache       Cache   `toml:"cache"`
	Routes      []Route `toml:"route"`
}

type Cache struct {
	Enabled     bool   `toml:"enabled"`      // Enable cache / 启用缓存
	UseRedis    bool   `toml:"use_redis"`    // Use Redis for caching / 使用Redis缓存
	RedisURL    string `toml:"redis_url"`    // Redis connection URL / Redis连接URL
	RedisDB     int    `toml:"redis_db"`     // Redis database number / Redis数据库编号
	RedisPrefix string `toml:"redis_prefix"` // Redis key prefix / Redis键前缀
}

type Route struct {
	Path        string `toml:"path"`         // Route path / 路由路径
	Backend     string `toml:"backend"`      // Backend service URL / 后端服务URL
	UaClient    string `toml:"ua_client"`    // User-Agent / 用户代理
	CacheTTL    int    `toml:"cache_ttl"`    // Cache TTL in seconds (0 = no cache) / 缓存时间，单位为秒，0表示不缓存
	CacheEnable bool   `toml:"cache_enable"` // Enable cache for this route / 是否启用缓存，默认跟随全局设置
}

// ParseConfig parses the config file at the given path
// 解析给定路径的配置文件
func ParseConfig(path string) (*Config, error) {
	logger.Debug("parsing config", zap.String("path", path))
	var config Config
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		logger.Error("failed to parse config", zap.Error(err))
		return nil, err
	}

	if config.LogFilePath != "" {
		logger.SetLogFilePath(config.LogFilePath)
		logger.SetSaveToFile(true)
		logger.Reset()
	}

	return &config, nil
}

// ValidateConfig validates the config
// 验证配置
func ValidateConfig(config *Config) error {

	if config.Port < 0 && config.Port > 65535 {
		logger.Error("Port is not valid", zap.Int("port", config.Port))
		return fmt.Errorf("port is not valid")
	}

	if config.Host == "" {
		logger.Error("Host is not valid", zap.String("host", config.Host))
		return fmt.Errorf("host is not valid")
	}

	// Validate cache config / 验证缓存配置
	if config.Cache.Enabled && config.Cache.UseRedis {
		if config.Cache.RedisURL == "" {
			logger.Error("Redis URL is empty but Redis cache is enabled")
			return fmt.Errorf("redis URL is empty but Redis cache is enabled")
		}

		// Validate Redis connection / 验证Redis连接
		_, err := url.Parse(config.Cache.RedisURL)
		if err != nil {
			logger.Error("Redis URL is not valid", zap.String("redis_url", config.Cache.RedisURL))
			return fmt.Errorf("redis URL is not valid: %v", err)
		}
	}

	if len(config.Routes) == 0 {
		logger.Error("no routes found in config")
		return fmt.Errorf("no routes found in config")
	}

	existingPaths := make(map[string]bool)

	for _, route := range config.Routes {
		if route.Path == "" {
			logger.Error("route path is empty", zap.String("path", route.Path))
			return fmt.Errorf("route path is empty")
		}
		if existingPaths[route.Path] {
			logger.Error("route path is duplicated", zap.String("path", route.Path))
			return fmt.Errorf("route path is duplicated")
		}
		existingPaths[route.Path] = true
		if route.Backend == "" {
			logger.Error("route backend is empty", zap.String("path", route.Path))
			return fmt.Errorf("route backend is empty")
		}

		if _, err := url.ParseRequestURI(route.Backend); err != nil {
			logger.Error("route backend is not a valid URL", zap.String("path", route.Path), zap.String("backend", route.Backend))
			return fmt.Errorf("route backend is not a valid URL")
		}

		if _, err := network.HttpConnect(route.Backend); err != nil {
			logger.Error("failed to connect to route backend", zap.String("path", route.Path), zap.String("backend", route.Backend))
			return fmt.Errorf("failed to connect to route backend")
		}

		// Validate cache TTL / 验证缓存TTL
		if route.CacheTTL < 0 {
			logger.Error("route cache TTL is negative", zap.String("path", route.Path), zap.Int("cache_ttl", route.CacheTTL))
			return fmt.Errorf("route cache TTL is negative")
		}
	}

	return nil
}

// GetExampleConfig returns the example config as a string
// 返回示例配置作为字符串
func GetExampleConfig() (string, error) {
	exampleConfig, err := exampleConfigToml.ReadFile("example_config.toml")
	if err != nil {
		return "", err
	}
	return string(exampleConfig), nil
}

// GenerateExampleConfigPath generates an example config file at the given path
// 在给定路径生成示例配置文件
func GenerateExampleConfigPath(examplePath string) error {
	exampleConfig, err := GetExampleConfig()
	if err != nil {
		logger.Error("Failed to get example config")
		return err
	}

	if examplePath == "" {
		examplePath = "./example.toml"
	}

	ext := filepath.Ext(examplePath)
	if ext != ".toml" {
		logger.Error("example config file must have .toml extension", zap.String("path", examplePath))
		return fmt.Errorf("example config file must have .toml extension")
	}

	dir := filepath.Dir(examplePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		logger.Error("example config directory does not exist", zap.String("path", dir), zap.Error(err))
		return fmt.Errorf("example config directory does not exist")
	}

	return os.WriteFile(examplePath, []byte(exampleConfig), 0644)
}
