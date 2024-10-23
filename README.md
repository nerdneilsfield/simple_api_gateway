好的，我来为您编写一个 README.md 文件。这个 README 将基于您提供的代码和项目结构。

```markdown
# Simple API Gateway

Simple API Gateway 是一个轻量级的 API 网关工具，用于代理请求到多个后端服务。

## 功能特点

- 支持多后端服务代理
- 配置文件验证
- 详细的日志记录
- 支持调试和发布模式

## 安装

确保您已安装 Go 1.16 或更高版本，然后运行：

```bash
go get github.com/nerdneilsfield/simple_api_gateway
```

## 使用方法

Simple API Gateway 提供了以下命令：

1. 启动服务：

```bash
simple-api-gateway serve <config_file_path>
```

2. 检查配置文件：

```bash
simple-api-gateway check <config_file_path>
```

3. 查看版本信息：

```bash
simple-api-gateway version
```

## 配置

配置文件使用 TOML 格式。

```toml
# example_test.toml
port = 8080
host = "0.0.0.0"
log_file_path = "/var/log/simple-api-gateway.log"

[[route]]
path = "/cloudflare"
backend = "https://api.cloudflare.com"
ua_client = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
```

## 开发

项目结构：

- `cmd/`: 包含命令行接口相关代码
- `internal/`: 包含内部包
  - `config/`: 配置解析和验证
  - `router/`: 路由设置和请求处理

## 贡献

欢迎提交 issues 和 pull requests。

## 许可证

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