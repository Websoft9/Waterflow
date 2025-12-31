# Story 验证报告 - Story 4.2

**文档:** [4-2-node-parameter-schema-validation.md](4-2-node-parameter-schema-validation.md)  
**验证日期:** 2025-12-31  
**验证框架:** `.bmad/core/tasks/validate-workflow.xml`  
**清单:** `.bmad/bmm/workflows/4-implementation/create-story/checklist.md`

---

## 执行摘要

**总体评分:** 92/100 (A-)

**关键发现:**
- ✅ **20 个优势项** - 技术设计完整,代码示例丰富
- ⚠️ **5 个需增强项** - 缺少代码基线、类型转换细节、Mock 策略
- 🔥 **2 个关键缺失** - 当前代码引用、表达式参数处理

**建议操作:**
- **必须修复 (Critical):** 2 项 - 代码基线和表达式处理
- **应该增强 (High):** 3 项 - 类型转换、测试策略、性能基准
- **可选优化 (Medium):** 0 项

---

## 1. 源文档分析完整性

### ✅ 优势

1. **技术设计详细** - ValidateInputs() 完整算法、类型验证实现
2. **代码示例丰富** - 25+ 个完整代码片段
3. **错误类型完整** - InputValidationError、ParameterError 详细定义
4. **ADR 引用准确** - ADR-0002 错误分类、ADR-0003 ParamSpec

### 🚨 关键缺失 (#1)

**缺失:** **当前 validator.go 的代码基线**

**问题:** Story 说"扩展现有文件"但未提供:
- validator.go 当前已实现的函数列表
- ValidateInputs() 是否已有 stub 实现
- 现有代码的具体行号和内容

**影响:** LLM 可能:
- 不知道从哪里开始添加代码
- 重复实现已存在的辅助函数
- 破坏现有的 ValidateNode() 实现

**建议修复:**
```markdown
## 当前代码基线 (必读)

### pkg/dsl/node/validator.go 现状

**文件:** `pkg/dsl/node/validator.go` (L1-264)

**已实现的函数:**
```go
// L7-60: ValidateNode(n Node) error
// 验证节点结构完整性 (Name, Version, Metadata, ParamSpec)

// L62-70: isValidNodeName(name string) bool
// 辅助函数 - 验证节点名称格式

// L72-76: isValidVersion(version string) bool
// 辅助函数 - 验证版本格式

// L78-87: isValidCategory(category string) bool
// 辅助函数 - 验证节点类别

// L89-135: validateParamSpec(paramName string, spec ParamSpec) error
// 辅助函数 - 验证 ParamSpec 结构
```

**L137-150: ValidateInputs() STUB 实现**
```go
func ValidateInputs(inputs map[string]interface{}, specs map[string]ParamSpec) error {
    // TODO: Story 4.2 implementation
    return nil
}
```

**本 Story 需完成:**
1. 替换 L137-150 的 stub 为完整实现
2. 可复用现有辅助函数 (validateParamSpec 验证 Pattern 编译)
3. 添加新的辅助函数: validateType, validatePattern, validateEnum, validateRange, applyDefaults

**代码插入位置:**
- ValidateInputs() 主函数: 替换 L137-150
- 辅助函数: 在 L151+ 添加 (文件末尾)
```

---

## 2. 技术规格灾难预防

### ✅ 优势

1. **类型系统完整** - 支持 6 种类型 (string, int, float, bool, object, array)
2. **约束验证清晰** - Pattern, Enum, MinValue, MaxValue
3. **错误不可重试** - NonRetryable() 正确标记

### 🚨 关键缺失 (#2)

**缺失:** **JSON 数值类型转换的具体处理**

**问题:** YAML/JSON 解析后:
- 所有数字默认解析为 `float64` (Go encoding/json 行为)
- 用户定义 `timeout: 30` 会被解析为 `float64(30.0)` 而非 `int(30)`
- 简单的类型断言会导致验证失败

**灾难场景:**
```yaml
steps:
  - uses: exec/shell@v1
    with:
      timeout: 30  # YAML 解析后是 float64,但 ParamSpec 期望 int
```

**验证逻辑:**
```go
// ❌ 简单实现会失败
case "int":
    if _, ok := value.(int); !ok {
        return typeError  // value 是 float64, 验证失败!
    }
```

**建议修复:**
```markdown
### 类型转换兼容性处理

#### YAML/JSON 数值解析行为

**Go encoding/json 和 yaml.v3 行为:**
- 所有整数解析为 `float64` (除非显式 Unmarshal 到 int)
- `timeout: 30` → `float64(30.0)`
- `port: 8080` → `float64(8080.0)`

**兼容性策略:**

```go
func validateType(paramName string, value interface{}, expectedType string) *ParameterError {
    switch expectedType {
    case "string":
        if _, ok := value.(string); !ok {
            return typeError(paramName, "string", value)
        }
    
    case "int":
        switch v := value.(type) {
        case int, int32, int64:
            return nil  // 原生 int 类型
        case float64:
            // JSON 解析的数字,检查是否为整数
            if v == float64(int64(v)) {
                return nil  // ✅ 30.0 视为有效 int
            }
            return typeError(paramName, "int", value)  // 30.5 拒绝
        default:
            return typeError(paramName, "int", value)
        }
    
    case "float":
        switch v := value.(type) {
        case float32, float64:
            return nil
        case int, int32, int64:
            return nil  // ✅ int 可以隐式转换为 float
        default:
            return typeError(paramName, "float", value)
        }
    
    case "bool":
        if _, ok := value.(bool); !ok {
            return typeError(paramName, "bool", value)
        }
    
    case "object":
        if _, ok := value.(map[string]interface{}); !ok {
            return typeError(paramName, "object", value)
        }
    
    case "array":
        if _, ok := value.([]interface{}); !ok {
            return typeError(paramName, "array", value)
        }
    }
    
    return nil
}
```

**测试用例 (必需):**
```go
func TestValidateInputs_JSONNumberHandling(t *testing.T) {
    specs := map[string]ParamSpec{
        "timeout": {Type: "int", Required: true},
        "ratio":   {Type: "float", Required: true},
    }
    
    // 模拟 JSON 解析结果 (所有数字都是 float64)
    inputs := map[string]interface{}{
        "timeout": float64(30),      // ✅ 应接受 (整数值)
        "ratio":   float64(0.5),     // ✅ 应接受
    }
    
    err := ValidateInputs(inputs, specs)
    assert.NoError(t, err)
    
    // 拒绝非整数的 float64
    inputs["timeout"] = float64(30.5)
    err = ValidateInputs(inputs, specs)
    assert.Error(t, err)
}
```
```

---

## 3. 表达式参数处理

### ⚠️ 需增强 (#3)

**问题:** Story 提到"表达式字段跳过静态验证"但未详细说明:
- 如何检测参数值是表达式?
- 运行时如何确保表达式求值后再验证?
- DSL Validator 和 Runtime Validator 的协作机制

**影响:** 可能导致:
- 表达式参数在提交时被错误验证
- 运行时验证跳过表达式参数

**建议补充:**

```markdown
### 表达式参数的双阶段验证

#### 静态验证 (DSL Validator, Story 1.3)

**检测表达式:**
```go
func isExpression(value interface{}) bool {
    if str, ok := value.(string); ok {
        return strings.Contains(str, "${{") && strings.Contains(str, "}}")
    }
    // 对象和数组可能嵌套表达式,递归检查
    return false
}
```

**DSL Validator 中跳过表达式:**
```go
// pkg/dsl/validator.go (Story 1.3)
func (v *DSLValidator) ValidateStep(step *Step) error {
    // 获取节点 ParamSpec
    nodeInstance, _ := v.nodeRegistry.Get(step.Uses)
    specs := nodeInstance.Params()
    
    // 验证静态参数
    for paramName, spec := range specs {
        value, exists := step.With[paramName]
        if !exists {
            if spec.Required {
                return fmt.Errorf("missing required param: %s", paramName)
            }
            continue
        }
        
        // ⚠️ 跳过表达式参数
        if isExpression(value) {
            continue  // 运行时验证
        }
        
        // 验证静态值
        if err := node.ValidateInputs(
            map[string]interface{}{paramName: value},
            map[string]ParamSpec{paramName: spec},
        ); err != nil {
            return err
        }
    }
    
    return nil
}
```

#### 运行时验证 (Activity, 本 Story)

**表达式求值后验证:**
```go
// internal/agent/activity.go
func (a *ExecuteNodeActivity) Execute(ctx context.Context, input ActivityInput) (*ActivityOutput, error) {
    // 1. 获取节点
    nodeInstance, _ := a.nodeRegistry.Get(input.NodeType)
    
    // 2. 表达式求值 (Story 1.4 实现)
    evaluatedInputs := a.evaluateExpressions(input.Inputs, input.Context)
    // evaluatedInputs 中所有 ${{ }} 已替换为实际值
    
    // 3. 运行时参数验证 (本 Story)
    if err := node.ValidateInputs(evaluatedInputs, nodeInstance.Params()); err != nil {
        return nil, err  // 验证失败,不执行
    }
    
    // 4. 执行节点
    result, err := nodeInstance.Execute(ctx, evaluatedInputs)
    return result, err
}
```

**测试场景:**
```yaml
# 工作流定义
vars:
  timeout_value: 30

steps:
  - uses: exec/shell@v1
    with:
      command: echo hello
      timeout: ${{ vars.timeout_value }}  # 表达式
```

**验证流程:**
1. **DSL Validator:** 检测到 `${{ vars.timeout_value }}` 是表达式,跳过验证
2. **Activity 执行时:** 表达式求值为 `30` (int or float64)
3. **ValidateInputs():** 验证 `timeout: 30` 符合 int 类型要求
```

---

## 4. 测试策略

### ✅ 优势

1. **测试用例详细** - 类型、约束、错误类型全覆盖
2. **集成测试考虑** - 端到端和真实节点测试
3. **覆盖率目标明确** - >90%

### ⚠️ 需增强 (#4)

**缺失:** **Mock 策略和依赖隔离**

**问题:** 单元测试可能:
- 依赖真实的 NodeRegistry (耦合)
- 无法模拟 Activity 执行环境
- 难以测试 Temporal 错误分类

**建议补充:**

```markdown
### 测试隔离和 Mock 策略

#### 单元测试 - 纯函数测试

**ValidateInputs() 是纯函数 - 无需 Mock:**
```go
func TestValidateInputs_PureFunction(t *testing.T) {
    // 直接构造参数,无外部依赖
    specs := map[string]ParamSpec{
        "command": {Type: "string", Required: true},
    }
    inputs := map[string]interface{}{
        "command": "echo hello",
    }
    
    err := ValidateInputs(inputs, specs)
    assert.NoError(t, err)
}
```

**优势:** 快速、稳定、易维护

#### 集成测试 - Activity 集成

**Mock NodeRegistry:**
```go
type MockNodeRegistry struct {
    nodes map[string]node.Node
}

func (m *MockNodeRegistry) Get(nodeType string) (node.Node, error) {
    if n, ok := m.nodes[nodeType]; ok {
        return n, nil
    }
    return nil, fmt.Errorf("node not found")
}

func TestActivityParameterValidation(t *testing.T) {
    // 1. 创建 Mock 节点
    mockNode := &MockNode{
        NameValue: "test/validator",
        ParamsValue: map[string]ParamSpec{
            "command": {Type: "string", Required: true},
        },
    }
    
    // 2. Mock Registry
    registry := &MockNodeRegistry{
        nodes: map[string]node.Node{
            "test/validator@v1": mockNode,
        },
    }
    
    // 3. 创建 Activity
    activity := &ExecuteNodeActivity{
        nodeRegistry: registry,
    }
    
    // 4. 测试有效输入
    validInput := ActivityInput{
        NodeType: "test/validator@v1",
        Inputs:   map[string]interface{}{"command": "ls"},
    }
    _, err := activity.Execute(context.Background(), validInput)
    assert.NoError(t, err)
    
    // 5. 测试无效输入 (缺少必需参数)
    invalidInput := ActivityInput{
        NodeType: "test/validator@v1",
        Inputs:   map[string]interface{}{},  // 缺少 command
    }
    _, err = activity.Execute(context.Background(), invalidInput)
    assert.Error(t, err)
    
    // 6. 验证错误类型
    var validationErr *node.InputValidationError
    assert.True(t, errors.As(err, &validationErr))
    assert.True(t, validationErr.NonRetryable())
}
```
```

---

## 5. 性能优化

### ✅ 优势

1. **缓存策略** - 正则表达式编译缓存
2. **性能目标** - <100µs (10 参数)
3. **基准测试** - 提供完整基准测试

### ⚠️ 需增强 (#5)

**缺失:** **缓存并发安全的详细说明**

**问题:** sync.Map 使用正确但未说明:
- 为什么选择 sync.Map 而非 map+mutex?
- Load/Store 的并发场景分析
- 缓存命中率监控

**建议补充:**

```markdown
### 正则缓存并发安全分析

#### sync.Map vs map+RWMutex

**选择 sync.Map 的原因:**

| 场景 | sync.Map | map+RWMutex | 结论 |
|------|----------|-------------|------|
| **读多写少** | ✅ 优秀 | ✅ 良好 | sync.Map 略胜 |
| **写入频率** | ✅ 首次编译写入 | ✅ 首次编译写入 | 相当 |
| **并发读** | ✅ 无锁 | ✅ RLock 共享 | sync.Map 更优 |
| **内存占用** | ⚠️ 稍高 | ✅ 较低 | 可接受 |

**并发场景:**
```
Pattern 缓存生命周期:
1. 首次验证某 Pattern → Store() 写入缓存
2. 后续验证相同 Pattern → Load() 读取 (无锁,高性能)
3. Pattern 总数有限 (<100) → 内存占用可忽略
```

**实现:**
```go
var patternCache sync.Map  // 全局缓存

func validatePattern(paramName, value, pattern string) *ParameterError {
    // 1. 尝试从缓存加载 (无锁读取)
    if cached, ok := patternCache.Load(pattern); ok {
        re := cached.(*regexp.Regexp)
        if !re.MatchString(value) {
            return patternMismatchError(paramName, pattern, value)
        }
        return nil
    }
    
    // 2. 缓存未命中,编译并存储
    re, err := regexp.Compile(pattern)
    if err != nil {
        return patternCompileError(paramName, pattern, err)
    }
    
    // 3. 存储到缓存 (可能多次 Store 同一 key,但无害)
    patternCache.Store(pattern, re)
    
    // 4. 验证
    if !re.MatchString(value) {
        return patternMismatchError(paramName, pattern, value)
    }
    
    return nil
}
```

**并发写入竞态:**
```go
// 场景: 两个 goroutine 同时验证相同的新 Pattern
// Goroutine A:                Goroutine B:
Load("^[a-z]+$") → miss       Load("^[a-z]+$") → miss
Compile("^[a-z]+$") → re1     Compile("^[a-z]+$") → re2
Store("^[a-z]+$", re1)        Store("^[a-z]+$", re2)
                              ← re2 覆盖 re1 (无害,结果相同)
```

**监控缓存效率:**
```go
var patternCacheMetrics = struct {
    hits   uint64
    misses uint64
}{}

func validatePattern(...) {
    if cached, ok := patternCache.Load(pattern); ok {
        atomic.AddUint64(&patternCacheMetrics.hits, 1)
        // ... 使用缓存
    } else {
        atomic.AddUint64(&patternCacheMetrics.misses, 1)
        // ... 编译并缓存
    }
}

// Prometheus 指标
var patternCacheHitRate = promauto.NewGaugeFunc(
    prometheus.GaugeOpts{
        Name: "waterflow_pattern_cache_hit_rate",
        Help: "Pattern regex cache hit rate",
    },
    func() float64 {
        hits := atomic.LoadUint64(&patternCacheMetrics.hits)
        misses := atomic.LoadUint64(&patternCacheMetrics.misses)
        total := hits + misses
        if total == 0 {
            return 0
        }
        return float64(hits) / float64(total)
    },
)
```
```

---

## 改进建议汇总

### 🔥 Critical (必须修复)

| # | 类别 | 问题 | 影响 | 优先级 |
|---|------|------|------|--------|
| 1 | 代码基线 | 缺少 validator.go 当前实现引用 | LLM 不知道从哪开始 | P0 |
| 2 | 类型转换 | JSON 数值类型转换未详细说明 | 验证失败 | P0 |

### ⚡ High (应该增强)

| # | 类别 | 问题 | 影响 | 优先级 |
|---|------|------|------|--------|
| 3 | 表达式处理 | 双阶段验证机制不够清晰 | 表达式参数验证错误 | P1 |
| 4 | 测试策略 | 缺少 Mock 策略说明 | 测试耦合度高 | P1 |
| 5 | 性能优化 | 缓存并发安全未详细说明 | 理解不足 | P1 |

---

## 总体评价

**优势:**
- ✅ 技术设计完整且详细
- ✅ 代码示例丰富实用
- ✅ 错误处理清晰
- ✅ 测试用例全面

**需改进:**
- ⚠️ 代码基线引用缺失
- ⚠️ 类型转换兼容性需补充
- ⚠️ 表达式参数处理需详细化

**Story 现状:** 92分 (A-) - 技术设计优秀,补充代码基线和类型转换细节后可达 A+

---

## 下一步操作

**建议应用改进:**
1. **critical** - 应用 Critical 级别改进 (2项)
2. **critical+high** - 应用 Critical + High (5项,推荐⭐)
3. **select** - 选择特定编号
4. **review** - 查看详细修改
5. **none** - 保持原样

**我的推荐:** **critical+high** (应用 1-5 项改进)

**您的选择:**
```
