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

### 节点插件接口:

```go
// pkg/node/interface.go
type Node interface {
    Execute(ctx context.Context, args map[string]interface{}) (NodeResult, error)
}

type NodeResult struct {
    Outputs map[string]string
    Logs    []string
}
```

### 插件实现:

```go
// plugins/checkout/main.go
package main

import (
    "context"
    "waterflow/pkg/node"
)

type CheckoutNode struct{}

func (n *CheckoutNode) Execute(ctx context.Context, args map[string]interface{}) (node.NodeResult, error) {
    repo := args["repository"].(string)
    
    // Git checkout 逻辑
    return node.NodeResult{
        Outputs: map[string]string{
            "commit": "abc123",
        },
    }, nil
}

// 插件注册函数
func Register() node.Node {
    return &CheckoutNode{}
}
```

### 插件管理器:

```go
// pkg/agent/plugin_manager.go
type PluginManager struct {
    plugins map[string]plugin.Plugin
    nodes   map[string]node.Node
}

func (pm *PluginManager) LoadPlugin(path string) error {
    // 加载 .so 文件
    p, err := plugin.Open(path)
    if err != nil {
        return err
    }
    
    // 查找 Register 函数
    symbol, err := p.Lookup("Register")
    if err != nil {
        return err
    }
    
    // 调用 Register 获取 Node 实例
    register := symbol.(func() node.Node)
    nodeInstance := register()
    
    // 注册到 NodeRegistry
    pm.nodes[name] = nodeInstance
    return nil
}

func (pm *PluginManager) GetNode(nodeType string) (node.Node, error) {
    node, ok := pm.nodes[nodeType]
    if !ok {
        return nil, fmt.Errorf("node not found: %s", nodeType)
    }
    return node, nil
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
