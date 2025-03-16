package router

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/nerdneilsfield/simple_api_gateway/internal/cache"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"github.com/nerdneilsfield/simple_api_gateway/internal/loadbalancer"
	"go.uber.org/zap"
)

var (
	logger       = loggerPkg.GetLogger()
	cacheManager *cache.CacheManager
)

// 存储每个路由的负载均衡器
// Store load balancers for each route
var (
	routeLoadBalancers = make(map[string]loadbalancer.LoadBalancer)
	loadBalancerMutex  sync.RWMutex
)

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
	key := route.Path + ":" + hex.EncodeToString(h.Sum(nil))
	logger.Debug("Generated cache key",
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
		zap.String("route", route.Path),
		zap.String("key", key))
	return key
}

// shouldCache determines if a request should be cached based on configuration
// 根据配置确定请求是否应该被缓存
func shouldCache(route config.Route, globalCacheEnabled bool, requestPath string) bool {
	// If route explicitly disables cache, don't cache
	// 如果路由明确禁用缓存，则不缓存
	if !route.CacheEnable {
		logger.Debug("Cache disabled for route", zap.String("path", route.Path))
		return false
	}

	// If route enables cache but global cache is disabled, don't cache
	// 如果路由启用缓存，但全局缓存禁用，则不缓存
	if !globalCacheEnabled {
		logger.Debug("Global cache disabled", zap.String("path", route.Path))
		return false
	}

	// If cache TTL is 0, don't cache
	// 如果缓存TTL为0，则不缓存
	if route.CacheTTL <= 0 {
		logger.Debug("Cache TTL is 0, not caching", zap.String("path", route.Path))
		return false
	}

	// Check if the request path is in the cache paths list
	// 检查请求路径是否在可缓存路径列表中
	if len(route.CachePaths) > 0 {
		relativePath := strings.TrimPrefix(requestPath, route.Path)
		logger.Debug("Checking cache paths",
			zap.String("routePath", route.Path),
			zap.String("requestPath", requestPath),
			zap.String("relativePath", relativePath),
			zap.Strings("cachePaths", route.CachePaths))

		// If CachePaths is specified but the path doesn't match any, don't cache
		// 如果指定了CachePaths但路径不匹配任何一个，则不缓存
		pathMatch := false
		for _, cachePath := range route.CachePaths {
			if strings.HasPrefix(relativePath, cachePath) {
				pathMatch = true
				logger.Debug("Path match found for caching",
					zap.String("relativePath", relativePath),
					zap.String("cachePath", cachePath))
				break
			}
		}
		if !pathMatch {
			logger.Debug("No matching cache path found, not caching",
				zap.String("relativePath", relativePath))
			return false
		}
	} else {
		logger.Debug("No cache paths specified, caching all paths for route",
			zap.String("path", route.Path))
	}

	logger.Debug("Request will be cached", zap.String("path", requestPath), zap.Int("ttl", route.CacheTTL))
	return true
}

// getLoadBalancer 获取或创建路由的负载均衡器
// getLoadBalancer gets or creates a load balancer for a route
func getLoadBalancer(route config.Route) loadbalancer.LoadBalancer {
	loadBalancerMutex.RLock()
	lb, exists := routeLoadBalancers[route.Path]
	loadBalancerMutex.RUnlock()

	if !exists {
		loadBalancerMutex.Lock()
		defer loadBalancerMutex.Unlock()

		// 再次检查，避免并发创建
		// Check again to avoid concurrent creation
		lb, exists = routeLoadBalancers[route.Path]
		if !exists {
			lb = loadbalancer.NewRoundRobinLoadBalancer(route.Backends)
			routeLoadBalancers[route.Path] = lb
			logger.Info("Created load balancer for route",
				zap.String("path", route.Path),
				zap.Strings("backends", route.Backends))
		}
	}

	return lb
}

// sendProxyRequest sends the request to the backend and returns the response
// 向后端发送请求并返回响应
func sendProxyRequest(c *fiber.Ctx, targetFullURL string, route config.Route) (int, []byte, map[string][]string, error) {
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

	// 创建响应对象来存储响应
	// Create response object to store the response
	resp := fiber.AcquireResponse()
	defer fiber.ReleaseResponse(resp)
	req.SetResponse(resp)

	// Send request
	// 发送请求
	if err := req.Parse(); err != nil {
		return 0, nil, nil, err
	}

	// 获取响应
	// Get response
	statusCode, body, errs := req.Bytes()
	if len(errs) > 0 {
		return 0, nil, nil, errs[0]
	}

	// 获取响应头
	// Get response headers
	headers := make(map[string][]string)
	resp.Header.VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		headers[k] = append(headers[k], v)
	})

	return statusCode, body, headers, nil
}

// handleBackendRequest processes the request to the backend server
// 处理后端服务器请求
func handleBackendRequest(c *fiber.Ctx, lb loadbalancer.LoadBalancer, route config.Route) (int, []byte, map[string][]string, error) {
	// 记录开始时间，用于计算响应时间
	// Record start time for response time calculation
	startTime := time.Now()

	// 获取下一个后端
	// Get next backend
	backendURL := lb.NextBackend()
	if backendURL == "" {
		return 503, nil, nil, c.Status(503).SendString("No backend servers available")
	}

	// 构建代理请求
	// Build proxy request
	targetFullURL, err := buildTargetURL(c, backendURL, route)
	if err != nil {
		logger.Error("Error parsing backend URL", zap.String("backend", backendURL), zap.Error(err))
		return 500, nil, nil, c.Status(500).SendString("Error parsing backend URL")
	}

	// 创建并发送请求
	// Create and send request
	statusCode, body, headers, err := sendProxyRequest(c, targetFullURL, route)
	if err != nil {
		lb.ReportFailure(backendURL)
		logger.Error("Backend request failed",
			zap.String("backend", backendURL),
			zap.Error(err))
		return 500, nil, nil, c.Status(500).SendString(fmt.Sprintf("Error: %v", err))
	}

	// 请求成功，报告成功
	// Request succeeded, report success
	responseTime := time.Since(startTime)
	lb.ReportSuccess(backendURL, responseTime)
	logger.Debug("Backend request succeeded",
		zap.String("backend", backendURL),
		zap.Int("statusCode", statusCode),
		zap.Duration("responseTime", responseTime))

	return statusCode, body, headers, nil
}

// buildTargetURL constructs the target URL for the proxy request
// 构建代理请求的目标URL
func buildTargetURL(c *fiber.Ctx, backendURL string, route config.Route) (string, error) {
	// 解析后端URL
	// Parse backend URL
	targetURL, err := url.Parse(backendURL)
	if err != nil {
		return "", err
	}

	// Build target URL
	// 构建目标URL
	trimmedPath := c.Path()[len(route.Path):]
	queryString := string(c.Request().URI().QueryString())
	targetFullURL := targetURL.String() + trimmedPath
	if queryString != "" {
		targetFullURL += "?" + queryString
	}

	return targetFullURL, nil
}

// CreateNewHandler creates a new request handler for the given route
// 为给定的路由创建新的请求处理程序
func CreateNewHandler(route config.Route, globalCacheEnabled bool) fiber.Handler {
	// 获取路由的负载均衡器
	// Get load balancer for the route
	lb := getLoadBalancer(route)

	return func(c *fiber.Ctx) error {
		requestStartTime := time.Now()
		requestPath := c.Path()
		requestMethod := c.Method()

		logger.Debug("Handling request",
			zap.String("path", requestPath),
			zap.String("method", requestMethod),
			zap.String("route", route.Path))

		// 检查是否应该使用缓存
		// Check if caching should be used
		useCache := shouldCache(route, globalCacheEnabled, requestPath) && cacheManager != nil

		logCacheStatus(useCache, requestPath, requestMethod)

		// 如果使用缓存，尝试从缓存获取响应
		// If using cache, try to get response from cache
		if useCache {
			if cachedResponse := tryGetFromCache(c, route, requestPath, requestMethod); cachedResponse != nil {
				return c.Send(cachedResponse)
			}
		}

		// 处理后端请求
		// Handle backend request
		statusCode, body, headers, err := handleBackendRequest(c, lb, route)
		if err != nil {
			return err
		}

		// 如果需要，缓存响应
		// Cache response if needed
		if useCache {
			tryCacheResponse(c, route, requestPath, requestMethod, statusCode, body)
		}

		// 记录请求总处理时间
		// Record total request processing time
		requestDuration := time.Since(requestStartTime)
		logger.Debug("Request completed",
			zap.String("path", requestPath),
			zap.String("method", requestMethod),
			zap.Int("statusCode", statusCode),
			zap.Duration("totalTime", requestDuration))

		// 设置响应状态码
		// Set response status code
		c.Status(statusCode)

		// 复制响应头
		// Copy response headers
		for key, values := range headers {
			for _, value := range values {
				c.Response().Header.Add(key, value)
			}
		}

		// 发送响应体
		// Send response body
		return c.Send(body)
	}
}

// logCacheStatus logs whether cache is enabled for the request
// 记录请求是否启用了缓存
func logCacheStatus(useCache bool, requestPath, requestMethod string) {
	if useCache {
		logger.Debug("Cache is enabled for this request",
			zap.String("path", requestPath),
			zap.String("method", requestMethod))
	} else {
		logger.Debug("Cache is disabled for this request",
			zap.String("path", requestPath),
			zap.String("method", requestMethod))
	}
}

// tryGetFromCache attempts to get a response from cache
// 尝试从缓存获取响应
func tryGetFromCache(c *fiber.Ctx, route config.Route, requestPath, requestMethod string) []byte {
	cacheKey := generateCacheKey(c, route)
	logger.Debug("Attempting to get response from cache",
		zap.String("path", requestPath),
		zap.String("key", cacheKey))

	cacheStartTime := time.Now()
	cachedResponse, err := cacheManager.Get(cacheKey)
	cacheLookupDuration := time.Since(cacheStartTime)

	if err == nil {
		// Cache hit, return cached response
		// 缓存命中，直接返回缓存的响应
		logger.Debug("Cache hit",
			zap.String("path", requestPath),
			zap.String("method", requestMethod),
			zap.String("key", cacheKey),
			zap.Duration("lookupTime", cacheLookupDuration),
			zap.Int("responseSize", len(cachedResponse)))

		return cachedResponse
	}

	logger.Debug("Cache miss",
		zap.String("path", requestPath),
		zap.String("method", requestMethod),
		zap.String("key", cacheKey),
		zap.Duration("lookupTime", cacheLookupDuration),
		zap.Error(err))

	return nil
}

// tryCacheResponse attempts to cache a successful response
// 尝试缓存成功的响应
func tryCacheResponse(c *fiber.Ctx, route config.Route, requestPath, requestMethod string, statusCode int, body []byte) {
	// If successful response and should cache, cache the response
	// 如果是成功的响应并且应该缓存，则缓存响应
	if statusCode >= 200 && statusCode < 300 {
		cacheKey := generateCacheKey(c, route)
		logger.Debug("Caching successful response",
			zap.String("path", requestPath),
			zap.String("method", requestMethod),
			zap.String("key", cacheKey),
			zap.Int("statusCode", statusCode),
			zap.Int("responseSize", len(body)),
			zap.Int("ttl", route.CacheTTL))

		cacheStartTime := time.Now()
		if err := cacheManager.Set(cacheKey, body, route.CacheTTL); err != nil {
			logger.Error("Failed to cache response",
				zap.String("path", requestPath),
				zap.String("key", cacheKey),
				zap.Error(err))
		} else {
			cacheDuration := time.Since(cacheStartTime)
			logger.Debug("Response cached successfully",
				zap.String("path", requestPath),
				zap.String("key", cacheKey),
				zap.Int("ttl", route.CacheTTL),
				zap.Duration("cacheTime", cacheDuration))
		}
	} else {
		logger.Debug("Not caching response",
			zap.String("path", requestPath),
			zap.String("method", requestMethod),
			zap.Int("statusCode", statusCode),
			zap.Bool("useCache", true))
	}
}

// Run starts the API gateway server
// 启动API网关服务器
func Run(config_ *config.Config) {
	app := fiber.New()

	// Initialize cache manager
	// 初始化缓存管理器
	if config_.Cache.Enabled {
		logger.Info("Initializing cache manager",
			zap.Bool("useRedis", config_.Cache.UseRedis),
			zap.String("redisPrefix", config_.Cache.RedisPrefix))

		var err error
		cacheStartTime := time.Now()
		cacheManager, err = cache.NewCacheManager(config_.Cache)
		cacheDuration := time.Since(cacheStartTime)

		if err != nil {
			logger.Warn("Failed to initialize cache manager, running without cache",
				zap.Error(err),
				zap.Duration("initTime", cacheDuration))
		} else {
			logger.Info("Cache manager initialized successfully",
				zap.Bool("useRedis", config_.Cache.UseRedis),
				zap.Duration("initTime", cacheDuration))
			defer func() {
				logger.Info("Closing cache manager")
				cacheManager.Close()
			}()
		}
	} else {
		logger.Info("Cache is disabled in configuration, running without cache")
	}

	// 初始化路由处理程序
	// Initialize route handlers
	routeCount := len(config_.Routes)
	logger.Info("Initializing routes", zap.Int("routeCount", routeCount))

	for i, route := range config_.Routes {
		backendCount := len(route.Backends)
		logger.Info("Setting up route",
			zap.Int("routeIndex", i+1),
			zap.Int("totalRoutes", routeCount),
			zap.String("path", route.Path),
			zap.Int("backendCount", backendCount),
			zap.Bool("cacheEnabled", route.CacheEnable && config_.Cache.Enabled),
			zap.Int("cacheTTL", route.CacheTTL),
			zap.Int("cachePathCount", len(route.CachePaths)))

		app.All(route.Path+"/*", CreateNewHandler(route, config_.Cache.Enabled))
	}

	addrString := config_.Host + ":" + fmt.Sprint(config_.Port)
	logger.Info("Starting server", zap.String("address", addrString))
	if err := app.Listen(addrString); err != nil {
		logger.Fatal("failed to run server", zap.Error(err))
	}
}
