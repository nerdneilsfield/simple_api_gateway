projectname := "simple_api_gateway"

# 列出所有可用的命令
default:
    @just --list

# 构建 Golang 二进制文件
build:
    go build -ldflags "-X main.version=$(git describe --abbrev=0 --tags)" -o {{projectname}}

# 安装 Golang 二进制文件
install:
    go install -ldflags "-X main.version=$(git describe --abbrev=0 --tags)"

# 运行应用程序
run:
    go run -ldflags "-X main.version=$(git describe --abbrev=0 --tags)" main.go

# 更新 Go 模块依赖
mod-tidy:
    go mod tidy

# 安装构建依赖
bootstrap: mod-tidy
    go generate -tags tools tools/tools.go
    go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
    go install golang.org/x/tools/cmd/goimports@latest
    go install honnef.co/go/tools/cmd/staticcheck@latest
    go install mvdan.cc/gofumpt@latest
    go install github.com/daixiang0/gci@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 运行测试并显示覆盖率
test: clean
    go test --cover -parallel=1 -v -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | sort -rnk3

# 清理环境
clean:
    rm -rf coverage.out dist {{projectname}} {{projectname}}.exe

# 显示测试覆盖率
cover:
    go test -v -race $(go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
    go tool cover -func=coverage.out

# 格式化 Go 文件
fmt:
    gofumpt -w .
    gci write .

# 运行 linter
lint:
    golangci-lint run -c .golang-ci.yml

# 运行 go vet 检查
vet:
    go vet ./...

# 运行 gocyclo 检查循环复杂度
cyclo:
    gocyclo -over 15 .

# 运行 staticcheck 静态分析
staticcheck:
    staticcheck ./...

# 运行 goimports 检查导入格式
imports:
    goimports -l -w .

# 运行所有代码检查
check: fmt vet cyclo staticcheck imports lint
    @echo "All code checks passed!"

# 测试发布
release-test:
    goreleaser release  --snapshot --clean

# 运行 pre-commit 钩子（已注释）
# pre-commit:
#     pre-commit run --all-files
