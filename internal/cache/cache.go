package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"go.uber.org/zap"
)

var logger = loggerPkg.GetLogger()

// Cache interface defines methods for cache operations
// 缓存接口定义了缓存操作的方法
type Cache interface {
	Get(key string) (*CacheItem, error)
	Set(key string, value *CacheItem, ttl int) error
	Delete(key string) error
	Close() error
}

// MemoryCache implements Cache interface using in-memory storage
// 内存缓存实现了使用内存存储的缓存接口
type MemoryCache struct {
	cache sync.Map
}

type memoryCacheItem struct {
	value      *CacheItem
	expiration time.Time
}

// NewMemoryCache creates a new memory cache instance
// 创建一个新的内存缓存实例
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		cache: sync.Map{},
	}

	// Start a goroutine to periodically clean expired cache items
	// 启动一个goroutine定期清理过期的缓存项
	go func() {
		for {
			time.Sleep(time.Minute)
			cache.cleanExpired()
		}
	}()

	return cache
}

// cleanExpired removes expired items from the cache
// 从缓存中删除过期项
func (c *MemoryCache) cleanExpired() {
	now := time.Now()
	c.cache.Range(func(key, value interface{}) bool {
		item, ok := value.(memoryCacheItem)
		if !ok {
			c.cache.Delete(key)
			return true
		}

		if !item.expiration.IsZero() && item.expiration.Before(now) {
			c.cache.Delete(key)
		}
		return true
	})
}

// Get retrieves a value from the cache by key
// 通过键从缓存中获取值
func (c *MemoryCache) Get(key string) (*CacheItem, error) {
	logger.Debug("Memory cache: attempting to get item", zap.String("key", key))
	value, ok := c.cache.Load(key)
	if !ok {
		logger.Debug("Memory cache: key not found", zap.String("key", key))
		return nil, errors.New("key not found")
	}

	item, ok := value.(memoryCacheItem)
	if !ok {
		logger.Debug("Memory cache: invalid cache item", zap.String("key", key))
		return nil, errors.New("invalid cache item")
	}

	if !item.expiration.IsZero() && item.expiration.Before(time.Now()) {
		logger.Debug("Memory cache: key expired", zap.String("key", key), zap.Time("expiration", item.expiration))
		c.cache.Delete(key)
		return nil, errors.New("key expired")
	}

	logger.Debug("Memory cache: item retrieved successfully", zap.String("key", key), zap.Int("size", len(item.value.Body)))
	return item.value, nil
}

// Set stores a value in the cache with the given key and TTL
// 将值存储在缓存中，使用给定的键和TTL
func (c *MemoryCache) Set(key string, value *CacheItem, ttl int) error {
	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(time.Duration(ttl) * time.Second)
	}

	// 过滤头部
	value.Headers = FilterHeaders(value.Headers)

	logger.Debug("Memory cache: storing item",
		zap.String("key", key),
		zap.Int("size", len(value.Body)),
		zap.Int("ttl", ttl),
		zap.Time("expiration", expiration))

	c.cache.Store(key, memoryCacheItem{
		value:      value,
		expiration: expiration,
	})

	return nil
}

// Delete removes a value from the cache by key
// 通过键从缓存中删除值
func (c *MemoryCache) Delete(key string) error {
	logger.Debug("Memory cache: deleting item", zap.String("key", key))
	c.cache.Delete(key)
	return nil
}

// Close cleans up resources used by the cache
// 清理缓存使用的资源
func (c *MemoryCache) Close() error {
	// Memory cache doesn't need to close anything
	// 内存缓存不需要关闭任何东西
	return nil
}

// RedisCache implements Cache interface using Redis
// Redis缓存实现了使用Redis的缓存接口
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
	prefix string
}

// NewRedisCache creates a new Redis cache instance
// 创建一个新的Redis缓存实例
func NewRedisCache(config config.Cache) (*RedisCache, error) {
	opts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		logger.Error("Failed to parse Redis URL", zap.Error(err))
		return nil, err
	}

	opts.DB = config.RedisDB
	client := redis.NewClient(opts)
	ctx := context.Background()

	// Test connection
	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return nil, err
	}

	return &RedisCache{
		client: client,
		ctx:    ctx,
		prefix: config.RedisPrefix,
	}, nil
}

// Get retrieves a value from Redis by key
// 通过键从Redis获取值
func (c *RedisCache) Get(key string) (*CacheItem, error) {
	fullKey := c.prefix + key
	logger.Debug("Redis cache: attempting to get item", zap.String("key", fullKey))

	value, err := c.client.Get(c.ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			logger.Debug("Redis cache: key not found", zap.String("key", fullKey))
			return nil, errors.New("key not found")
		}
		logger.Debug("Redis cache: error getting item", zap.String("key", fullKey), zap.Error(err))
		return nil, err
	}

	var item CacheItem
	if err := json.Unmarshal(value, &item); err != nil {
		return nil, err
	}

	logger.Debug("Redis cache: item retrieved successfully", zap.String("key", fullKey), zap.Int("size", len(item.Body)))
	return &item, nil
}

// Set stores a value in Redis with the given key and TTL
// 将值存储在Redis中，使用给定的键和TTL
func (c *RedisCache) Set(key string, value *CacheItem, ttl int) error {
	fullKey := c.prefix + key
	var expiration time.Duration
	if ttl > 0 {
		expiration = time.Duration(ttl) * time.Second
	}

	// 过滤头部
	value.Headers = FilterHeaders(value.Headers)

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	logger.Debug("Redis cache: storing item",
		zap.String("key", fullKey),
		zap.Int("size", len(value.Body)),
		zap.Int("ttl", ttl),
		zap.Duration("expiration", expiration))

	err = c.client.Set(c.ctx, fullKey, data, expiration).Err()
	if err != nil {
		logger.Debug("Redis cache: error setting item", zap.String("key", fullKey), zap.Error(err))
	}
	return err
}

// Delete removes a value from Redis by key
// 通过键从Redis中删除值
func (c *RedisCache) Delete(key string) error {
	fullKey := c.prefix + key
	logger.Debug("Redis cache: deleting item", zap.String("key", fullKey))

	err := c.client.Del(c.ctx, fullKey).Err()
	if err != nil {
		logger.Debug("Redis cache: error deleting item", zap.String("key", fullKey), zap.Error(err))
	}
	return err
}

// Close closes the Redis client connection
// 关闭Redis客户端连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// CacheManager manages cache operations and provides a unified interface
// 缓存管理器管理缓存操作并提供统一接口
type CacheManager struct {
	cache  Cache
	config config.Cache
}

// NewCacheManager creates a new cache manager based on configuration
// 根据配置创建新的缓存管理器
func NewCacheManager(config config.Cache) (*CacheManager, error) {
	var cache Cache
	var err error

	if !config.Enabled {
		logger.Info("Cache is disabled")
		return nil, errors.New("cache is disabled")
	}

	if config.UseRedis {
		logger.Info("Using Redis cache")
		cache, err = NewRedisCache(config)
		if err != nil {
			logger.Error("Failed to create Redis cache", zap.Error(err))
			logger.Info("Falling back to memory cache")
			cache = NewMemoryCache()
		}
	} else {
		logger.Info("Using memory cache")
		cache = NewMemoryCache()
	}

	return &CacheManager{
		cache:  cache,
		config: config,
	}, nil
}

// Get retrieves a value from the cache by key
// 通过键从缓存获取值
func (m *CacheManager) Get(key string) (*CacheItem, error) {
	logger.Debug("Cache manager: get operation", zap.String("key", key))
	value, err := m.cache.Get(key)
	if err != nil {
		logger.Debug("Cache manager: get operation failed", zap.String("key", key), zap.Error(err))
		return nil, err
	}
	logger.Debug("Cache manager: get operation succeeded", zap.String("key", key), zap.Int("size", len(value.Body)))
	return value, nil
}

// Set stores a value in the cache with the given key and TTL
// 将值存储在缓存中，使用给定的键和TTL
func (m *CacheManager) Set(key string, value *CacheItem, ttl int) error {
	logger.Debug("Cache manager: set operation", zap.String("key", key), zap.Int("size", len(value.Body)), zap.Int("ttl", ttl))
	err := m.cache.Set(key, value, ttl)
	if err != nil {
		logger.Debug("Cache manager: set operation failed", zap.String("key", key), zap.Error(err))
	}
	return err
}

// Delete removes a value from the cache by key
// 通过键从缓存中删除值
func (m *CacheManager) Delete(key string) error {
	logger.Debug("Cache manager: delete operation", zap.String("key", key))
	err := m.cache.Delete(key)
	if err != nil {
		logger.Debug("Cache manager: delete operation failed", zap.String("key", key), zap.Error(err))
	}
	return err
}

// Close cleans up resources used by the cache
// 清理缓存使用的资源
func (m *CacheManager) Close() error {
	return m.cache.Close()
}

// 定义不需要缓存的头部列表
var excludedHeaders = map[string]bool{
	"Date":              true,
	"Connection":        true,
	"Transfer-Encoding": true,
	"Cf-Ray":            true,
	"Cf-Cache-Status":   true,
	"Cf-Connecting-Ip":  true,
	"Cf-Worker":         true,
	"X-Real-Ip":         true,
	"X-Forwarded-For":   true,
	"X-Forwarded-Proto": true,
	"X-Request-Id":      true,
	"X-Request-Start":   true,
	"X-Response-Time":   true,
	"X-Runtime":         true,
	"Server":            true,
	"Via":               true,
	"Alt-Svc":           true,
	"Content-Length":    true, // 因为我们可能会修改内容
}

// FilterHeaders 过滤不需要缓存的响应头
func FilterHeaders(headers map[string][]string) map[string][]string {
	filteredHeaders := make(map[string][]string)
	for key, values := range headers {
		// 检查头部名称是否在排除列表中（不区分大小写）
		if !excludedHeaders[strings.ToTitle(key)] {
			filteredHeaders[key] = values
		}
	}
	return filteredHeaders
}

// CacheItem 结构体保持不变，但在存储时使用过滤后的头部
type CacheItem struct {
	Body    []byte              `json:"body"`
	Headers map[string][]string `json:"headers"`
}
