# ADR-0005: 表达式系统语法

**状态:** ✅ 已采纳  
**日期:** 2025-12-13  
**决策者:** 架构团队  

## 背景

DSL 需要支持动态值和条件逻辑:

```yaml
steps:
  - name: Use Output
    with:
      commit: ???  # 如何引用前面步骤的输出?
  
  - name: Conditional
    if: ???  # 如何判断条件?
```

需要决定表达式的语法和能力:
1. **简单变量替换** - 只支持 `${var}` 引用
2. **完整表达式** - 支持运算/函数/条件
3. **脚本语言嵌入** - Lua/JavaScript 等

## 决策

采用 **GitHub Actions 风格表达式语法**: `${{ expression }}`

## 理由

### 核心优势:

1. **用户熟悉**
   - GitHub Actions 已广泛使用
   - 与 YAML DSL 保持一致
   - 降低学习成本

2. **明确边界**
   - `${{ }}` 清晰标识表达式
   - 与普通字符串区分明显
   - 避免意外替换

3. **足够的表达能力**
   - 变量引用: `${{ workflow.id }}`
   - 运算: `${{ 1 + 2 }}`
   - 函数: `${{ toJSON(job.outputs) }}`
   - 条件: `${{ job.status == 'success' }}`

4. **安全性**
   - 沙箱执行,无文件系统访问
   - 无代码注入风险
   - 计算超时保护

### 与其他方案对比:

| 方案 | 优点 | 缺点 | 决策 |
|------|------|------|------|
| **${{ }}** | 熟悉,安全,能力适中 | 表达能力有限 | ✅ 选择 |
| `${var}` | 简单 | 只能引用,无运算 | ❌ |
| Lua/JS 嵌入 | 能力强 | 安全风险,学习成本高 | ❌ |
| Jinja2 模板 | 功能强大 | 学习成本,Python 依赖 | ❌ |

## 后果

### 正面影响:

✅ **熟悉语法** - 用户直接复用 GHA 经验  
✅ **安全执行** - 沙箱隔离,无注入风险  
✅ **清晰边界** - `${{ }}` 明确标识表达式  
✅ **足够能力** - 覆盖 90% 常见场景  

### 负面影响:

⚠️ **表达能力有限** - 无法实现复杂逻辑(需用脚本节点)  
⚠️ **实现复杂度** - 需要词法分析/语法分析/求值引擎  

### 风险缓解:

- 对于复杂逻辑,使用 `run` 节点执行脚本
- 提供详细的表达式错误提示

## 语法规范

### 上下文对象:

```yaml
# workflow 上下文
${{ workflow.id }}           # 工作流 ID
${{ workflow.name }}         # 工作流名称
${{ workflow.repository }}   # 仓库名

# job 上下文
${{ job.id }}               # Job ID
${{ job.status }}           # Job 状态: success/failure/cancelled

# steps 上下文
${{ steps.<step-id>.outputs.<key> }}  # 步骤输出

# inputs 上下文
${{ inputs.branch }}        # 输入参数

# env 上下文
${{ env.PATH }}            # 环境变量
```

### 运算符:

```yaml
# 比较
${{ job.status == 'success' }}
${{ steps.test.outputs.code != 0 }}
${{ env.CPU_COUNT > 4 }}

# 逻辑
${{ job.status == 'success' && steps.test.outputs.passed }}
${{ env.SKIP_TESTS || inputs.force-run }}
${{ !env.PRODUCTION }}

# 算术
${{ 1 + 2 }}
${{ env.TIMEOUT * 60 }}
```

### 内置函数:

```yaml
# 字符串
${{ format('Hello {0}', inputs.name) }}
${{ contains('hello world', 'world') }}  # true
${{ startsWith(job.id, 'build-') }}
${{ endsWith(env.FILE, '.json') }}

# JSON
${{ toJSON(steps.build.outputs) }}
${{ fromJSON(env.CONFIG).database.host }}

# 条件
${{ success() }}          # 所有前置步骤成功
${{ failure() }}          # 任一前置步骤失败
${{ always() }}           # 总是执行
${{ cancelled() }}        # 工作流被取消
```

### 使用场景:

#### 1. 引用变量:

```yaml
steps:
  - name: Checkout
    uses: checkout@v1
    with:
      repository: ${{ workflow.repository }}
      branch: ${{ inputs.branch }}
```

#### 2. 条件执行:

```yaml
steps:
  - name: Deploy
    if: ${{ job.status == 'success' && env.ENVIRONMENT == 'production' }}
    uses: deploy@v1
```

#### 3. 动态参数:

```yaml
steps:
  - name: Build
    uses: run@v1
    with:
      command: |
        echo "Commit: ${{ steps.checkout.outputs.commit }}"
        echo "Status: ${{ job.status }}"
```

#### 4. 计算值:

```yaml
steps:
  - name: Wait
    uses: sleep@v1
    with:
      seconds: ${{ inputs.timeout * 60 }}  # 分钟转秒
```

## 实现示例

### 表达式引擎:

```go
// pkg/expr/engine.go
package expr

type Engine struct {
    context map[string]interface{}
}

func NewEngine(ctx map[string]interface{}) *Engine {
    return &Engine{context: ctx}
}

// Evaluate 求值表达式
func (e *Engine) Evaluate(expr string) (interface{}, error) {
    // 1. 词法分析
    tokens := e.tokenize(expr)
    
    // 2. 语法分析(生成 AST)
    ast, err := e.parse(tokens)
    if err != nil {
        return nil, err
    }
    
    // 3. 求值
    return e.eval(ast)
}

// 示例: 简化版求值
func (e *Engine) eval(node ASTNode) (interface{}, error) {
    switch node.Type {
    case NodeVariable:
        // 变量引用: workflow.id
        return e.resolveVariable(node.Value)
    
    case NodeBinaryOp:
        // 二元运算: a == b
        left, _ := e.eval(node.Left)
        right, _ := e.eval(node.Right)
        return e.applyOp(node.Op, left, right)
    
    case NodeFunction:
        // 函数调用: format('Hello {0}', name)
        return e.callFunction(node.Name, node.Args)
    }
}
```

### 在 YAML 中替换表达式:

```go
// pkg/dsl/expr_replacer.go
func ReplaceExpressions(input string, ctx map[string]interface{}) (string, error) {
    // 正则匹配 ${{ ... }}
    re := regexp.MustCompile(`\$\{\{(.+?)\}\}`)
    
    return re.ReplaceAllStringFunc(input, func(match string) string {
        // 提取表达式内容
        expr := strings.TrimSpace(match[3 : len(match)-2])
        
        // 求值
        engine := expr.NewEngine(ctx)
        result, err := engine.Evaluate(expr)
        if err != nil {
            return match // 求值失败,保留原文
        }
        
        // 转为字符串
        return fmt.Sprintf("%v", result)
    })
}
```

### 使用示例:

```go
// Server 中渲染 Job
func (s *Server) RenderJob(job *Job, ctx map[string]interface{}) (*Job, error) {
    rendered := &Job{}
    
    for _, step := range job.Steps {
        // 渲染 step.With 中的表达式
        for key, value := range step.With {
            if strValue, ok := value.(string); ok {
                rendered, err := ReplaceExpressions(strValue, ctx)
                if err != nil {
                    return nil, err
                }
                step.With[key] = rendered
            }
        }
        
        // 求值 if 条件
        if step.If != "" {
            engine := expr.NewEngine(ctx)
            result, _ := engine.Evaluate(step.If)
            if !result.(bool) {
                continue // 跳过此步骤
            }
        }
        
        rendered.Steps = append(rendered.Steps, step)
    }
    
    return rendered, nil
}
```

## 表达式库选择

### 方案 1: 自研 (推荐)

**优点:**
- 完全控制,精确匹配 GHA 语法
- 无外部依赖
- 沙箱安全

**缺点:**
- 开发成本高(词法/语法分析)

### 方案 2: 使用 expr 库

[antonmedv/expr](https://github.com/antonmedv/expr) - Go 表达式引擎

```go
import "github.com/antonmedv/expr"

result, err := expr.Eval("1 + 2", nil)
```

**优点:**
- 现成的引擎,开发成本低
- 性能好,已优化

**缺点:**
- 语法与 GHA 有差异
- 需要适配上下文对象

**决策:** MVP 阶段使用 `expr` 库快速实现,后续可替换为自研引擎。

## 安全考虑

### 沙箱隔离:

```go
// 禁止访问文件系统
func (e *Engine) callFunction(name string, args []interface{}) (interface{}, error) {
    allowedFunctions := []string{
        "format", "contains", "startsWith", "endsWith",
        "toJSON", "fromJSON", "success", "failure",
    }
    
    if !contains(allowedFunctions, name) {
        return nil, fmt.Errorf("function not allowed: %s", name)
    }
    
    // 执行函数
}
```

### 计算超时:

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

result, err := engine.EvaluateWithContext(ctx, expr)
```

### 资源限制:

```go
// 限制表达式长度
if len(expr) > 1024 {
    return nil, fmt.Errorf("expression too long")
}

// 限制嵌套深度
if depth > 10 {
    return nil, fmt.Errorf("expression too complex")
}
```

## 替代方案

### 方案 A: Shell 变量语法 `${var}` (被拒绝)

```yaml
with:
  commit: ${steps.checkout.outputs.commit}
```

**被拒绝原因:**
- ❌ 只能引用变量,无法运算
- ❌ 无法实现条件逻辑
- ❌ 容易与 Shell 变量混淆

### 方案 B: Jinja2 模板 (被拒绝)

```yaml
with:
  message: "{{ workflow.name }} finished with {{ job.status }}"
```

**被拒绝原因:**
- ❌ 学习成本高(Python 模板语法)
- ❌ 用户不熟悉
- ❌ 需要 Python 依赖(cgo)

### 方案 C: 嵌入 JavaScript/Lua (被拒绝)

```yaml
if: |
  function() {
    return job.status == 'success' && env.PROD
  }
```

**被拒绝原因:**
- ❌ 安全风险大(需要严格沙箱)
- ❌ 学习成本高
- ❌ 性能开销大

## 参考资料

- [GitHub Actions 表达式语法](https://docs.github.com/en/actions/learn-github-actions/expressions)
- [antonmedv/expr](https://github.com/antonmedv/expr)
- [PRD: 表达式系统](../prd.md#表达式系统)
