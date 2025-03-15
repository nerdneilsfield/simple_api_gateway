package router

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"

	"github.com/gofiber/fiber/v2"
	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/nerdneilsfield/simple_api_gateway/internal/cache"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"go.uber.org/zap"
)

var logger = loggerPkg.GetLogger()
var cacheManager *cache.CacheManager

// generateCacheKey creates a unique cache key based on the request
// 根据请求创建唯一的缓存键
func generateCacheKey(c *fiber.Ctx, route config.Route) string {
	// Use request method, path, query parameters, and body to generate cache key
	// 使用请求方法、路径、查询参数和请求体生成缓存键
	h := md5.New()
	h.Write([]byte(c.Method()))
	h.Write([]byte(c.Path()))
	h.Write(c.Request().URI().QueryString())
	h.Write(c.Body())
	return route.Path + ":" + hex.EncodeToString(h.Sum(nil))
}

// shouldCache determines if a request should be cached based on configuration
// 根据配置确定请求是否应该被缓存
func shouldCache(route config.Route, globalCacheEnabled bool) bool {
	// If route explicitly disables cache, don't cache
	// 如果路由明确禁用缓存，则不缓存
	if !route.CacheEnable {
		return false
	}

	// If route enables cache but global cache is disabled, don't cache
	// 如果路由启用缓存，但全局缓存禁用，则不缓存
	if !globalCacheEnabled {
		return false
	}

	// If cache TTL is 0, don't cache
	// 如果缓存TTL为0，则不缓存
	if route.CacheTTL <= 0 {
		return false
	}

	return true
}

// CreateNewHandler creates a new request handler for the given route
// 为给定的路由创建新的请求处理程序
func CreateNewHandler(route config.Route, globalCacheEnabled bool) fiber.Handler {
	targetURL, err := url.Parse(route.Backend)
	if err != nil {
		return func(c *fiber.Ctx) error {
			return c.Status(500).SendString("Error parsing backend URL")
		}
	}

	return func(c *fiber.Ctx) error {
		// Check if caching should be used
		// 检查是否应该使用缓存
		useCache := shouldCache(route, globalCacheEnabled) && cacheManager != nil

		// If using cache, try to get response from cache
		// 如果使用缓存，尝试从缓存获取响应
		if useCache {
			cacheKey := generateCacheKey(c, route)
			cachedResponse, err := cacheManager.Get(cacheKey)
			if err == nil {
				// Cache hit, return cached response
				// 缓存命中，直接返回缓存的响应
				logger.Debug("Cache hit", zap.String("path", c.Path()), zap.String("method", c.Method()))
				return c.Send(cachedResponse)
			}
			logger.Debug("Cache miss", zap.String("path", c.Path()), zap.String("method", c.Method()), zap.Error(err))
		}

		// Build target URL
		// 构建目标URL
		trimmedPath := c.Path()[len(route.Path):]
		queryString := string(c.Request().URI().QueryString())
		targetFullURL := targetURL.String() + trimmedPath
		if queryString != "" {
			targetFullURL += "?" + queryString
		}

		// Create proxy request
		// 创建代理请求
		req := fiber.AcquireAgent()
		defer fiber.ReleaseAgent(req)

		// Set method and URL
		// 设置方法和URL
		req.Request().SetRequestURI(targetFullURL)
		req.Request().Header.SetMethod(string(c.Method()))

		// Copy all headers
		// 复制所有头部
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Request().Header.SetBytesKV(key, value)
		})

		if route.UaClient != "" {
			req.Request().Header.Set("User-Agent", route.UaClient)
		}

		// Add request body
		// 添加请求体
		if len(c.Body()) > 0 {
			req.Request().SetBody(c.Body())
		}

		// Send request
		// 发送请求
		if err := req.Parse(); err != nil {
			return c.Status(500).SendString(fmt.Sprintf("Error: %v", err))
		}

		// Send request and get response
		// 发送请求并获取响应
		statusCode, body, errs := req.Bytes()
		if len(errs) > 0 {
			return c.Status(500).SendString(fmt.Sprintf("Error: %v", errs))
		}

		// If successful response and should cache, cache the response
		// 如果是成功的响应并且应该缓存，则缓存响应
		if useCache && statusCode >= 200 && statusCode < 300 {
			cacheKey := generateCacheKey(c, route)
			if err := cacheManager.Set(cacheKey, body, route.CacheTTL); err != nil {
				logger.Error("Failed to cache response", zap.String("path", c.Path()), zap.Error(err))
			} else {
				logger.Debug("Response cached", zap.String("path", c.Path()), zap.Int("ttl", route.CacheTTL))
			}
		}

		// Set response
		// 设置响应
		c.Status(statusCode)

		// Copy response headers (this part may need to be adjusted based on actual needs)
		// 复制响应头 (这部分可能需要根据实际情况调整)

		return c.Send(body)
	}
}

// Run starts the API gateway server
// 启动API网关服务器
func Run(config_ *config.Config) {
	app := fiber.New()

	// Initialize cache manager
	// 初始化缓存管理器
	if config_.Cache.Enabled {
		var err error
		cacheManager, err = cache.NewCacheManager(config_.Cache)
		if err != nil {
			logger.Warn("Failed to initialize cache manager, running without cache", zap.Error(err))
		} else {
			defer cacheManager.Close()
		}
	}

	for _, route := range config_.Routes {
		app.All(route.Path+"/*", CreateNewHandler(route, config_.Cache.Enabled))
	}

	addrString := config_.Host + ":" + fmt.Sprint(config_.Port)
	if err := app.Listen(addrString); err != nil {
		logger.Fatal("failed to run server", zap.Error(err))
	}
}
