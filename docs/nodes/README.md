# Waterflow 核心节点库

Waterflow 提供了 7 个核心节点，覆盖命令执行、流程控制、网络通信、文件操作、容器管理等常见场景。

## 快速参考

| 节点 | 版本 | 分类 | 简介 | 常用场景 |
|------|------|------|------|----------|
| [exec/shell](exec/shell.md) | v1 | 执行 | Shell 命令执行 | 系统命令、工具调用、脚本执行 |
| [exec/script](exec/script.md) | v1 | 执行 | 脚本文件执行 | 复杂脚本、多语言支持、批处理 |
| [flow/sleep](flow/sleep.md) | v1 | 流程 | 延迟等待 | 服务启动等待、API 限流、健康检查间隔 |
| [http/request](http/request.md) | v1 | 网络 | HTTP 请求 | REST API 调用、Webhook 通知、数据同步 |
| [file/transfer](file/transfer.md) | v1 | 文件 | 文件传输 (SFTP/SCP) | 配置分发、日志收集、备份传输 |
| [docker/exec](docker/exec.md) | v1 | 容器 | Docker 命令执行 | 容器管理、镜像操作、容器监控 |
| [docker/compose](docker/compose.md) | v1 | 容器 | Compose 栈管理 | 多容器部署、应用栈管理、环境管理 |

## 按分类浏览

### 执行类 (exec)

命令和脚本执行节点，用于在 Agent 服务器上执行系统命令和脚本。

- **[exec/shell](exec/shell.md)** - 执行 Shell 命令
  - 用途: 运行系统命令、调用 CLI 工具、文件操作
  - 典型场景: `ls`, `git`, `kubectl`, `aws`
  
- **[exec/script](exec/script.md)** - 执行脚本文件
  - 用途: 执行 Bash/Python/Node 等脚本
  - 典型场景: 复杂逻辑、多行脚本、多语言支持

### 流程控制类 (flow)

工作流流程控制节点，用于管理工作流执行流程。

- **[flow/sleep](flow/sleep.md)** - 延迟等待
  - 用途: 在步骤间添加延迟
  - 典型场景: 等待服务启动、API 限流、健康检查间隔

### 网络通信类 (http)

HTTP/REST API 集成节点，用于与外部系统通信。

- **[http/request](http/request.md)** - 发送 HTTP 请求
  - 用途: 调用 REST API、发送 Webhook
  - 典型场景: API 集成、数据同步、通知推送

### 文件操作类 (file)

文件传输和操作节点，用于服务器间文件管理。

- **[file/transfer](file/transfer.md)** - SFTP/SCP 文件传输
  - 用途: 服务器间文件上传/下载
  - 典型场景: 配置分发、日志收集、部署资产

### 容器管理类 (docker)

Docker 容器和编排节点，用于容器化应用管理。

- **[docker/exec](docker/exec.md)** - 执行 Docker CLI 命令
  - 用途: 管理容器和镜像
  - 典型场景: 容器启停、镜像拉取、日志查看
  
- **[docker/compose](docker/compose.md)** - 管理 Compose 栈
  - 用途: 部署和管理多容器应用
  - 典型场景: 应用栈部署、开发环境、蓝绿部署

## 如何使用节点

### 基本语法

工作流中使用节点的基本语法：

```yaml
steps:
  - name: Step name
    uses: category/name@version
    with:
      param1: value1
      param2: value2
```

**组成部分:**
- `name`: 步骤名称（可选）
- `uses`: 节点标识符 `category/name@version`
- `with`: 节点参数（键值对）

### 引用输出

使用 `id` 标识步骤，后续步骤可引用其输出：

```yaml
steps:
  - name: Get info
    uses: exec/shell@v1
    with:
      command: whoami
    id: user_info
  
  - name: Use output
    uses: exec/shell@v1
    with:
      command: echo "Current user is ${{ steps.user_info.outputs.stdout }}"
```

**输出引用语法:**
- `${{ steps.<step_id>.outputs.<output_name> }}`
- 可用于任何参数值

### 条件执行

根据前一步骤的结果执行不同操作：

```yaml
steps:
  - name: Check service
    uses: http/request@v1
    with:
      url: http://localhost:8080/health
    id: health_check
    continue-on-error: true
  
  - name: Service is healthy
    if: steps.health_check.outputs.status_code == 200
    uses: exec/shell@v1
    with:
      command: echo "Service is running"
  
  - name: Service is down
    if: steps.health_check.outputs.status_code != 200
    uses: exec/shell@v1
    with:
      command: echo "Service is not available"
```

### 错误处理

#### 继续执行（忽略错误）

```yaml
steps:
  - name: Try to stop container
    uses: docker/exec@v1
    with:
      command: stop
      args: ["my-container"]
    continue-on-error: true  # 容器不存在也继续执行
```

#### 重试策略 (Story 4.3)

```yaml
steps:
  - name: Pull image with retry
    uses: docker/exec@v1
    retry-strategy:
      max-attempts: 3
      initial-interval: 5s
      backoff-coefficient: 2.0
      max-interval: 60s
    with:
      command: pull
      args: ["nginx:latest"]
```

**重试参数:**
- `max-attempts`: 最大尝试次数 (1-10)
- `initial-interval`: 初始重试间隔 (≥1s)
- `backoff-coefficient`: 退避系数 (1.0-10.0)
- `max-interval`: 最大重试间隔

**重试算法:** 指数退避 (`间隔 = initial-interval * backoff-coefficient ^ attempt`)

**永久性错误 (不重试):**  
某些错误类型会立即失败，不进行重试:
- `validation_error` - 参数验证错误
- `schema_error` - Schema 验证错误
- `not_found` - 资源不存在
- `permission_denied` - 权限不足
- `invalid_argument` - 无效参数
- `node_not_registered` - 节点未注册
- `plugin_load_error` - 插件加载失败

更多详情参考: [配置文档 - retry-strategy](../configuration.md#retry-strategy-重试策略)

## 节点错误处理最佳实践

在自定义节点中,应该正确区分永久性错误和临时性错误:

### 永久性错误 (NonRetryableError)

用于参数错误、权限问题等,重试无意义的场景:

```go
import "github.com/Websoft9/waterflow/pkg/dsl/node"

func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
    // 参数验证
    replicas, ok := inputs["replicas"].(float64)
    if !ok || replicas <= 0 {
        return nil, &node.NonRetryableError{
            ErrorType: "validation_error",
            Message:   "parameter 'replicas' must be positive integer",
        }
    }
    
    // 权限检查
    if !hasPermission(ctx) {
        return nil, &node.NonRetryableError{
            ErrorType: "permission_denied",
            Message:   "insufficient permissions to execute this operation",
        }
    }
    
    // ... 执行逻辑
}
```

### 临时性错误 (可重试)

用于网络超时、服务不可用等,重试可能成功的场景:

```go
func (n *MyNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
    // 网络调用
    resp, err := http.Get(url)
    if err != nil {
        // 返回普通 error,Waterflow 会自动重试
        return nil, fmt.Errorf("network call failed: %w", err)
    }
    
    // 服务不可用 (503)
    if resp.StatusCode == 503 {
        return nil, fmt.Errorf("service temporarily unavailable")
    }
    
    // ... 处理响应
}
```

### 错误类型对照表

| 场景 | 错误类型 | 重试 | 示例 |
|------|----------|------|------|
| 参数验证失败 | validation_error | ❌ | 缺少必填参数 |
| JSON Schema 错误 | schema_error | ❌ | 参数类型不匹配 |
| 资源不存在 | not_found | ❌ | 文件/容器不存在 |
| 权限不足 | permission_denied | ❌ | SSH 认证失败 |
| 无效参数 | invalid_argument | ❌ | 端口号超出范围 |
| 网络超时 | (普通 error) | ✅ | http.Get() timeout |
| 服务不可用 | (普通 error) | ✅ | API 返回 503 |
| 临时故障 | (普通 error) | ✅ | 数据库连接失败 |

更多节点开发指南,参考: [节点开发文档](../guides/node-development.md)

### 使用 Secrets

敏感信息（密码、密钥、Token）使用 Secrets 管理：

```yaml
steps:
  - name: Upload with SSH
    uses: file/transfer@v1
    with:
      mode: upload
      host: example.com
      user: deploy
      password: ${{ secrets.SSH_PASSWORD }}  # 使用 Secret
      source_path: /local/app.tar.gz
      target_path: /remote/app.tar.gz
```

**Secrets 引用语法:**
- `${{ secrets.SECRET_NAME }}`
- Secrets 在日志中自动脱敏

### 环境变量

在工作流级别或步骤级别定义环境变量：

```yaml
env:
  DEPLOY_ENV: production
  API_URL: https://api.example.com

steps:
  - name: Deploy with env
    uses: exec/shell@v1
    with:
      command: ./deploy.sh
      env:
        VERSION: "1.2.3"  # 步骤级别环境变量
```

## 节点参数通用说明

### 参数类型

| 类型 | 说明 | 示例 |
|------|------|------|
| string | 字符串 | `"hello"`, `"30s"` |
| int | 整数 | `30`, `8080` |
| bool | 布尔值 | `true`, `false` |
| array | 数组 | `["-la", "/tmp"]` |
| object | 键值对对象 | `{KEY: "value"}` |

### 参数必需性

- ✅ **必需参数**: 必须提供，否则执行失败
- ❌ **可选参数**: 可省略，使用默认值

### 时间格式

多个节点支持时间参数（如 `timeout`, `duration`）：

| 格式 | 说明 | 示例 |
|------|------|------|
| `30s` | 秒 | 30 秒 |
| `5m` | 分钟 | 5 分钟 |
| `2h` | 小时 | 2 小时 |
| `1h30m` | 组合 | 1 小时 30 分钟 |

## 版本兼容性

所有核心节点当前版本为 **v1**，保证向后兼容。

**版本策略:**
- **v1**: 当前稳定版本，新增参数必须为可选参数
- **v2**: 计划中（可能引入不兼容变更）

**升级原则:**
- 同一主版本内保证向后兼容
- 新增参数必须是可选的且有合理默认值
- 不会删除现有参数或改变核心行为

## 常见问题 (FAQ)

### 1. 如何选择合适的节点？

根据任务类型选择：

- **执行系统命令**: 使用 `exec/shell`
- **执行脚本文件**: 使用 `exec/script`
- **等待/延迟**: 使用 `flow/sleep`
- **调用 HTTP API**: 使用 `http/request`
- **传输文件**: 使用 `file/transfer`
- **管理 Docker 容器**: 使用 `docker/exec`
- **部署 Compose 栈**: 使用 `docker/compose`

### 2. 节点在哪里执行？

节点在 **Agent 服务器**上执行。Agent 是部署在目标服务器上的执行器，接收 Waterflow Server 分发的任务。

### 3. 如何调试节点执行失败？

1. **查看日志**: 节点输出的 `stdout` 和 `stderr` 包含详细错误信息
2. **检查退出码**: `exit_code` 显示命令执行结果
3. **验证参数**: 确认所有必需参数已提供且格式正确
4. **检查环境**: 确认 Agent 服务器满足节点前置条件（如 Docker 已安装）

### 4. 节点执行超时怎么办？

1. **增加超时时间**: 根据实际情况调整 `timeout` 参数
2. **优化执行效率**: 减少不必要的操作
3. **使用异步模式**: 某些节点支持后台执行（如 `docker/compose` 的 `detach` 参数）

### 5. 如何处理敏感信息？

- **使用 Secrets**: 密码、密钥等敏感信息存储在 Secrets 中
- **避免硬编码**: 不要在 YAML 中直接写入密码
- **日志脱敏**: Secrets 值在日志中自动隐藏

### 6. 节点支持并发执行吗？

支持。工作流可以同时执行多个步骤，每个节点实例独立执行，互不干扰。

### 7. 如何贡献自定义节点?

参见 [自定义节点开发指南](../guides/node-development.md)。

**示例参考**:
- **入门**: [Echo Node](../../examples/plugins/echo/) - 简单回显示例
- **实战**: [Greeter Node](../../examples/plugins/greeter/) - 多语言问候示例 (推荐, 包含完整测试和文档)

```bash
# 快速体验 Greeter 节点
cd examples/plugins/greeter
make build test
# ✅ 测试通过, 覆盖率 93.8%
```

## 参考文档

- [工作流语法参考](../yaml-dsl-syntax-reference.md)
- [自定义节点开发指南](../guides/node-development.md)
- [Architecture: 节点系统](../adr/0003-plugin-based-node-system.md)
- [Story 3.1: 节点接口设计](../sprint-artifacts/3-1-node-interface-design.md)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: 稳定
