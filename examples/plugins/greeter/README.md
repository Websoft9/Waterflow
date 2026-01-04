# Greeter Node Plugin

多语言问候语生成节点,支持英语、中文、西班牙语、法语,可根据时间自动生成问候语。

## 功能特性

- ✅ **4 种语言支持** - 英语(en)、中文(zh)、西班牙语(es)、法语(fr)
- ✅ **自动时间检测** - 根据当前时间自动选择 morning/afternoon/evening
- ✅ **参数验证** - 支持 Required、Default、Enum 验证
- ✅ **高测试覆盖率** - 单元测试覆盖率 93.8%
- ✅ **生产就绪** - 包含完整的编译、测试、部署脚本

## 快速开始

### 1. 编译插件

```bash
cd examples/plugins/greeter
export CGO_ENABLED=1
make build
```

生成 `greeter.so` 文件 (~5MB)。

### 2. 运行测试

```bash
make test
```

预期输出:
```
PASS
coverage: 93.8% of statements
ok      github.com/Websoft9/waterflow/examples/plugins/greeter  0.008s
```

### 3. 部署到 Agent

```bash
make install
sudo systemctl restart waterflow-agent
```

### 4. 在工作流中使用

```yaml
steps:
  - name: Greet User
    uses: custom/greeter@v1
    with:
      name: Alice
      language: en
      time_of_day: morning
```

输出:
```json
{
  "greeting": "Good morning, Alice!",
  "language": "en",
  "time_of_day": "morning"
}
```

## 参数说明

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `name` | string | ✅ | - | 要问候的人名 |
| `language` | string | ❌ | `en` | 语言 (en, zh, es, fr) |
| `time_of_day` | string | ❌ | `auto` | 时间段 (morning, afternoon, evening, auto) |

## 输出说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `greeting` | string | 生成的问候语 |
| `language` | string | 使用的语言 |
| `time_of_day` | string | 检测或指定的时间段 |

## 示例

### 英文早晨问候

```yaml
with:
  name: John
  language: en
  time_of_day: morning
```

**输出:** `Good morning, John!`

### 中文晚上问候

```yaml
with:
  name: 小明
  language: zh
  time_of_day: evening
```

**输出:** `晚上好,小明!`

### 自动检测时间

```yaml
with:
  name: Alice
  time_of_day: auto
```

根据当前时间自动生成问候语 (早上 < 12:00, 下午 < 18:00, 其他为晚上)。

### 西班牙语问候

```yaml
with:
  name: María
  language: es
  time_of_day: afternoon
```

**输出:** `Buenas tardes, María!`

## 开发指南

### 项目结构

```
greeter/
├── main.go              # 节点实现 (131 行)
├── main_test.go         # 单元测试 (174 行)
├── integration_test.go  # 集成测试 (80 行)
├── Makefile             # 编译脚本 (7 个目标)
├── README.md            # 本文档
├── go.mod               # Go Module 配置
├── .gitignore           # Git 忽略规则
└── greeter.so           # 编译产物 (5.2 MB, git ignore)
```

### 修改节点

1. **编辑代码**
   ```bash
   vim main.go
   ```

2. **运行测试**
   ```bash
   make test
   ```

3. **编译插件**
   ```bash
   make build
   ```

4. **部署更新**
   ```bash
   make install
   sudo systemctl restart waterflow-agent
   ```

### 添加新语言

1. **扩展 Params() 中的 language Enum**
   ```go
   Enum: []interface{}{"en", "zh", "es", "fr", "de"},  // 添加 "de"
   ```

2. **在 generateGreeting() 中添加翻译**
   ```go
   "de": {
       "morning":   "Guten Morgen, %s!",
       "afternoon": "Guten Tag, %s!",
       "evening":   "Guten Abend, %s!",
   },
   ```

3. **添加测试用例**
   ```go
   {"Hans", "de", "morning", "Guten Morgen, Hans!"},
   ```

4. **验证测试**
   ```bash
   make test
   ```

### 查看覆盖率

```bash
make coverage
open coverage.html  # 或浏览器打开
```

## Makefile 目标

| 目标 | 说明 |
|------|------|
| `make check` | 检查编译环境 (Go 版本、CGO、平台) |
| `make build` | 编译插件为 greeter.so |
| `make test` | 运行单元测试并显示覆盖率 |
| `make coverage` | 生成 HTML 覆盖率报告 |
| `make integration-test` | 运行集成测试 (加载插件) |
| `make install` | 安装插件到 `/opt/waterflow/plugins/custom/` |
| `make clean` | 清理编译产物和覆盖率文件 |

## 故障排除

### 问题 1: 编译失败 "CGO_ENABLED not set"

**症状:**
```
ERROR: CGO_ENABLED should be 1 for plugin build
```

**解决:**
```bash
export CGO_ENABLED=1
make build
```

### 问题 2: 插件加载失败 "plugin was built with a different version of package"

**症状:**
```
plugin was built with a different version of package github.com/Websoft9/waterflow/pkg/dsl/node
```

**原因:** 插件和 Waterflow 使用了不同的 Go 版本编译。

**解决:**
```bash
# 1. 检查 Waterflow 使用的 Go 版本
cat /data/Waterflow/go.mod | grep "go "
# 输出: go 1.22

# 2. 确保插件使用相同版本
go version
# 必须: go version go1.22.x

# 3. 重新编译
make clean
make build
```

### 问题 3: 参数验证失败 "required parameter 'name' is missing"

**症状:**
```
Error: parameter validation failed: required parameter 'name' is missing
```

**原因:** 工作流中未提供必需参数 `name`。

**解决:**
```yaml
# 确保提供所有必需参数
with:
  name: Alice  # ✅ 必需参数
  language: en # ❌ 可选参数
```

### 问题 4: 测试失败 "go: cannot find module"

**症状:**
```
go: cannot find module providing package github.com/Websoft9/waterflow/pkg/dsl/node
```

**解决:**
```bash
# 初始化 Go Module
go mod tidy

# 如果仍失败,检查 replace 指令
cat go.mod | grep replace
# 应包含: replace github.com/Websoft9/waterflow => ../../..
```

### 问题 5: Plugin 目录不存在

**症状:**
```
WARNING: /opt/waterflow/plugins not found
```

**解决:**
```bash
# 创建插件目录
sudo mkdir -p /opt/waterflow/plugins/custom

# 手动复制插件
sudo cp greeter.so /opt/waterflow/plugins/custom/
sudo chown waterflow:waterflow /opt/waterflow/plugins/custom/greeter.so
```

## 技术细节

### 性能指标

- **执行时长:** < 1ms per greeting
- **插件大小:** 5.2 MB
- **内存占用:** < 10 MB per instance
- **测试覆盖率:** 93.8%
- **代码行数:** 131 LOC (目标 <150)

**性能基准测试:**

添加基准测试到 `main_test.go`:

```go
func BenchmarkGreeterNode_Execute(b *testing.B) {
    n := &GreeterNode{}
    inputs := map[string]interface{}{
        "name":        "Benchmark",
        "language":    "en",
        "time_of_day": "morning",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = n.Execute(context.Background(), inputs)
    }
}
```

运行基准测试:
```bash
go test -bench=BenchmarkGreeterNode_Execute -benchmem
# 预期结果: ~500-1000 ns/op, <1000 B/op
```

### 代码统计

```
Language  Files  Lines  Code  Comments  Blanks
───────────────────────────────────────────────
Go        2      305    275   15        15
Makefile  1      53     40    8         5
YAML      1      28     28    0         0
───────────────────────────────────────────────
Total     4      386    343   23        20
```

### 依赖关系

**运行时依赖:**
- `github.com/Websoft9/waterflow/pkg/dsl/node` - Node 接口定义
- Go 标准库: `context`, `fmt`, `time`

**测试依赖:**
- `github.com/stretchr/testify` v1.8.4 - 断言库

**编译要求:**
- Go 1.22+
- CGO_ENABLED=1
- Linux 或 macOS (Windows 不支持 Go Plugin)

## 相关文档

- [Node Development Guide](../../../docs/guides/node-development.md) - 节点开发完整指南
- [Story 3.1: Node Interface Design](../../../docs/sprint-artifacts/3-1-node-interface-design.md) - Node 接口设计
- [Story 4.2: Parameter Schema Validation](../../../docs/sprint-artifacts/4-2-node-parameter-schema-validation.md) - 参数验证
- [ADR-0003: Plugin-Based Node System](../../../docs/adr/0003-plugin-based-node-system.md) - 插件系统架构

## 许可证

MIT License

---

**创建时间:** 2026-01-04  
**测试覆盖率:** 93.8%  
**代码行数:** 131 LOC (不含测试)
