# Project Context

## Purpose
Simple API Gateway is a lightweight HTTP gateway that proxies requests to
multiple backend services with round-robin load balancing, failover, and
optional caching (Redis or in-memory). The goal is to provide a small,
high-performance gateway with simple configuration and low operational
overhead.

## Tech Stack
- Go 1.23.2
- Fiber v2 for HTTP server/proxying
- Cobra for CLI commands
- TOML configuration via BurntSushi/toml
- Zap structured logging (via shlogin logger wrapper)
- Redis (optional cache backend) with in-memory fallback
- Goreleaser and Docker for packaging/distribution

## Project Conventions

### Code Style
- Format with `gofumpt` and `gci` (imports); `goimports` is available.
- Keep line length under 180 characters (golangci-lint `lll`).
- Use structured logging (`zap`) instead of ad-hoc prints.
- Comments in core packages are often bilingual (English/Chinese).

### Architecture Patterns
- CLI entrypoint in `main.go`; subcommands in `cmd/` (Cobra).
- Core logic lives in `internal/` with package boundaries:
  `config`, `router`, `cache`, `loadbalancer`.
- TOML config is parsed and validated before startup.
- Per-route handlers perform cache lookup, backend selection, and response
  caching.
- Load balancing tracks backend health based on request success/failure and
  retries after a timeout.

### Testing Strategy
- There are currently no `_test.go` files in the repo.
- Use `go test ./...` and Makefile targets (`test`, `check`) when tests are added.

### Git Workflow
- Releases are tag-based; builds embed version info via `git describe`.
- Changelog grouping expects Conventional Commit prefixes (`feat`, `fix`, `doc`).
- Prefer feature branches and small, descriptive commits.

## Domain Context
- The gateway routes requests by path to one or more backend URLs.
- Backends are selected via round-robin with failover after repeated failures.
- Caching is configurable globally and per route (TTL, enable/disable, paths).

## Important Constraints
- Configuration MUST be TOML and validated; invalid configs fail fast.
- Redis is optional; the gateway must operate without it.
- Low latency and minimal overhead are primary goals.

## External Dependencies
- Redis (optional) for cache storage
- Backend service URLs defined per route
- Docker and Docker Compose for containerized deployment
- Systemd/init scripts used in OS packages (goreleaser)
