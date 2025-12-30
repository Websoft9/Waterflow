# ADR-0003: 插件化节点系统

**状态:** ✅ 已采纳  
**日期:** 2025-12-15  
**决策者:** 架构团队  

## 背景

Waterflow 需要支持多种类型的节点(Node):
- 内置节点: `checkout`, `run`, `cache`, `artifact`
- 第三方节点: 社区贡献的节点
- 私有节点: 企业内部定制节点

需要决定节点的实现方式:
1. **内置编译** - 所有节点硬编码在 Agent 二进制中
2. **插件系统** - 节点作为独立的 `.so` 动态库加载
3. **RPC 调用** - 节点作为独立进程,通过 RPC 调用

## 决策

采用 **插件系统**:所有节点(包括内置节点)都作为 Go Plugin (`.so` 文件)实现。

## 理由

### 核心优势:

1. **扩展性**
   - 第三方可以开发和发布节点插件
   - 不需要重新编译 Agent 即可添加新节点
   - 支持热加载(Agent 运行时加载新插件)

2. **隔离性**
   - 插件崩溃不影响 Agent 核心
   - 内存隔离(一定程度)
   - 版本独立(节点可以有自己的依赖)

3. **统一机制**
   - 内置节点和第三方节点使用相同的加载机制
   - 简化架构,无特殊处理

4. **动态更新**
   - 更新节点只需替换 `.so` 文件
   - 不需要重启 Agent(热加载)

### 与其他方案对比:

| 方案 | 优点 | 缺点 | 决策 |
|------|------|------|------|
| **插件系统** | 可扩展,热加载,隔离 | Go Plugin 限制多 | ✅ 选择 |
| 内置编译 | 简单,性能好 | 不可扩展,每次更新需重新编译 | ❌ |
| RPC 进程 | 完全隔离,跨语言 | 性能开销大,部署复杂 | ❌ |

## 后果

### 正面影响:

✅ **可扩展** - 第三方可以开发节点插件  
✅ **热加载** - 运行时更新节点实现  
✅ **统一机制** - 所有节点同等对待  
✅ **版本管理** - 节点版本独立于 Agent  

### 负面影响:

⚠️ **Go Plugin 限制**
   - 只支持 Linux/macOS (不支持 Windows)
   - Go 版本必须匹配(编译 Plugin 和 Agent 的 Go 版本)
   - CGO 必须启用

⚠️ **调试困难**
   - 插件崩溃难以定位
   - 无法在插件中使用 delve 调试器

### 风险缓解:

- **跨平台**: Windows 使用内置编译模式(fallback)
- **版本管理**: 严格规定 Plugin 编译环境
- **调试**: 提供 `--debug-plugin` 模式,将插件代码内联编译

## 实现示例

### 节点插件接口 (Story 3.1 实际实现):

```go
// pkg/dsl/node/interface.go
package node

import (
    "context"
    "time"
)

// Node 接口 - 所有自定义节点必须实现
type Node interface {
    // 基础信息 (Story 1.3)
    Name() string
    Version() string
    Params() map[string]ParamSpec
    
    // 执行和元数据 (Story 3.1)
    Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error)
    Metadata() NodeMetadata
}

// 节点执行结果
type NodeResult struct {
    Outputs  map[string]interface{} // 输出数据
    Logs     []string               // 执行日志
    Duration time.Duration          // 执行耗时
    Metadata map[string]interface{} // 扩展元数据
}

// 节点元数据（用于文档和验证）
type NodeMetadata struct {
    Description  string                  // 节点描述
    Category     string                  // 分类 (exec/docker/http/file/flow)
    InputSchema  map[string]ParamSpec    // 输入参数 Schema
    OutputSchema map[string]interface{}  // 输出结构定义
}

// 参数规格（支持高级验证）
type ParamSpec struct {
    Type        string        // 参数类型
    Required    bool          // 是否必需
    Description string        // 参数说明
    Default     interface{}   // 默认值
    Pattern     string        // 正则表达式验证
    Enum        []interface{} // 枚举值验证 (Story 3.1 新增)
    MinValue    *float64      // 最小值（数值类型，Story 3.1 新增）
    MaxValue    *float64      // 最大值（数值类型，Story 3.1 新增）
}

// 插件注册函数签名
type RegisterFunc func() Node
```

### 插件实现 (实际示例):

```go
// examples/plugins/echo/main.go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/Websoft9/waterflow/pkg/dsl/node"
)

// EchoNode 实现 Node 接口
type EchoNode struct{}

// Name 返回节点名称
func (n *EchoNode) Name() string {
    return "exec/echo"
}

// Version 返回节点版本
func (n *EchoNode) Version() string {
    return "v1.0.0"
}

// Params 返回参数规格
func (n *EchoNode) Params() map[string]node.ParamSpec {
    return map[string]node.ParamSpec{
        "message": {
            Type:        "string",
            Required:    true,
            Description: "Message to echo",
        },
    }
}

// Execute 执行节点逻辑
func (n *EchoNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
    startTime := time.Now()
    
    // 获取输入参数
    message, ok := inputs["message"].(string)
    if !ok {
        return nil, fmt.Errorf("message must be string")
    }
    
    // 创建结果
    result := node.NewNodeResult()
    result.AddLog(fmt.Sprintf("Echoing: %s", message))
    result.SetOutput("echo", message)
    result.Duration = time.Since(startTime)
    
    return result, nil
}

// Metadata 返回节点元数据
func (n *EchoNode) Metadata() node.NodeMetadata {
    return node.NewNodeMetadata(
        "Echo a message",
        "exec",
        map[string]node.ParamSpec{
            "message": {
                Type:        "string",
                Required:    true,
                Description: "Message to echo",
            },
        },
        map[string]interface{}{
            "echo": "string",
        },
    )
}

// Register 是插件必须导出的注册函数
func Register() node.Node {
    return &EchoNode{}
}
```

**编译插件:**
```bash
cd examples/plugins/echo
export CGO_ENABLED=1
go build -buildmode=plugin -o echo.so main.go
```

### 插件管理器 (实际实现):

```go
// internal/agent/plugin_manager.go
package agent

import (
    "fmt"
    "path/filepath"
    "plugin"
    "strings"
    "go.uber.org/zap"
    "github.com/Websoft9/waterflow/pkg/dsl/node"
)

type PluginManager struct {
    logger   *zap.Logger
    plugins  map[string]*plugin.Plugin
    registry *node.NodeRegistry
}

func NewPluginManager(logger *zap.Logger, registry *node.NodeRegistry) *PluginManager {
    return &PluginManager{
        logger:   logger,
        plugins:  make(map[string]*plugin.Plugin),
        registry: registry,
    }
}

func (pm *PluginManager) LoadPlugin(path string) error {
    // 1. 加载 .so 文件
    p, err := plugin.Open(path)
    if err != nil {
        return &node.PluginLoadError{PluginPath: path, Cause: err}
    }
    
    // 2. 查找 Register 函数
    symbol, err := p.Lookup("Register")
    if err != nil {
        return &node.RegisterFunctionNotFoundError{PluginPath: path}
    }
    
    // 3. 调用 Register 获取 Node 实例
    register, ok := symbol.(func() node.Node)
    if !ok {
        return &node.RegisterFunctionSignatureError{PluginPath: path}
    }
    
    nodeInstance := register()
    
    // 4. 验证节点
    if err := node.ValidateNode(nodeInstance); err != nil {
        return err
    }
    
    // 5. 注册到 NodeRegistry
    if err := pm.registry.Register(nodeInstance); err != nil {
        return err
    }
    
    pm.logger.Info("Plugin loaded successfully",
        zap.String("path", path),
        zap.String("node", nodeInstance.Name()),
    )
    
    return nil
}

func (pm *PluginManager) GetNode(nodeType string) (node.Node, error) {
    return pm.registry.Get(nodeType)
}
```

**插件错误类型 (Story 3.1):**
```go
// pkg/dsl/node/errors.go
type PluginNotFoundError struct {
    PluginPath string
}

type PluginLoadError struct {
    PluginPath string
    Cause      error
}

type PluginVersionMismatchError struct {
    PluginPath     string
    ExpectedGoVersion string
    ActualGoVersion   string
}

type RegisterFunctionNotFoundError struct {
    PluginPath string
}

type RegisterFunctionSignatureError struct {
    PluginPath string
}

type InvalidNodeError struct {
    NodeName string
    Reason   string
}

type NodeValidationError struct {
    NodeName string
    Errors   []string
}
```

### Agent 启动时自动加载:

```go
// pkg/agent/worker.go
func (w *Worker) Start() error {
    // 扫描插件目录
    pluginDir := "/opt/waterflow/plugins"
    files, _ := os.ReadDir(pluginDir)
    
    for _, file := range files {
        if strings.HasSuffix(file.Name(), ".so") {
            pluginPath := filepath.Join(pluginDir, file.Name())
            if err := w.pluginManager.LoadPlugin(pluginPath); err != nil {
                log.Warnf("Failed to load plugin %s: %v", file.Name(), err)
            }
        }
    }
    
    // 启动 Temporal Worker
    return w.temporalWorker.Start()
}
```

## 插件发布机制

### 目录结构:

```
/opt/waterflow/plugins/
  ├── checkout.so        # 内置插件
  ├── run.so
  ├── cache.so
  └── custom/
      └── slack-notify.so  # 第三方插件
```

### 热加载:

```go
// Agent 监听插件目录变化
watcher, _ := fsnotify.NewWatcher()
watcher.Add(pluginDir)

for {
    select {
    case event := <-watcher.Events:
        if event.Op&fsnotify.Write == fsnotify.Write {
            // 重新加载插件
            pm.ReloadPlugin(event.Name)
        }
    }
}
```

## 最佳实践 (Story 3.1 总结)

### 1. 开发环境要求

**必需条件:**
- Go 版本: 与 Agent 完全一致 (当前 1.22.0+)
- CGO_ENABLED=1 (Go Plugin 强制要求)
- GOOS: linux 或 darwin (不支持 Windows)
- GOARCH: amd64 或 arm64

**验证命令:**
```bash
# 检查 Go 版本
go version  # 必须与 Agent 一致

# 检查 CGO
go env CGO_ENABLED  # 必须为 1

# 检查平台
go env GOOS GOARCH
```

### 2. 节点实现规范

**5个必需方法:**
```go
type Node interface {
    Name() string                     // 格式: category/name (如 exec/echo)
    Version() string                  // 格式: vX.Y.Z (如 v1.0.0)
    Params() map[string]ParamSpec     // 参数规格
    Execute(ctx, inputs) (*Result, error)  // 执行逻辑
    Metadata() NodeMetadata           // 节点元数据
}
```

**参数验证:**
- 使用 ParamSpec 定义参数类型和验证规则
- 支持 Required, Pattern, Enum, MinValue, MaxValue
- Execute 方法应进行额外验证
- 使用 ValidateInputs() 辅助函数

**错误处理:**
- 返回明确的错误信息
- 使用 context 支持超时和取消
- 关键操作添加日志到 NodeResult.Logs

### 3. 测试要求

**单元测试 (必需):**
```go
func TestEchoNode_Execute(t *testing.T) {
    node := &EchoNode{}
    result, err := node.Execute(context.Background(), map[string]interface{}{
        "message": "test",
    })
    assert.NoError(t, err)
    assert.Equal(t, "test", result.Outputs["echo"])
}
```

**覆盖率目标:** >80%

**性能基准 (建议):**
```go
func BenchmarkEchoNode_Execute(b *testing.B) {
    node := &EchoNode{}
    inputs := map[string]interface{}{"message": "test"}
    for i := 0; i < b.N; i++ {
        node.Execute(context.Background(), inputs)
    }
}
```

### 4. 编译和部署

**编译命令:**
```bash
go build -buildmode=plugin -o mynode.so main.go
```

**部署:**
```bash
# 复制到 Agent 插件目录
cp mynode.so /opt/waterflow/plugins/

# 重启 Agent 或等待热加载
systemctl restart waterflow-agent
```

**文件大小:** 典型插件 3-10 MB (由于 Go runtime 开销)

### 5. 常见问题

**问题 1: "plugin was built with a different version of package"**
- 原因: Go 版本不匹配
- 解决: 使用与 Agent 相同的 Go 版本编译

**问题 2: "plugin.Open: plugin.so: undefined symbol"**
- 原因: 依赖库不一致
- 解决: 确保 go.mod 与 Agent 一致

**问题 3: "plugin: not a Go plugin"**
- 原因: CGO_ENABLED=0
- 解决: 设置 `export CGO_ENABLED=1`

**问题 4: 插件无法在 Windows 编译**
- 原因: Go Plugin 不支持 Windows
- 解决: 使用 Linux/macOS 开发，或使用 WSL2

### 6. 开发工作流

1. **开发阶段:**
   ```bash
   cd examples/plugins/template
   make check  # 验证环境
   make build  # 编译插件
   make test   # 运行测试
   ```

2. **测试阶段:**
   - 在本地 Agent 环境加载插件
   - 使用 YAML workflow 测试节点功能
   - 检查日志和输出结果

3. **发布阶段:**
   - 确保测试覆盖率 >80%
   - 编写 README.md 说明使用方法
   - 发布 .so 文件和文档

### 7. 性能考虑

- **插件加载:** 每个插件约 10-50ms 加载时间
- **执行开销:** Go Plugin 调用开销 <1μs (可忽略)
- **内存占用:** 每个插件约 5-20MB (共享 Go runtime)
- **并发安全:** 确保 Node 实例是线程安全的

**推荐架构:**
```go
type MyNode struct {
    mu    sync.Mutex  // 保护共享状态
    cache map[string]interface{}
}

func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*NodeResult, error) {
    n.mu.Lock()
    defer n.mu.Unlock()
    // 执行逻辑
}
```

## 替代方案

### 方案 A: 内置编译 (被拒绝)

所有节点硬编码在 Agent 中:

```go
// Agent 中直接引用
import (
    "waterflow/nodes/checkout"
    "waterflow/nodes/run"
)

func (w *Worker) ExecuteNode(nodeType string) {
    switch nodeType {
    case "checkout":
        return checkout.Execute(ctx, args)
    case "run":
        return run.Execute(ctx, args)
    }
}
```

**被拒绝原因:**
- ❌ 添加新节点需要重新编译 Agent
- ❌ 无法支持第三方节点
- ❌ 更新节点需要更新整个 Agent

### 方案 B: RPC 独立进程 (被拒绝)

每个节点作为独立进程,通过 gRPC 调用:

```protobuf
service NodeService {
  rpc Execute(NodeRequest) returns (NodeResponse);
}
```

**被拒绝原因:**
- ❌ 进程间通信开销大
- ❌ 部署复杂(每个节点一个二进制)
- ❌ 资源占用高(每个节点一个进程)

### 方案 C: WASM 插件 (考虑但未采纳)

使用 WebAssembly 作为插件格式:

**未采纳原因:**
- Go 的 WASM 支持不成熟
- 性能不如 Native Plugin
- 调试更加困难
- **可能在未来采纳**(如果 Go WASM 成熟)

## 参考资料

- [Go Plugin 官方文档](https://pkg.go.dev/plugin)
- [Agent 架构设计](../analysis/agent-architecture.md)
- [Epic 3: 节点系统](../epics.md#epic-3-节点系统)
