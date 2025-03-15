//go:build tools
// +build tools

package tools

import (
	// 用于检查循环复杂度
	_ "github.com/fzipp/gocyclo/cmd/gocyclo"
	// 用于代码质量检查
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	// 用于格式化代码
	_ "github.com/incu6us/goimports-reviser"
	// 用于导入格式化
	_ "golang.org/x/tools/cmd/goimports"
	// 用于静态代码分析
	_ "honnef.co/go/tools/cmd/staticcheck"
)

// 此文件仅用于记录和安装工具依赖
// 它不会被编译到最终的二进制文件中
