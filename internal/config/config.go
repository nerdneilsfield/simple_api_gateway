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
	Path          string            `toml:"path"`           // Route path / 路由路径
	Backends      []string          `toml:"backends"`       // Backend service URLs / 后端服务URL列表
	UaClient      string            `toml:"ua_client"`      // User-Agent / 用户代理
	CacheTTL      int               `toml:"cache_ttl"`      // Cache TTL in seconds (0 = no cache) / 缓存时间，单位为秒，0表示不缓存
	CacheEnable   bool              `toml:"cache_enable"`   // Enable cache for this route / 是否启用缓存，默认跟随全局设置
	CachePaths    []string          `toml:"cache_paths"`    // Relative paths that can be cached / 可以被缓存的相对路径列表
	CustomHeaders map[string]string `toml:"custom_headers"` // Custom headers to add to requests / 添加到请求中的自定义头部
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

	logger.Debug("config parsed", zap.Any("config", config))

	return &config, nil
}

// ValidateConfig validates the config
// 验证配置
func ValidateConfig(config *Config) error {
	// 验证基本配置
	if err := validateBasicConfig(config); err != nil {
		return err
	}

	// 验证缓存配置
	if err := validateCacheConfig(config); err != nil {
		return err
	}

	// 验证路由配置
	if err := validateRoutes(config); err != nil {
		return err
	}

	return nil
}

// validateBasicConfig validates the basic configuration
// 验证基本配置
func validateBasicConfig(config *Config) error {
	if config.Port < 0 || config.Port > 65535 {
		logger.Error("Port is not valid", zap.Int("port", config.Port))
		return fmt.Errorf("port is not valid")
	}

	if config.Host == "" {
		logger.Error("Host is not valid", zap.String("host", config.Host))
		return fmt.Errorf("host is not valid")
	}

	return nil
}

// validateCacheConfig validates the cache configuration
// 验证缓存配置
func validateCacheConfig(config *Config) error {
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

	return nil
}

// validateRoutes validates the route configurations
// 验证路由配置
func validateRoutes(config *Config) error {
	if len(config.Routes) == 0 {
		logger.Error("no routes found in config")
		return fmt.Errorf("no routes found in config")
	}

	existingPaths := make(map[string]bool)

	for _, route := range config.Routes {
		if err := validateSingleRoute(route, existingPaths); err != nil {
			return err
		}
		existingPaths[route.Path] = true
	}

	return nil
}

// validateSingleRoute validates a single route configuration
// 验证单个路由配置
func validateSingleRoute(route Route, existingPaths map[string]bool) error {
	// 验证路径
	if err := validateRoutePath(route, existingPaths); err != nil {
		return err
	}

	// 验证后端服务
	if err := validateRouteBackends(route); err != nil {
		return err
	}

	// 验证缓存TTL
	if route.CacheTTL < 0 {
		logger.Error("route cache TTL is negative", zap.String("path", route.Path), zap.Int("cache_ttl", route.CacheTTL))
		return fmt.Errorf("route cache TTL is negative")
	}

	return nil
}

// validateRoutePath validates the route path
// 验证路由路径
func validateRoutePath(route Route, existingPaths map[string]bool) error {
	if route.Path == "" {
		logger.Error("route path is empty", zap.String("path", route.Path))
		return fmt.Errorf("route path is empty")
	}

	if existingPaths[route.Path] {
		logger.Error("route path is duplicated", zap.String("path", route.Path))
		return fmt.Errorf("route path is duplicated")
	}

	return nil
}

// validateRouteBackends validates the route backends
// 验证路由后端服务
func validateRouteBackends(route Route) error {
	// 验证后端服务列表
	if len(route.Backends) == 0 {
		logger.Error("route backends is empty", zap.String("path", route.Path))
		return fmt.Errorf("route backends is empty")
	}

	// 验证每个后端服务URL
	for _, backend := range route.Backends {
		if err := validateSingleBackend(route.Path, backend); err != nil {
			return err
		}
	}

	return nil
}

// validateSingleBackend validates a single backend URL
// 验证单个后端服务URL
func validateSingleBackend(routePath, backend string) error {
	if backend == "" {
		logger.Error("route backend is empty", zap.String("path", routePath))
		return fmt.Errorf("route backend is empty")
	}

	if _, err := url.ParseRequestURI(backend); err != nil {
		logger.Error("route backend is not a valid URL", zap.String("path", routePath), zap.String("backend", backend))
		return fmt.Errorf("route backend is not a valid URL")
	}

	if _, err := network.HttpConnect(backend); err != nil {
		logger.Warn("failed to connect to route backend, but will try during runtime",
			zap.String("path", routePath),
			zap.String("backend", backend),
			zap.Error(err))
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

	return os.WriteFile(examplePath, []byte(exampleConfig), 0o644)
}
