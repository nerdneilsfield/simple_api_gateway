# Docker部署指南 / Docker Deployment Guide

本项目提供了两种Docker Compose配置，一种使用Redis作为缓存，另一种使用内存缓存。

*This project provides two Docker Compose configurations, one using Redis as cache and another using in-memory cache.*

## 使用Redis缓存 / With Redis Cache

使用以下命令启动带有Redis缓存的API网关：

*Use the following command to start the API gateway with Redis cache:*

```bash
docker-compose -f docker-compose-with-redis.yml up -d
```

这将启动两个容器：
*This will start two containers:*

1. `simple-api-gateway`: API网关服务 / API gateway service
2. `redis-cache`: Redis缓存服务 / Redis cache service

## 使用内存缓存 / With Memory Cache

使用以下命令启动使用内存缓存的API网关：

*Use the following command to start the API gateway with memory cache:*

```bash
docker-compose -f docker-compose-without-redis.yml up -d
```

这将只启动API网关服务，使用内存作为缓存。

*This will only start the API gateway service, using memory as cache.*

## 配置文件 / Configuration Files

- `config-with-redis.toml`: 使用Redis缓存的配置 / Configuration with Redis cache
- `config-without-redis.toml`: 使用内存缓存的配置 / Configuration with memory cache

您可以根据需要修改这些配置文件。

*You can modify these configuration files as needed.*

## 停止服务 / Stop Services

要停止服务，请使用以下命令：

*To stop the services, use the following command:*

```bash
# 对于Redis版本 / For Redis version
docker-compose -f docker-compose-with-redis.yml down

# 对于内存缓存版本 / For memory cache version
docker-compose -f docker-compose-without-redis.yml down
```

## 注意事项 / Notes

- Redis数据将持久化到名为`redis-data`的Docker卷中
  *Redis data will be persisted in a Docker volume named `redis-data`*
- 两个配置使用相同的端口(8080)，请不要同时运行
  *Both configurations use the same port (8080), please do not run them simultaneously* 