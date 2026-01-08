<p align="center">
  <img src="https://raw.githubusercontent.com/nerdneilsfield/simple_api_gateway/master/assets/logo.png" alt="Simple API Gateway Logo" width="200">
</p>

# Simple API Gateway

[![Go Report Card](https://goreportcard.com/badge/github.com/nerdneilsfield/simple_api_gateway)](https://goreportcard.com/report/github.com/nerdneilsfield/simple_api_gateway)
[![GoDoc](https://godoc.org/github.com/nerdneilsfield/simple_api_gateway?status.svg)](https://godoc.org/github.com/nerdneilsfield/simple_api_gateway)
[![License](https://img.shields.io/github/license/nerdneilsfield/simple_api_gateway)](https://github.com/nerdneilsfield/simple_api_gateway/blob/master/LICENSE)
[![Release](https://img.shields.io/github/v/release/nerdneilsfield/simple_api_gateway)](https://github.com/nerdneilsfield/simple_api_gateway/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nerdneilsfield/simple_api_gateway)](https://golang.org/)
[![Docker Pulls](https://img.shields.io/docker/pulls/nerdneils/simple_api_gateway)](https://hub.docker.com/r/nerdneils/simple_api_gateway)
[![Build Status](https://github.com/nerdneilsfield/simple_api_gateway/workflows/build/badge.svg)](https://github.com/nerdneilsfield/simple_api_gateway/actions)

<p align="center">
  <b>一个轻量级、高性能的 API 网关，支持负载均衡、缓存和多后端服务</b>
  <br>
  <b>A lightweight, high-performance API Gateway with load balancing, caching, and multi-backend support</b>
</p>

Simple API Gateway is a lightweight API gateway tool for proxying requests to multiple backend services.

*简单API网关是一个轻量级的API网关工具，用于将请求代理到多个后端服务。*

## Features / 功能特点

- Support for multiple backend service proxying / 支持多后端服务代理
- Load balancing with round-robin and failover / 支持轮询负载均衡和故障转移
- Configuration file validation / 配置文件验证
- Detailed logging / 详细的日志记录
- Support for debug and release modes / 支持调试和发布模式
- Request caching with Redis or in-memory / 支持Redis或内存请求缓存
- Built-in RepoWiki docs at `/wiki`

## Quick Start / 快速开始

### 1. Install / 安装

```bash
# Using Go
go install github.com/nerdneilsfield/simple_api_gateway@latest

# Using Docker
docker pull nerdneils/simple_api_gateway:latest
```

### 2. Create a config file / 创建配置文件

```bash
simple-api-gateway gen config.toml
```

### 3. Start the gateway / 启动网关

```bash
simple-api-gateway serve config.toml
```

That's it! Your API Gateway is now running on port 8080 (default).

*就是这样！您的 API 网关现在运行在 8080 端口（默认）。*

## Installation / 安装

Ensure you have Go 1.16 or higher installed, then run:

*确保您已安装 Go 1.16 或更高版本，然后运行：*

```bash
go get github.com/nerdneilsfield/simple_api_gateway
```

## Usage / 使用方法

Simple API Gateway provides the following commands:

*Simple API Gateway 提供以下命令：*

1. Start the service / 启动服务:

```bash
simple-api-gateway serve <config_file_path>
```

2. Check the config file / 检查配置文件:

```bash
simple-api-gateway check <config_file_path>
```

3. View the version information / 查看版本信息:

```bash
simple-api-gateway version
```

4. Generate the config file / 生成配置文件:

```bash
simple-api-gateway gen <config_file_path>
```

## Running with Docker / 使用 Docker 运行

### Simple Docker Run / 简单Docker运行

```bash
docker run -d --name simple-api-gateway -p 8080:8080 -v /etc/simple_api_gateway/config.toml:/config.toml nerdneils/simple_api_gateway:latest
```

### Docker Compose / Docker Compose部署

The project provides two Docker Compose configurations: one with Redis cache and one with in-memory cache.

*项目提供了两种Docker Compose配置：一种使用Redis缓存，一种使用内存缓存。*

#### With Redis Cache / 使用Redis缓存

<details>
<summary>点击展开 Docker Compose 配置示例 / Click to expand Docker Compose example</summary>

```yaml
# docker-compose-with-redis.yml
services:
  # API Gateway Service / API网关服务
  simple-api-gateway:
    image: nerdneils/simple_api_gateway:latest
    container_name: simple-api-gateway
    ports:
      - "8080:8080"
    volumes:
      - ./config-with-redis.toml:/config.toml
    depends_on:
      - redis
    restart: always
    command: serve /config.toml
    networks:
      - api-gateway-network

  # Redis Cache Service / Redis缓存服务
  redis:
    image: redis:7-alpine
    container_name: redis-cache
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    restart: always
    networks:
      - api-gateway-network
    command: redis-server --appendonly yes
    sysctls:
      vm.overcommit_memory: 1

networks:
  api-gateway-network:
    driver: bridge

volumes:
  redis-data:
    driver: local
```

</details>

Start with / 启动命令:

```bash
docker-compose -f docker-compose-with-redis.yml up -d
```

#### Without Redis (Memory Cache) / 不使用Redis（内存缓存）

<details>
<summary>点击展开 Docker Compose 配置示例 / Click to expand Docker Compose example</summary>

```yaml
# docker-compose-without-redis.yml
services:
  # API Gateway Service (Memory Cache) / API网关服务（内存缓存）
  simple-api-gateway:
    image: nerdneils/simple_api_gateway:latest
    container_name: simple-api-gateway
    ports:
      - "8080:8080"
    volumes:
      - ./config-without-redis.toml:/config.toml
    restart: always
    command: serve /config.toml
    networks:
      - api-gateway-network

networks:
  api-gateway-network:
    driver: bridge
```

</details>

Start with / 启动命令:

```bash
docker-compose -f docker-compose-without-redis.yml up -d
```

For more details on Docker deployment, see [DOCKER-README.md](DOCKER-README.md).

*有关Docker部署的更多详细信息，请参阅[DOCKER-README.md](DOCKER-README.md)。*

## Configuration / 配置

Configuration file uses TOML format. Example configuration:

*配置文件使用 TOML 格式。配置文件示例：*

<details>
<summary>点击展开配置示例 / Click to expand configuration example</summary>

```toml
# example_test.toml
port = 8080                                  # Port to listen on / 监听端口
host = "0.0.0.0"                            # Host to bind to / 绑定主机
log_file_path = "/var/log/simple-api-gateway.log"  # Log file path / 日志文件路径

[cache]
enabled = true                              # Enable cache / 启用缓存
use_redis = true                            # Use Redis for caching / 使用Redis缓存
redis_url = "redis://localhost:6379"        # Redis connection URL / Redis连接URL
redis_db = 0                                # Redis database number / Redis数据库编号
redis_prefix = "api_gateway:"               # Redis key prefix / Redis键前缀

[[route]]
path = "/api"                               # Route path / 路由路径
backends = [                                # Backend service URLs / 后端服务URL列表
  "https://api1.example.com",
  "https://api2.example.com",
  "https://api3.example.com"
]
ua_client = "User-Agent string"             # User-Agent / 用户代理
cache_ttl = 60                              # Cache TTL in seconds / 缓存有效期（秒）
cache_enable = true                         # Enable cache for this route / 为此路由启用缓存
```

</details>

## Load Balancing / 负载均衡

Simple API Gateway supports load balancing across multiple backend servers for each route.

*Simple API Gateway 支持对每个路由的多个后端服务器进行负载均衡。*

### Features / 特性

- Round-robin load balancing / 轮询负载均衡
- Automatic failover / 自动故障转移
- Health checking / 健康检查
- Backend recovery / 后端恢复

### Configuration / 配置

For each route, you can specify multiple backend servers:

*对于每个路由，您可以指定多个后端服务器：*

<details>
<summary>点击展开负载均衡配置示例 / Click to expand load balancing configuration example</summary>

```toml
[[route]]
path = "/api"
backends = [
  "https://api1.example.com",
  "https://api2.example.com",
  "https://api3.example.com"
]
```

</details>

### Behavior / 行为

- Requests are distributed across healthy backends in a round-robin fashion
  *请求以轮询方式分布在健康的后端之间*
- If a backend fails, it is marked as unhealthy and removed from the rotation
  *如果后端失败，它将被标记为不健康并从轮询中移除*
- After a timeout period (default: 30 seconds), unhealthy backends are retried
  *在超时期（默认：30秒）后，将重试不健康的后端*
- If all backends are unhealthy, the system will reset and try all backends again
  *如果所有后端都不健康，系统将重置并再次尝试所有后端*

## Caching Feature / 缓存功能

Simple API Gateway supports request caching using Redis or in-memory cache to improve performance.

*Simple API Gateway 支持使用 Redis 或内存缓存请求，以提高性能。*

### Cache Configuration / 缓存配置

Add the following section to your configuration file to configure caching:

*在配置文件中添加以下部分来配置缓存：*

<details>
<summary>点击展开全局缓存配置示例 / Click to expand global cache configuration example</summary>

```toml
[cache]
enabled = true                              # Enable cache / 启用缓存
use_redis = true                            # Use Redis for caching / 使用Redis缓存
redis_url = "redis://localhost:6379"        # Redis connection URL / Redis连接URL
redis_db = 0                                # Redis database number / Redis数据库编号
redis_prefix = "api_gateway:"               # Redis key prefix / Redis键前缀
```

</details>

### Route Cache Configuration / 路由缓存配置

For each route, you can configure caching behavior individually:

*对于每个路由，可以单独配置缓存行为：*

<details>
<summary>点击展开路由缓存配置示例 / Click to expand route cache configuration example</summary>

```toml
[[route]]
path = "/api"                               # Route path / 路由路径
backends = [                                # Backend service URLs / 后端服务URL列表
  "https://api1.example.com",
  "https://api2.example.com"
]
ua_client = "User-Agent string"             # User-Agent / 用户代理
cache_ttl = 60                              # Cache TTL in seconds (0 = no cache) / 缓存有效期（秒，0表示不缓存）
cache_enable = true                         # Enable cache for this route / 为此路由启用缓存
cache_paths = [                             # Relative paths that can be cached / 可以被缓存的相对路径列表
  "/v1/users",                              # Only cache paths starting with /v1/users / 只缓存以 /v1/users 开头的路径
  "/v1/products",                           # Only cache paths starting with /v1/products / 只缓存以 /v1/products 开头的路径
]
```

</details>

### Caching Behavior / 缓存行为

- If global cache is disabled (`cache.enabled = false`), no routes will be cached
  *如果全局禁用缓存（`cache.enabled = false`），则所有路由都不会缓存*
- If a route explicitly disables caching (`cache_enable = false`), that route won't be cached
  *如果路由明确禁用缓存（`cache_enable = false`），则该路由不会缓存*
- If cache TTL is 0, the route won't be cached
  *如果缓存TTL为0，则该路由不会缓存*
- If Redis connection fails, the system will automatically fall back to in-memory cache
  *如果Redis连接失败，系统会自动降级使用内存缓存*
- If `cache_paths` is specified, only requests to those relative paths will be cached
  *如果指定了`cache_paths`，则只有对这些相对路径的请求才会被缓存*
- If `cache_paths` is empty, all paths under the route will be cached
  *如果`cache_paths`为空，则路由下的所有路径都会被缓存*

Cache keys are generated from the request method, path, query parameters, and request body, ensuring that identical requests hit the same cache.

*缓存键由请求方法、路径、查询参数和请求体组合生成，确保相同的请求会命中相同的缓存。*

## Tech Stack / 技术栈

Simple API Gateway is built with the following technologies:

*Simple API Gateway 使用以下技术构建：*

- **[Go](https://golang.org/)**: Core language providing high performance and concurrency
- **[Gin](https://github.com/gin-gonic/gin)**: HTTP web framework for routing and middleware
- **[Redis](https://redis.io/)**: Optional caching backend for high-performance request caching
- **[TOML](https://github.com/BurntSushi/toml)**: Human-friendly configuration format
- **[Zap](https://github.com/uber-go/zap)**: Ultra-fast, structured logging

## Performance / 性能

Simple API Gateway is designed for high performance with minimal overhead:

*Simple API Gateway 设计为高性能，低开销：*

- **Low Latency**: Typically adds <2ms overhead to proxied requests
- **High Throughput**: Capable of handling thousands of requests per second
- **Memory Efficient**: Low memory footprint even under high load
- **Caching**: Dramatically improves performance for repeated requests

<details>
<summary>点击查看性能基准测试 / Click to view performance benchmarks</summary>

| Scenario | Requests/sec | Latency (avg) | Memory Usage |
|----------|--------------|---------------|--------------|
| Direct Backend | 5,000 | 10ms | - |
| With API Gateway | 4,800 | 12ms | 20MB |
| With Caching | 15,000 | 3ms | 45MB |

*测试环境: 4 CPU cores, 8GB RAM, 1Gbps network*
</details>

## Development / 开发

Project structure:

*项目结构：*

- `cmd/`: Contains command-line interface related code / 包含命令行接口相关代码
- `internal/`: Contains internal packages / 包含内部包
  - `config/`: Configuration parsing and validation / 配置解析和验证
  - `router/`: Route setup and request handling / 路由设置和请求处理
  - `cache/`: Caching implementation / 缓存实现
  - `loadbalancer/`: Load balancing implementation / 负载均衡实现

## Community / 社区

Join our community to get help, share ideas, and contribute to the project:

*加入我们的社区，获取帮助、分享想法并为项目做出贡献：*

- [GitHub Discussions](https://github.com/nerdneilsfield/simple_api_gateway/discussions)

## Support / 支持

If you find Simple API Gateway useful, please consider supporting the project:

*如果您觉得 Simple API Gateway 有用，请考虑支持该项目：*

- ⭐ Star the project on GitHub
- 🐛 Report bugs and suggest features
- 🔀 Submit pull requests
- 📚 Improve documentation
- 🌍 Help with translations

## Contributing / 贡献

Contributions via issues and pull requests are welcome.

*欢迎通过 issues 和 pull requests 做出贡献。*

## License / 许可证

[BSD 3-Clause License]

```
BSD 3-Clause License

Copyright (c) 2024, DengQi

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from
   this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

## Star History / 星标历史

[![Star History Chart](https://api.star-history.com/svg?repos=nerdneilsfield/simple_api_gateway&type=Date)](https://star-history.com/#nerdneilsfield/simple_api_gateway&Date)
