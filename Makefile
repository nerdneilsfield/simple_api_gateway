projectname?=simple_api_gateway

default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## build golang binary
	@go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -o $(projectname)

.PHONY: install
install: ## install golang binary
	@go install -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)"

.PHONY: run
run: ## run the app
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)"  main.go

.PHONY: mod-tidy
mod-tidy: ## update go module dependencies
	go mod tidy

.PHONY: bootstrap
bootstrap: mod-tidy ## install build deps
	go generate -tags tools tools/tools.go
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/daixiang0/gci@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: test
test: clean ## display test coverage
	go test --cover -parallel=1 -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | sort -rnk3
	
.PHONY: clean
clean: ## clean up environment
	@rm -rf coverage.out dist/ $(projectname)

.PHONY: cover
cover: ## display test coverage
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: fmt
fmt: ## format go files
	gofumpt -w .
	gci write .

.PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golang-ci.yml

.PHONY: vet
vet: ## run go vet
	go vet ./...

.PHONY: cyclo
cyclo: ## check cyclomatic complexity
	gocyclo -over 15 .

.PHONY: staticcheck
staticcheck: ## run staticcheck static analysis
	staticcheck ./...

.PHONY: imports
imports: ## check and fix import formatting
	goimports -l -w .

.PHONY: check
check: fmt vet cyclo staticcheck imports lint ## run all code checks
	@echo "All code checks passed!"

.PHONY: release-test
release-test: ## test release
	goreleaser release --rm-dist --snapshot --clean --skip-publish

# .PHONY: pre-commit
# pre-commit:	## run pre-commit hooks
# 	pre-commit run --all-files
