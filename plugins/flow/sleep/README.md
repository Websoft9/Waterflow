# Flow/Sleep Node

延迟等待节点 - 在工作流步骤间添加可控的时间延迟。

## 功能描述

`flow/sleep` 节点提供精确的时间延迟功能，支持秒、分钟、小时等多种时间单位。延迟期间工作流状态会持久化，并支持通过 context 取消延迟操作。

## 使用场景

1. **服务启动等待** - 启动 Docker 容器后等待服务就绪
2. **API 限流控制** - API 调用之间添加延迟避免触发限流
3. **重试延迟** - 失败后等待一段时间再重试
4. **健康检查间隔** - 定期检查服务状态
5. **部署验证** - 部署完成后等待系统稳定

## 参数

### 输入参数

| 参数名 | 类型 | 必需 | 描述 | 示例 |
|--------|------|------|------|------|
| `duration` | string | 是 | 延迟时长 | `30s`, `5m`, `1h`, `1h30m`, `60` |

**支持的时间格式：**
- `30s` - 30 秒
- `5m` - 5 分钟
- `1h` - 1 小时
- `1h30m` - 1 小时 30 分钟
- `30m45s` - 30 分钟 45 秒
- `1h30m45s` - 1 小时 30 分钟 45 秒
- `60` - 60 秒（纯数字默认为秒）

**限制条件：**
- 最小延迟：1 秒
- 最大延迟：24 小时

### 输出参数

| 参数名 | 类型 | 描述 |
|--------|------|------|
| `duration_seconds` | int | 实际延迟秒数 |
| `completed` | bool | 是否正常完成（true）或被取消（false）|
| `elapsed_ms` | int | 实际经过的时间（毫秒）|

## 使用示例

### 示例 1: 基础延迟

```yaml
name: Basic Sleep Example
jobs:
  wait-30-seconds:
    name: Wait 30 Seconds
    runs-on: linux-amd64
    steps:
      - name: Wait for 30 seconds
        uses: flow/sleep@v1
        with:
          duration: 30s
```

### 示例 2: 服务启动等待

```yaml
name: Service Startup Wait
jobs:
  deploy-and-wait:
    name: Deploy and Wait for Service
    runs-on: linux-amd64
    steps:
      - name: Start Docker container
        uses: docker/exec@v1
        with:
          command: run
          args: ["-d", "--name", "webapp", "nginx:latest"]
      
      - name: Wait for container to be ready
        uses: flow/sleep@v1
        with:
          duration: 30s
      
      - name: Health check
        uses: http/request@v1
        with:
          url: http://localhost:80/health
```

### 示例 3: API 限流控制

```yaml
name: API Rate Limiting
jobs:
  fetch-data:
    name: Fetch Data with Rate Limiting
    runs-on: linux-amd64
    steps:
      - name: First API call
        uses: http/request@v1
        with:
          url: https://api.example.com/users/1
      
      - name: Rate limit delay
        uses: flow/sleep@v1
        with:
          duration: 2s
      
      - name: Second API call
        uses: http/request@v1
        with:
          url: https://api.example.com/users/2
      
      - name: Another delay
        uses: flow/sleep@v1
        with:
          duration: 2s
      
      - name: Third API call
        uses: http/request@v1
        with:
          url: https://api.example.com/users/3
```

### 示例 4: 重试延迟

```yaml
name: Retry with Delay
jobs:
  connect-with-retry:
    name: Connect with Retry Logic
    runs-on: linux-amd64
    steps:
      - name: Try to connect
        uses: exec/shell@v1
        with:
          command: nc -zv example.com 80
        continue-on-error: true
      
      - name: Wait before retry
        uses: flow/sleep@v1
        with:
          duration: 10s
      
      - name: Retry connection
        uses: exec/shell@v1
        with:
          command: nc -zv example.com 80
```

### 示例 5: 长时间延迟

```yaml
name: Long Duration Sleep
jobs:
  wait-hours:
    name: Wait Multiple Hours
    runs-on: linux-amd64
    steps:
      - name: Deploy application
        uses: docker/compose@v1
        with:
          action: up
      
      - name: Wait for system stabilization
        uses: flow/sleep@v1
        with:
          duration: 1h30m  # 1.5 hours
      
      - name: Run validation tests
        uses: exec/shell@v1
        with:
          command: ./run-tests.sh
```

### 示例 6: 纯数字格式（默认秒）

```yaml
name: Numeric Duration
jobs:
  simple-wait:
    name: Simple Wait
    runs-on: linux-amd64
    steps:
      - name: Wait 60 seconds (using number)
        uses: flow/sleep@v1
        with:
          duration: 60  # Defaults to seconds
```

## 特性

### 取消支持

延迟操作支持通过 context 取消。当工作流被取消时，sleep 节点会立即返回，`completed` 输出为 `false`。

### 精度

- 短延迟（< 1 秒）：精度 ±1-5ms
- 长延迟（> 1 分钟）：精度误差可忽略不计

### 并发安全

节点实现是并发安全的，可以在多个 goroutine 中同时调用。

## 编译和测试

### 编译插件

```bash
make build
```

### 运行测试

```bash
make test
```

### 查看覆盖率

```bash
make coverage
```

### 集成测试

```bash
make integration-test
```

### 清理

```bash
make clean
```

### 安装

```bash
make install
```

## 技术细节

### 实现机制

- 使用 Go 标准库 `time.NewTimer` 实现精确延迟
- 使用 `select` 语句同时监听 timer 和 context 取消
- Timer 在函数返回前正确停止，避免资源泄漏

### 依赖

- Go 1.22+
- `github.com/websoft9/waterflow/pkg/dsl/node` - Node 接口

### 性能

- 内存占用：极低（~几百 bytes）
- CPU 占用：0%（sleep 期间）
- 取消响应时间：< 1ms

## 错误处理

| 错误情况 | 错误消息 |
|----------|----------|
| duration 参数缺失 | `duration is required` |
| 无效的格式 | `invalid duration: invalid duration format: ...` |
| 延迟小于 1 秒 | `duration must be at least 1 second` |
| 延迟超过 24 小时 | `duration must not exceed 24 hours` |

## 限制和注意事项

### 平台兼容性

- ✅ **Linux**: 完全支持
- ✅ **macOS**: 完全支持
- ❌ **Windows**: 不支持（Go Plugin 系统限制）

在 Windows 平台上，请使用内置编译模式或考虑其他延迟实现方式。

### 编译要求

1. **Go 版本**: 必须使用 Go 1.22 或更高版本
2. **CGO**: 必须启用 CGO (`CGO_ENABLED=1`)
3. **编译器匹配**: 编译插件和 Waterflow Agent 必须使用相同的 Go 版本和编译器

```bash
# 检查 Go 版本
go version

# 确保 CGO 启用
export CGO_ENABLED=1

# 编译插件
make build
```

### 延迟精度

- **短延迟** (< 1 秒): 精度 ±1-5 毫秒
- **中延迟** (1 秒 - 1 分钟): 精度 ±10-20 毫秒
- **长延迟** (> 1 分钟): 精度误差可忽略不计

精度受操作系统调度器影响，在高负载系统上可能略有偏差。

### 最佳实践

1. **避免过短延迟**: 小于 100ms 的延迟可能不稳定，建议使用至少 1 秒
2. **使用合理的最大延迟**: 虽然支持最长 24 小时，但建议不超过 2 小时
3. **考虑取消场景**: 长时间延迟应该配合 context 取消机制使用
4. **生产环境验证**: 在生产环境部署前，务必验证延迟精度是否满足需求

### 资源消耗

- **内存**: 每个 sleep 实例约占用 200-500 bytes
- **CPU**: Sleep 期间 CPU 占用为 0%
- **Goroutine**: 每个 sleep 操作占用 1 个 goroutine

在高并发场景下（如数千个并发 sleep），请注意 goroutine 数量限制。

## 许可证

与 Waterflow 项目保持一致
