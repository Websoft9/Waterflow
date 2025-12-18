# 开发指南

本文档介绍如何设置 Waterflow 开发环境和开发流程。

## 开发环境设置

### 前置要求

- **Go**: 1.24 或更高版本
- **Make**: 用于运行构建任务
- **Git**: 版本控制
- **Docker**: （可选）用于容器化测试
- **golangci-lint**: 代码检查工具

### 安装开发工具

```bash
# 安装 golangci-lint
make install-tools

# 或手动安装
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 克隆项目

```bash
git clone https://github.com/Websoft9/waterflow.git
cd waterflow
```

### 安装依赖

```bash
go mod download
go mod verify
```

## 开发流程

### 1. 创建功能分支

```bash
git checkout -b feature/your-feature-name
```

### 2. 编写代码

遵循项目代码规范（见下文）。

### 3. 编写测试

- 单元测试文件命名：`xxx_test.go`
- 测试函数命名：`TestXxx`
- 使用 table-driven tests 处理多个测试用例

```go
func TestYourFunction(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {"case1", "input1", "expected1"},
        {"case2", "input2", "expected2"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := YourFunction(tt.input)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 4. 运行测试

```bash
# 运行所有测试
make test

# 运行特定包的测试
go test -v ./pkg/config/

# 生成覆盖率报告
make coverage
```

### 5. 代码检查

```bash
# 运行 linter
make lint

# 格式化代码
make fmt
```

### 6. 构建

```bash
# 构建二进制
make build

# 运行
make run
```

### 7. 提交代码

```bash
git add .
git commit -m "feat: add your feature"
git push origin feature/your-feature-name
```

## 代码规范

### Go 代码风格

遵循以下规范：

- [Effective Go](https://go.dev/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### 命名约定

- **包名**: 小写，不使用下划线或驼峰
- **导出函数**: 驼峰命名（PascalCase）
- **私有函数**: 驼峰命名，首字母小写（camelCase）
- **常量**: 驼峰命名或全大写加下划线

### 错误处理

立即检查错误：

```go
result, err := SomeFunction()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### 注释

- 导出的函数、类型、常量必须有 godoc 注释
- 注释以声明的名称开头

```go
// Config holds all configuration for the application.
type Config struct {
    // ...
}

// Load loads configuration from file and environment variables.
func Load(configFile string) (*Config, error) {
    // ...
}
```

## 项目结构

```
waterflow/
├── cmd/server/          # 服务器入口
├── pkg/                 # 可导出的库代码
│   ├── config/          # 配置管理
│   └── logger/          # 日志系统
├── internal/            # 私有代码
│   └── server/          # HTTP 服务器实现
├── api/                 # API 定义
├── deployments/         # 部署配置
└── test/                # 集成测试
```

## 调试

### 使用 VSCode

创建 `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/server",
            "args": ["--config", "config.yaml", "--log-level", "debug"]
        }
    ]
}
```

### 使用 Delve

```bash
# 安装 delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试
dlv debug ./cmd/server -- --config config.yaml
```

## 常见问题

### 端口被占用

修改 `config.yaml` 中的端口：

```yaml
server:
  port: 9090
```

或使用命令行参数：

```bash
./bin/server --port 9090
```

### 测试失败

确保没有其他进程占用测试端口（18080, 18081）。

## 参考资源

- [Go 官方文档](https://go.dev/doc/)
- [Temporal 文档](https://docs.temporal.io/)
- [项目架构文档](architecture.md)
