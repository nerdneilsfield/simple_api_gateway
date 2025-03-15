package loadbalancer

import (
	"sync"
	"sync/atomic"
	"time"

	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"go.uber.org/zap"
)

var logger = loggerPkg.GetLogger()

// BackendStatus 表示后端服务的状态
// BackendStatus represents the status of a backend service
type BackendStatus struct {
	URL           string          // 后端服务URL / Backend service URL
	Healthy       bool            // 是否健康 / Whether it's healthy
	FailCount     int             // 连续失败次数 / Consecutive failure count
	LastFailTime  time.Time       // 最后一次失败时间 / Last failure time
	ResponseTimes []time.Duration // 最近的响应时间 / Recent response times
	mutex         *sync.RWMutex   // 读写锁 / Read-write lock
}

// LoadBalancer 负载均衡器接口
// LoadBalancer interface
type LoadBalancer interface {
	// NextBackend 返回下一个要使用的后端服务
	// NextBackend returns the next backend to use
	NextBackend() string

	// ReportSuccess 报告后端服务请求成功
	// ReportSuccess reports a successful request to the backend
	ReportSuccess(backend string, responseTime time.Duration)

	// ReportFailure 报告后端服务请求失败
	// ReportFailure reports a failed request to the backend
	ReportFailure(backend string)

	// GetBackends 获取所有后端服务
	// GetBackends returns all backends
	GetBackends() []string

	// GetHealthyBackends 获取所有健康的后端服务
	// GetHealthyBackends returns all healthy backends
	GetHealthyBackends() []string
}

// RoundRobinLoadBalancer 实现轮询负载均衡
// RoundRobinLoadBalancer implements round-robin load balancing
type RoundRobinLoadBalancer struct {
	backends     []BackendStatus // 后端服务列表 / List of backends
	current      uint32          // 当前索引 / Current index
	maxFailCount int             // 最大失败次数 / Maximum failure count
	failTimeout  time.Duration   // 失败超时时间 / Failure timeout
	mutex        sync.RWMutex    // 读写锁 / Read-write lock
}

// NewRoundRobinLoadBalancer 创建一个新的轮询负载均衡器
// NewRoundRobinLoadBalancer creates a new round-robin load balancer
func NewRoundRobinLoadBalancer(backends []string) *RoundRobinLoadBalancer {
	lb := &RoundRobinLoadBalancer{
		backends:     make([]BackendStatus, len(backends)),
		current:      0,
		maxFailCount: 3,                // 默认最大失败次数 / Default maximum failure count
		failTimeout:  30 * time.Second, // 默认失败超时时间 / Default failure timeout
		mutex:        sync.RWMutex{},
	}

	for i, backend := range backends {
		lb.backends[i] = BackendStatus{
			URL:           backend,
			Healthy:       true,
			FailCount:     0,
			LastFailTime:  time.Time{},
			ResponseTimes: make([]time.Duration, 0, 10),
			mutex:         &sync.RWMutex{},
		}
	}

	return lb
}

// NextBackend 返回下一个要使用的后端服务
// NextBackend returns the next backend to use
func (lb *RoundRobinLoadBalancer) NextBackend() string {
	// 首先尝试获取下一个健康的后端
	// First try to get the next healthy backend
	healthyBackends := lb.GetHealthyBackends()
	if len(healthyBackends) == 0 {
		// 如果没有健康的后端，重置所有后端状态并返回第一个
		// If no healthy backends, reset all backends and return the first one
		lb.resetBackends()

		lb.mutex.RLock()
		if len(lb.backends) > 0 {
			firstBackend := lb.backends[0].URL
			lb.mutex.RUnlock()
			return firstBackend
		}
		lb.mutex.RUnlock()
		return ""
	}

	// 使用原子操作增加计数器，实现线程安全的轮询
	// Use atomic operation to increment counter for thread-safe round-robin
	current := atomic.AddUint32(&lb.current, 1) % uint32(len(healthyBackends))
	return healthyBackends[current]
}

// ReportSuccess 报告后端服务请求成功
// ReportSuccess reports a successful request to the backend
func (lb *RoundRobinLoadBalancer) ReportSuccess(backend string, responseTime time.Duration) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	for i := range lb.backends {
		if lb.backends[i].URL == backend {
			lb.backends[i].mutex.Lock()
			lb.backends[i].Healthy = true
			lb.backends[i].FailCount = 0

			// 保存最近的响应时间，最多保存10个
			// Save recent response times, up to 10
			if len(lb.backends[i].ResponseTimes) >= 10 {
				lb.backends[i].ResponseTimes = lb.backends[i].ResponseTimes[1:]
			}
			lb.backends[i].ResponseTimes = append(lb.backends[i].ResponseTimes, responseTime)
			lb.backends[i].mutex.Unlock()

			logger.Debug("Backend reported success",
				zap.String("backend", backend),
				zap.Duration("responseTime", responseTime))
			break
		}
	}
}

// ReportFailure 报告后端服务请求失败
// ReportFailure reports a failed request to the backend
func (lb *RoundRobinLoadBalancer) ReportFailure(backend string) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	for i := range lb.backends {
		if lb.backends[i].URL == backend {
			lb.backends[i].mutex.Lock()
			lb.backends[i].FailCount++
			lb.backends[i].LastFailTime = time.Now()

			// 如果连续失败次数超过最大失败次数，标记为不健康
			// If consecutive failures exceed the maximum, mark as unhealthy
			if lb.backends[i].FailCount >= lb.maxFailCount {
				lb.backends[i].Healthy = false
				logger.Warn("Backend marked as unhealthy",
					zap.String("backend", backend),
					zap.Int("failCount", lb.backends[i].FailCount))
			} else {
				logger.Debug("Backend reported failure",
					zap.String("backend", backend),
					zap.Int("failCount", lb.backends[i].FailCount))
			}
			lb.backends[i].mutex.Unlock()
			break
		}
	}

	// 检查是否所有后端都不健康，如果是，重置所有后端
	// Check if all backends are unhealthy, if so, reset all backends
	allUnhealthy := true
	for i := range lb.backends {
		lb.backends[i].mutex.RLock()
		if lb.backends[i].Healthy {
			allUnhealthy = false
		}
		lb.backends[i].mutex.RUnlock()
	}

	if allUnhealthy {
		lb.resetBackends()
	}
}

// GetBackends 获取所有后端服务
// GetBackends returns all backends
func (lb *RoundRobinLoadBalancer) GetBackends() []string {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	result := make([]string, len(lb.backends))
	for i, backend := range lb.backends {
		result[i] = backend.URL
	}
	return result
}

// GetHealthyBackends 获取所有健康的后端服务
// GetHealthyBackends returns all healthy backends
func (lb *RoundRobinLoadBalancer) GetHealthyBackends() []string {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	var result []string
	now := time.Now()

	for i := range lb.backends {
		lb.backends[i].mutex.RLock()

		// 如果后端不健康但已经超过失败超时时间，重新标记为健康以便重试
		// If backend is unhealthy but failure timeout has passed, mark as healthy for retry
		if !lb.backends[i].Healthy && !lb.backends[i].LastFailTime.IsZero() {
			if now.Sub(lb.backends[i].LastFailTime) > lb.failTimeout {
				lb.backends[i].mutex.RUnlock()
				lb.backends[i].mutex.Lock()
				lb.backends[i].Healthy = true
				lb.backends[i].FailCount = 0
				lb.backends[i].mutex.Unlock()
				lb.backends[i].mutex.RLock()

				logger.Info("Backend recovery attempt", zap.String("backend", lb.backends[i].URL))
			}
		}

		if lb.backends[i].Healthy {
			result = append(result, lb.backends[i].URL)
		}
		lb.backends[i].mutex.RUnlock()
	}

	return result
}

// resetBackends 重置所有后端状态
// resetBackends resets all backend statuses
func (lb *RoundRobinLoadBalancer) resetBackends() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	logger.Warn("No healthy backends available, resetting all backends")

	for i := range lb.backends {
		lb.backends[i].mutex.Lock()
		lb.backends[i].Healthy = true
		lb.backends[i].FailCount = 0
		lb.backends[i].mutex.Unlock()
	}
}
