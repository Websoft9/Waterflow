# Story 3.2: Shell 命令执行节点 (.so 插件)

**状态:** done  
**Epic:** 3 - 核心节点插件库  
**Story ID:** 3.2  
**创建日期:** 2025-12-30  
**完成日期:** 2025-12-30  
**开发者就绪:** ✅

---

## 🎯 核心设计决策 (必读!)

**关键架构约束:**
- ✅ **第一个实际节点实现** - Shell 节点是所有节点的参考实现
- ✅ **基于 Story 3.1** - 使用 `pkg/dsl/node` 包中定义的 Node 接口
- ✅ **完整接口实现** - 实现所有 5 个方法: Name, Version, Params, Execute, Metadata
- ✅ **返回 NodeResult** - Execute 方法返回结构化的 `*NodeResult` 类型
- ✅ **插件编译** - 编译为 .so 文件，Agent 启动时自动加载

**依赖关系:**
- **必须完成:** Story 3.1 (节点接口设计)
- **使用接口:** `pkg/dsl/node.Node`
- **返回类型:** `pkg/dsl/node.NodeResult`
- **参数规范:** `pkg/dsl/node.ParamSpec`

**质量标杆:**
- 最基础、最常用的节点（90% 工作流会使用）
- 其他节点参考此实现的代码结构和测试策略
- 代码质量要求最高（测试覆盖率 >90%）

---

## Story

As a **工作流用户**,  
I want **在 Agent 上执行 Shell 命令**,  
So that **运行系统命令和脚本**。

---

## Acceptance Criteria

**AC1: 基础命令执行**  
**Given** Agent 在目标服务器运行  
**When** 工作流 Step 使用 `exec/shell` 节点  
**Then** 在 Agent 服务器执行指定命令  
**And** 节点编译为 shell.so 插件,Agent 启动时自动加载  
**And** 捕获 stdout 和 stderr  
**And** 返回退出码  

**AC2: 参数支持**  
**Given** Shell 节点执行时  
**When** 配置参数  
**Then** 支持参数: command, args, env, workdir, timeout  
**And** command 为必需参数  
**And** args 为可选参数 (字符串数组)  
**And** env 为可选参数 (键值对映射)  
**And** workdir 为可选参数 (工作目录路径)  
**And** timeout 为可选参数 (秒数,默认 60)  

**AC3: 错误处理**  
**Given** Shell 命令执行  
**When** 发生错误  
**Then** 超时自动终止进程  
**And** 命令执行失败 (exit code ≠ 0) 抛出错误  
**And** 错误信息包含 stdout 和 stderr  
**And** 区分临时错误 (超时) 和永久错误 (命令不存在)  

---

## Tasks / Subtasks

### Task 1: 创建 Shell 节点插件目录结构 (AC1)
- [x] 创建 `plugins/exec/shell/` 目录
- [x] 创建 `plugins/exec/shell/main.go` - 插件主文件
- [x] 创建 `plugins/exec/shell/main_test.go` - 单元测试
- [x] 创建 `plugins/exec/shell/Makefile` - 编译脚本
- [x] 创建 `plugins/exec/shell/README.md` - 节点文档

### Task 2: 实现 Shell 节点接口 (AC1, AC2)
- [x] 定义 ShellNode 结构体
  - [x] 实现 `Name() string` - 返回 "exec/shell"
  - [x] 实现 `Version() string` - 返回 "v1"
  - [x] 实现 `Params() map[string]node.ParamSpec` - 返回参数规范（与 Metadata 一致）
  - [x] 实现 `Metadata() node.NodeMetadata` - 返回节点元数据
  - [x] 实现 `Execute(ctx, inputs) (*NodeResult, error)` - 执行命令并返回结构化结果
- [x] 定义输入参数 Schema (AC2)
  - [x] command (string, required) - Shell 命令
  - [x] args (array, optional) - 命令参数
  - [x] env (object, optional) - 环境变量
  - [x] workdir (string, optional) - 工作目录
  - [x] timeout (int, optional, default 60) - 超时秒数
- [x] 定义输出结构
  - [x] stdout (string) - 标准输出
  - [x] stderr (string) - 标准错误
  - [x] exit_code (int) - 退出码
  - [x] duration_ms (int) - 执行时长 (毫秒)
- [x] 实现 Register() 函数
  - [x] 返回 ShellNode 实例
  - [x] 导出为 Go Plugin 符号

### Task 3: 实现命令执行逻辑 (AC1, AC3)
- [x] 解析输入参数
  - [x] 验证 command 非空
  - [x] 解析 args 数组
  - [x] 构建环境变量列表
  - [x] 验证 workdir 存在 (如果指定)
  - [x] 解析 timeout 值
- [x] 执行 Shell 命令
  - [x] 使用 `os/exec.CommandContext` 支持取消
  - [x] 设置命令参数: `sh -c "command"`
  - [x] 传递 args 到命令行
  - [x] 设置环境变量 (cmd.Env)
  - [x] 设置工作目录 (cmd.Dir)
  - [x] 创建 stdout/stderr 缓冲区
  - [x] 启动命令执行 (cmd.Start)
- [x] 超时控制
  - [x] 创建带超时的 context (context.WithTimeout)
  - [x] 超时时自动终止进程 (context 取消)
  - [x] 等待命令完成 (cmd.Wait)
- [x] 结果收集
  - [x] 捕获 stdout 输出
  - [x] 捕获 stderr 输出
  - [x] 获取退出码 (cmd.ProcessState.ExitCode())
  - [x] 记录执行时长
- [x] 错误处理 (AC3)
  - [x] 超时错误: 返回临时错误 (可重试)
  - [x] 命令不存在: 返回永久错误 (不可重试)
  - [x] Exit code ≠ 0: 包含 stdout/stderr 的错误
  - [x] 上下文取消: 清理子进程

### Task 4: 编写单元测试
- [x] 测试基本命令执行
  - [x] 成功执行: `echo hello`
  - [x] 验证 stdout 捕获
  - [x] 验证 exit_code = 0
- [x] 测试命令参数
  - [x] 命令带参数: `ls -la /tmp`
  - [x] 验证 args 正确传递
- [x] 测试环境变量
  - [x] 设置环境变量: `env: {KEY: "value"}`
  - [x] 验证命令可访问环境变量
- [x] 测试工作目录
  - [x] 设置 workdir: `/tmp`
  - [x] 验证命令在指定目录执行
- [x] 测试超时处理
  - [x] 长时间命令: `sleep 10` with timeout 1
  - [x] 验证超时终止
  - [x] 验证返回超时错误
- [x] 测试错误处理
  - [x] 命令不存在: `nonexistent-command`
  - [x] 命令失败: `false` (exit 1)
  - [x] 验证错误信息包含 stderr
- [x] 测试并发安全
  - [x] 多个 goroutine 并发调用 Execute
  - [x] 验证无 race condition
- [x] 测试覆盖率目标 >90% (实际: 95.2%)

### Task 5: 实现 Makefile 和编译脚本
- [x] 创建 Makefile 目标
  - [x] `make build` - 编译插件为 shell.so
    - [x] 检查 Go 版本 >= 1.22
    - [x] 检查 CGO_ENABLED=1
    - [x] 执行: `go build -buildmode=plugin -o shell.so`
  - [x] `make test` - 运行单元测试
    - [x] 执行: `go test -v -race -coverprofile=coverage.out`
    - [x] 显示覆盖率: `go tool cover -func=coverage.out`
  - [x] `make clean` - 清理构建产物
    - [x] 删除 shell.so
    - [x] 删除 coverage.out
  - [x] `make install` - 安装到插件目录
    - [x] 复制 shell.so 到 /opt/waterflow/plugins/
- [x] 添加依赖检查
  - [x] 检查 Go 是否安装
  - [x] 检查 CGO 是否启用
  - [x] 检查操作系统 (Linux/macOS)

### Task 6: 编写节点文档
- [x] 创建 README.md
  - [x] 节点描述和使用场景
  - [x] 参数说明 (类型、必需性、默认值)
  - [x] 输出说明
  - [x] 至少 3 个使用示例
    - [x] 基础命令: `echo hello`
    - [x] 带参数: `ls -la /tmp`
    - [x] 带环境变量和工作目录
  - [x] 错误处理说明
  - [x] 常见问题 FAQ
- [x] 添加 YAML 示例 (examples/workflows/shell-examples.yaml)

### Task 7: 集成测试
- [x] 创建 `plugins/exec/shell/integration_test.go`
- [x] 测试插件加载
  - [x] 编译插件为 .so 文件
  - [x] 使用 plugin.Open 加载
  - [x] 查找 Register 函数
  - [x] 调用 Register 获取节点实例
- [x] 测试与 NodeRegistry 集成
  - [x] 注册节点到 Registry
  - [x] 从 Registry 获取节点
  - [x] 执行节点并验证结果
- [x] 测试真实命令执行
  - [x] 系统命令: `whoami`, `pwd`, `date`
  - [x] 文件操作: `touch`, `rm`, `cat`
  - [x] 验证输出正确性

---

## Developer Context

### 架构背景

**依赖 Story:**
- **Story 3.1: 节点接口设计** - 定义了 Node 接口和插件机制
  - 必须先完成 Story 3.1 才能开始本 Story
  - 使用 Story 3.1 定义的 `pkg/node/interface.go`
  - 遵循 Story 3.1 的参数验证规范

**核心架构决策:**
- [ADR-0003: 插件化节点系统](../adr/0003-plugin-based-node-system.md) - Go Plugin 机制
- [ADR-0002: 单节点执行模式](../adr/0002-single-node-execution-pattern.md) - 每个 Step = 1 Activity

**Shell 节点定位:**
- **最基础的节点** - 所有其他节点的参考实现
- **最常用的节点** - 90% 的工作流会使用
- **质量标杆** - 其他节点参考此实现

### 技术栈和依赖

**Go 标准库:**
```go
import (
    "context"          // 上下文和取消
    "os/exec"          // 命令执行
    "time"             // 超时控制
    "bytes"            // 输出缓冲
    "fmt"              // 格式化
    "errors"           // 错误处理
    "plugin"           // Go Plugin (仅测试)
)
```

**项目依赖:**
```go
import (
    "github.com/websoft9/waterflow/pkg/dsl/node"  // Story 3.1 的 Node 接口
)
```

**测试依赖:**
```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

### 核心实现参考

#### Shell 节点结构
```go
// plugins/exec/shell/main.go
package main

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "time"
    "bytes"
    
    "github.com/websoft9/waterflow/pkg/dsl/node"
)

// ShellNode 实现 Shell 命令执行
type ShellNode struct{}

// Name 返回节点名称
func (n *ShellNode) Name() string {
    return "exec/shell"
}

// Version 返回节点版本
func (n *ShellNode) Version() string {
    return "v1"
}

// Params 返回参数规范（Story 1.3 已存在的方法）
func (n *ShellNode) Params() map[string]node.ParamSpec {
    return n.Metadata().InputSchema
}

// Metadata 返回节点元数据
func (n *ShellNode) Metadata() node.NodeMetadata {
    return node.NodeMetadata{
        Description: "Execute shell commands on the agent server",
        Category:    "exec",
        InputSchema: map[string]node.ParamSpec{
            "command": {
                Type:        "string",
                Required:    true,
                Description: "Shell command to execute",
            },
            "args": {
                Type:        "array",
                Required:    false,
                Description: "Command arguments",
            },
            "env": {
                Type:        "object",
                Required:    false,
                Description: "Environment variables",
            },
            "workdir": {
                Type:        "string",
                Required:    false,
                Description: "Working directory",
            },
            "timeout": {
                Type:        "int",
                Required:    false,
                Default:     60,
                MinValue:    1,
                MaxValue:    3600,
                Description: "Timeout in seconds",
            },
        },
        OutputSchema: map[string]interface{}{
            "stdout":      "string",
            "stderr":      "string",
            "exit_code":   "int",
            "duration_ms": "int",
        },
    }
}

// Execute 执行 Shell 命令（返回结构化 NodeResult）
func (n *ShellNode) Execute(
    ctx context.Context,
    inputs map[string]interface{},
) (*node.NodeResult, error) {
    startTime := time.Now()
    
    // 1. 解析参数
    command, ok := inputs["command"].(string)
    if !ok || command == "" {
        return nil, fmt.Errorf("command is required")
    }
    
    timeout := 60 // 默认超时
    if t, ok := inputs["timeout"].(int); ok {
        timeout = t
    }
    
    // 2. 创建带超时的 context
    cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
    defer cancel()
    
    // 3. 构建命令
    cmd := exec.CommandContext(cmdCtx, "sh", "-c", command)
    
    // 设置参数 (args)
    if args, ok := inputs["args"].([]interface{}); ok {
        strArgs := make([]string, len(args))
        for i, arg := range args {
            strArgs[i] = fmt.Sprintf("%v", arg)
        }
        cmd.Args = append(cmd.Args, strArgs...)
    }
    
    // 设置环境变量 (env)
    if env, ok := inputs["env"].(map[string]interface{}); ok {
        cmd.Env = os.Environ() // 继承当前环境
        for k, v := range env {
            cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", k, v))
        }
    }
    
    // 设置工作目录 (workdir)
    if workdir, ok := inputs["workdir"].(string); ok {
        cmd.Dir = workdir
    }
    
    // 4. 捕获输出
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // 5. 执行命令
    err := cmd.Run()
    
    duration := time.Since(startTime)
    
    // 6. 构建输出（使用 NodeResult 结构）
    exitCode := 0
    logs := []string{fmt.Sprintf("Executing: %s", command)}
    
    // 7. 处理错误
    if err != nil {
        // 获取退出码
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
            logs = append(logs, fmt.Sprintf("Command failed with exit code %d", exitCode))
            return &node.NodeResult{
                Outputs: map[string]interface{}{
                    "stdout":    stdout.String(),
                    "stderr":    stderr.String(),
                    "exit_code": exitCode,
                },
                Logs:     logs,
                Duration: duration,
            }, fmt.Errorf("command failed with exit code %d: %s", exitCode, stderr.String())
        }
        
        // 超时错误 (临时错误,可重试)
        if cmdCtx.Err() == context.DeadlineExceeded {
            logs = append(logs, fmt.Sprintf("Command timeout after %d seconds", timeout))
            return &node.NodeResult{
                Outputs: map[string]interface{}{
                    "stdout":    stdout.String(),
                    "stderr":    stderr.String(),
                    "exit_code": -1,
                },
                Logs:     logs,
                Duration: duration,
            }, fmt.Errorf("command timeout after %d seconds", timeout)
        }
        
        // 其他错误 (永久错误,不可重试)
        logs = append(logs, fmt.Sprintf("Command execution failed: %v", err))
        return &node.NodeResult{
            Outputs: map[string]interface{}{
                "stdout":    stdout.String(),
                "stderr":    stderr.String(),
                "exit_code": -1,
            },
            Logs:     logs,
            Duration: duration,
        }, fmt.Errorf("command execution failed: %w", err)
    }
    
    // 成功执行
    logs = append(logs, "Command completed successfully")
    return &node.NodeResult{
        Outputs: map[string]interface{}{
            "stdout":    stdout.String(),
            "stderr":    stderr.String(),
            "exit_code": 0,
        },
        Logs:     logs,
        Duration: duration,
    }, nil
}

// Register 是 Go Plugin 导出的注册函数
func Register() node.Node {
    return &ShellNode{}
}
```

#### 关键实现要点

**参数验证:**
- 必需参数检查: command 非空
- 类型验证: args 为数组, timeout 为整数
- 范围验证: timeout 在 1-3600 秒之间
- 详细错误信息（参考上面核心实现）

**超时控制:**
- 使用 `context.WithTimeout` 创建超时上下文
- 使用 `exec.CommandContext` 支持取消
- 超时时自动终止子进程
- 区分超时错误（临时）和其他错误（永久）

**输出结构:**
- 使用 `NodeResult` 结构化返回
- `Outputs`: 包含 stdout, stderr, exit_code
- `Logs`: 执行过程日志
- `Duration`: 实际执行时长

### 错误处理策略

**临时错误 (可重试):**
- 超时: `context.DeadlineExceeded`
- 上下文取消: `context.Canceled`
- 资源暂时不可用

**永久错误 (不可重试):**
- 参数错误: command 为空
- 命令不存在: `exec: "nonexistent": executable file not found`
- 工作目录不存在: `chdir /invalid: no such file or directory`
- 权限错误: `permission denied`

**Exit Code ≠ 0:**
- 不算"错误",但返回 error 以中止工作流
- 包含 stdout 和 stderr 在错误信息中
- 允许用户通过 `continue-on-error: true` 忽略

### 测试策略

#### 单元测试
```go
// plugins/exec/shell/main_test.go
package main

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestShellNode_Name(t *testing.T) {
    node := &ShellNode{}
    assert.Equal(t, "exec/shell", node.Name())
}

func TestShellNode_Version(t *testing.T) {
    node := &ShellNode{}
    assert.Equal(t, "v1", node.Version())
}

func TestShellNode_Execute_BasicCommand(t *testing.T) {
    node := &ShellNode{}
    
    inputs := map[string]interface{}{
        "command": "echo hello",
    }
    
    result, err := node.Execute(context.Background(), inputs)
    
    require.NoError(t, err)
    require.NotNil(t, result)
    assert.Equal(t, "hello\n", result.Outputs["stdout"])
    assert.Equal(t, "", result.Outputs["stderr"])
    assert.Equal(t, 0, result.Outputs["exit_code"])
    assert.Greater(t, result.Duration, time.Duration(0))
    assert.NotEmpty(t, result.Logs)
}

func TestShellNode_Execute_WithArgs(t *testing.T) {
    node := &ShellNode{}
    
    inputs := map[string]interface{}{
        "command": "ls",
        "args":    []interface{}{"-la", "/tmp"},
    }
    
    result, err := node.Execute(context.Background(), inputs)
    
    require.NoError(t, err)
    assert.Contains(t, result.Outputs["stdout"], "total")
    assert.Equal(t, 0, result.Outputs["exit_code"])
}

func TestShellNode_Execute_Timeout(t *testing.T) {
    node := &ShellNode{}
    
    inputs := map[string]interface{}{
        "command": "sleep 10",
        "timeout": 1, // 1秒超时
    }
    
    start := time.Now()
    result, err := node.Execute(context.Background(), inputs)
    duration := time.Since(start)
    
    require.Error(t, err)
    assert.Contains(t, err.Error(), "timeout")
    assert.Less(t, duration, 2*time.Second)
    assert.NotNil(t, result) // 即使错误也返回 result
    assert.Equal(t, -1, result.Outputs["exit_code"])
}

func TestShellNode_Execute_CommandFailed(t *testing.T) {
    node := &ShellNode{}
    
    inputs := map[string]interface{}{
        "command": "false", // 总是返回 exit 1
    }
    
    result, err := node.Execute(context.Background(), inputs)
    
    require.Error(t, err)
    assert.Contains(t, err.Error(), "exit code 1")
    assert.Equal(t, 1, result.Outputs["exit_code"])
}

func TestShellNode_Execute_ConcurrentSafe(t *testing.T) {
    node := &ShellNode{}
    
    // 并发执行多次
    done := make(chan bool, 10)
    for i := 0; i < 10; i++ {
        go func() {
            inputs := map[string]interface{}{
                "command": "echo test",
            }
            _, err := node.Execute(context.Background(), inputs)
            assert.NoError(t, err)
            done <- true
        }()
    }
    
    // 等待所有完成
    for i := 0; i < 10; i++ {
        <-done
    }
}
```

#### 集成测试
```go
// plugins/exec/shell/integration_test.go
// +build integration

package main

import (
    "context"
    "plugin"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/websoft9/waterflow/pkg/dsl/node"
)

func TestShellPlugin_Load(t *testing.T) {
    // 1. 加载插件
    p, err := plugin.Open("shell.so")
    require.NoError(t, err, "Failed to load plugin")
    
    // 2. 查找 Register 函数
    registerSymbol, err := p.Lookup("Register")
    require.NoError(t, err, "Register function not found")
    
    // 3. 调用 Register
    register, ok := registerSymbol.(func() node.Node)
    require.True(t, ok, "Register has wrong signature")
    
    nodeInstance := register()
    require.NotNil(t, nodeInstance)
    
    // 4. 验证节点属性
    assert.Equal(t, "exec/shell", nodeInstance.Name())
    assert.Equal(t, "v1", nodeInstance.Version())
}

func TestShellPlugin_Execute(t *testing.T) {
    // 加载插件
    p, _ := plugin.Open("shell.so")
    registerSymbol, _ := p.Lookup("Register")
    register := registerSymbol.(func() node.Node)
    nodeInstance := register()
    
    // 执行命令
    inputs := map[string]interface{}{
        "command": "whoami",
    }
    
    result, err := nodeInstance.Execute(context.Background(), inputs)
    
    require.NoError(t, err)
    assert.NotEmpty(t, result.Outputs["stdout"])
    assert.Equal(t, 0, result.Outputs["exit_code"])
}
```

### 性能考虑

**执行性能:**
- 命令启动延迟: ~1-5ms (取决于系统)
- 输出缓冲: 自动增长 (bytes.Buffer)
- 大输出场景: stdout/stderr > 1MB 时考虑流式处理

**内存占用:**
- ShellNode 实例: ~100 bytes
- 每次执行的缓冲区: ~输出大小 + 开销

**并发安全:**
- ShellNode 无状态,天然线程安全
- 每次 Execute 独立的 cmd 实例
- 支持无限制并发调用

### 文件结构

```
Waterflow/
├── plugins/
│   └── exec/
│       └── shell/                    # [NEW] Shell 节点插件
│           ├── main.go               # [NEW] 插件实现
│           ├── main_test.go          # [NEW] 单元测试
│           ├── integration_test.go   # [NEW] 集成测试
│           ├── Makefile              # [NEW] 编译脚本
│           └── README.md             # [NEW] 节点文档
├── pkg/
│   └── dsl/
│       └── node/                     # Story 3.1 定义的包
│           ├── registry.go           # [EXISTS] Story 1.3
│           ├── interface.go          # [EXISTS] Story 3.1 - Node 接口扩展
│           ├── result.go             # [EXISTS] Story 3.1 - NodeResult
│           ├── metadata.go           # [EXISTS] Story 3.1 - NodeMetadata
│           └── errors.go             # [EXISTS] Story 3.1 - 错误类型
└── examples/
    └── workflows/
        └── shell-examples.yaml       # [NEW] Shell 节点使用示例
```

### 编译和安装

#### Makefile 示例
```makefile
# plugins/exec/shell/Makefile
.PHONY: build test clean install

PLUGIN_NAME = shell.so
GO_VERSION_MIN = 1.22

build:
	@echo "Checking Go version..."
	@go version | grep -q "go1.2[2-9]" || (echo "Error: Go 1.22+ required" && exit 1)
	@echo "Checking CGO..."
	@test "$${CGO_ENABLED}" = "1" || (echo "Error: CGO_ENABLED=1 required" && exit 1)
	@echo "Building plugin..."
	CGO_ENABLED=1 go build -buildmode=plugin -o $(PLUGIN_NAME) main.go
	@echo "✓ Plugin built: $(PLUGIN_NAME)"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out
	@echo "\n=== Coverage Report ==="
	go tool cover -func=coverage.out
	@echo "✓ Tests passed"

integration-test: build
	@echo "Running integration tests..."
	go test -v -tags=integration -run TestShellPlugin
	@echo "✓ Integration tests passed"

clean:
	rm -f $(PLUGIN_NAME) coverage.out
	@echo "✓ Cleaned"

install: build
	@echo "Installing plugin..."
	mkdir -p /opt/waterflow/plugins
	cp $(PLUGIN_NAME) /opt/waterflow/plugins/
	@echo "✓ Installed to /opt/waterflow/plugins/$(PLUGIN_NAME)"

.DEFAULT_GOAL := build
```

#### 使用示例
```bash
# 编译插件
cd plugins/exec/shell
make build

# 运行测试
make test

# 集成测试
make integration-test

# 安装到 Agent
make install
```

### YAML 工作流示例

```yaml
# examples/workflows/shell-examples.yaml
name: Shell Node Examples

jobs:
  basic-commands:
    name: Basic Shell Commands
    runs-on: linux-amd64
    steps:
      - name: Simple echo
        uses: exec/shell@v1
        with:
          command: echo "Hello Waterflow"
      
      - name: Check system info
        uses: exec/shell@v1
        with:
          command: uname -a
      
      - name: List files
        uses: exec/shell@v1
        with:
          command: ls
          args: ["-la", "/tmp"]
          timeout: 10

  environment-variables:
    name: Environment Variables
    runs-on: linux-amd64
    steps:
      - name: Use custom env
        uses: exec/shell@v1
        with:
          command: echo "API_KEY=$API_KEY"
          env:
            API_KEY: "secret-key-123"
            ENV: "production"

  working-directory:
    name: Working Directory
    runs-on: linux-amd64
    steps:
      - name: Create temp file
        uses: exec/shell@v1
        with:
          command: touch test.txt
          workdir: /tmp
      
      - name: Verify file exists
        uses: exec/shell@v1
        with:
          command: ls test.txt
          workdir: /tmp

  error-handling:
    name: Error Handling
    runs-on: linux-amd64
    steps:
      - name: Timeout example
        uses: exec/shell@v1
        with:
          command: sleep 100
          timeout: 5
        continue-on-error: true
      
      - name: Command failure
        uses: exec/shell@v1
        with:
          command: exit 1
        continue-on-error: true
      
      - name: This step runs anyway
        uses: exec/shell@v1
        with:
          command: echo "Execution continues"
```

### Epic 2 经验应用

**来自 Epic 2 Retrospective 的关键教训:**

1. **完整的错误处理** ✅
   - 区分临时错误和永久错误
   - 提供详细的错误信息 (包含 stdout/stderr)
   - 超时正确清理子进程

2. **测试覆盖全面** ✅
   - 单元测试 >90% 覆盖率
   - 集成测试验证插件加载
   - 并发安全测试

3. **文档与代码对应** ✅
   - README 中的示例可直接运行
   - Makefile 命令经过验证
   - YAML 示例与实际实现对应

4. **依赖检查完整** ✅
   - Makefile 检查 Go 版本
   - Makefile 检查 CGO_ENABLED
   - 明确说明平台限制 (Linux/macOS)

### 开发顺序建议

**阶段 1: 基础实现 (Day 1)**
1. 创建目录结构和文件
2. 实现 ShellNode 结构体和接口方法
3. 实现基础的 Execute 逻辑 (不含超时和错误处理)
4. 基础单元测试 (echo hello)

**阶段 2: 完整功能 (Day 1-2)**
1. 添加参数解析 (args, env, workdir, timeout)
2. 实现超时控制
3. 完善错误处理
4. 扩展单元测试覆盖所有参数

**阶段 3: 测试和文档 (Day 2)**
1. 实现 Makefile
2. 编写集成测试
3. 编写 README 文档
4. 创建 YAML 示例

**阶段 4: 验证 (Day 2-3)**
1. 编译为 .so 插件
2. 测试插件加载
3. 运行所有测试
4. 验证覆盖率 >90%

**总估算: 2-3 工作日**

### 验收标准检查清单

- [ ] **AC1: 基础命令执行**
  - [ ] 节点编译为 shell.so
  - [ ] 实现 Node 接口所有 5 个方法（Name, Version, Params, Execute, Metadata）
  - [ ] Execute 返回 *NodeResult 结构
  - [ ] 捕获 stdout 和 stderr
  - [ ] 返回正确的 exit_code

- [ ] **AC2: 参数支持**
  - [ ] command (必需)
  - [ ] args (可选,数组)
  - [ ] env (可选,对象)
  - [ ] workdir (可选,字符串)
  - [ ] timeout (可选,整数,默认60)

- [ ] **AC3: 错误处理**
  - [ ] 超时自动终止进程
  - [ ] exit code ≠ 0 抛出错误
  - [ ] 错误信息包含 stdout/stderr
  - [ ] 区分临时错误和永久错误

- [ ] **代码质量**
  - [ ] 单元测试覆盖率 >90%
  - [ ] 集成测试通过
  - [ ] 无 race condition
  - [ ] golangci-lint 无错误

- [ ] **文档完整性**
  - [ ] README.md 清晰易懂
  - [ ] 包含至少 3 个示例
  - [ ] Makefile 所有命令可执行
  - [ ] YAML 示例可运行

- [ ] **插件机制**
  - [ ] 可编译为 .so 文件
  - [ ] plugin.Open 可加载
  - [ ] Register 函数可调用
  - [ ] 返回有效的 Node 实例

---

## References

**架构决策:**
- [ADR-0003: 插件化节点系统](../adr/0003-plugin-based-node-system.md)
- [ADR-0002: 单节点执行模式](../adr/0002-single-node-execution-pattern.md)

**依赖 Stories:**
- [Story 3.1: 节点接口设计](./3-1-node-interface-design.md) - Node 接口定义 (`pkg/dsl/node`)
- [Story 1.3: YAML DSL 解析和验证](./1-3-yaml-dsl-parsing-and-validation.md) - NodeRegistry 和 ParamSpec

**后续 Stories:**
- [Story 3.3: 脚本文件执行节点](../epics.md#story-33-脚本文件执行节点) - 基于 Shell 节点扩展

**技术文档:**
- [Go os/exec Package](https://pkg.go.dev/os/exec)
- [Go Plugin Package](https://pkg.go.dev/plugin)
- [Go Context Package](https://pkg.go.dev/context)

---

## Dev Agent Record

### Context Reference
- Epic 3 定义: [docs/epics.md](../epics.md#epic-3-核心节点插件库)
- Story 3.1 接口: [docs/sprint-artifacts/3-1-node-interface-design.md](./3-1-node-interface-design.md)

### Agent Model Used
Claude Sonnet 4.5

### Implementation Plan
1. 创建目录结构和基础文件框架
2. 实现 ShellNode 结构体和 5 个接口方法
3. 实现命令执行逻辑（参数解析、超时控制、错误处理）
4. 编写全面的单元测试达到 >90% 覆盖率
5. 实现 Makefile 构建脚本
6. 编写完整的 README 文档
7. 创建集成测试验证插件加载和执行

### Completion Notes
- [x] 所有 AC 已实现并验证
- [x] 单元测试通过 (覆盖率 95.2% > 90% ✅)
- [x] 集成测试通过 (4个测试场景全部通过 ✅)
- [x] 插件可成功编译和加载 (shell.so 4.9MB ✅)
- [x] 文档已完成 (README + YAML 示例 ✅)
- [x] 代码审查完成 (2025-12-30)
  - [x] 修复 args 参数命令注入风险 (shell 转义)
  - [x] 修复 env 值清理问题 (移除换行符和空字节)
  - [x] 降低 Execute 方法复杂度 (提取 parseInputs 和 buildCommand)
  - [x] 添加 README 安全警告章节
  - [x] Gosec 警告已记录并文档化
  - [x] 所有文件已提交到 Git

### File List
**新增文件:**
- `plugins/exec/shell/main.go` - Shell 节点实现 (216行)
- `plugins/exec/shell/main_test.go` - 单元测试 (21个测试用例)
- `plugins/exec/shell/integration_test.go` - 集成测试 (4个测试场景)
- `plugins/exec/shell/Makefile` - 编译脚本 (6个目标)
- `plugins/exec/shell/README.md` - 节点文档
- `plugins/exec/shell/go.mod` - Go 模块文件
- `examples/workflows/shell-examples.yaml` - YAML 示例

**依赖文件 (Story 3.1):**
- `pkg/dsl/node/interface.go` - Node 接口（扩展）
- `pkg/dsl/node/result.go` - NodeResult 结构
- `pkg/dsl/node/metadata.go` - NodeMetadata
- `pkg/dsl/node/errors.go` - 错误类型
- `pkg/dsl/node/registry.go` - ParamSpec (Story 1.3)

---

**Story 准备完成！开发者现在拥有创建 Shell 节点所需的所有上下文！** 🎯
