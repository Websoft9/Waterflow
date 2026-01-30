# Test Data Organization

本目录包含 Waterflow 项目的测试数据，按照 BMad 测试标准组织。

## 目录结构

```
testdata/
├── fixtures/                       # 通用测试夹具（跨所有测试类型）
│   ├── valid-workflows/           # 合法的 YAML 工作流样本
│   ├── invalid-workflows/         # 非法的 YAML 工作流（用于验证测试）
│   ├── expressions/               # 表达式引擎测试数据
│   ├── matrix/                    # Matrix 策略测试数据
│   ├── timeout-retry/             # 超时和重试测试数据
│   ├── retry/                     # 重试策略测试数据
│   └── greeter/                   # 示例工作流
├── benchmark/                     # 性能基准测试数据
├── workflows/                     # 通用工作流测试数据
├── deployment-test-app/           # 部署测试应用
├── deployment-test-workflow.yaml  # 部署测试工作流
├── deployment-test-rollback.yaml  # 回滚测试工作流
└── README.md                      # 本文档
```

## 测试数据分层

### 1. 项目级共享数据（本目录）
- **位置**: `testdata/`
- **用途**: 跨所有测试类型共享的数据
- **内容**: 
  - 通用 YAML 工作流样本
  - 验证测试用的非法数据
  - 性能基准测试数据

### 2. 测试类型级数据
- **Integration 共享**: `test/integration/testdata/`
- **Acceptance 共享**: `test/acceptance/testdata/` (可选)

### 3. Epic 级专属数据
- **Integration Epic 1**: `test/integration/epic1/testdata/`
- **Acceptance Epic 1**: `test/acceptance/epic1/testdata/`

## 使用规范

### 在测试中引用数据

```go
// 项目级共享数据
testdata := "testdata/fixtures/valid-workflows/simple.yaml"

// Epic 级专属数据
testdata := "test/integration/epic1/testdata/workflows/conditional.yaml"

// 使用相对路径（推荐）
testdata := filepath.Join("testdata", "fixtures", "valid-workflows", "simple.yaml")
```

### 添加新测试数据

1. **通用测试数据** → 放在 `testdata/fixtures/`
2. **Epic 专属数据** → 放在对应的 `test/{type}/epic{N}/testdata/`
3. **性能测试数据** → 放在 `testdata/benchmark/`

### 文件命名规范

- 使用小写字母和连字符：`simple-workflow.yaml`
- 描述性命名：`matrix-multi-dimension.yaml`
- 测试目的清晰：`invalid-missing-required.yaml`

## fixtures 子目录说明

### valid-workflows/
合法的 YAML 工作流样本，用于正向测试。

**示例**:
- `simple.yaml` - 最简单的工作流
- `multi-job.yaml` - 多 Job 工作流

### invalid-workflows/
非法的 YAML 工作流，用于验证错误处理。

**示例**:
- `missing-required.yaml` - 缺少必需字段
- `invalid-type.yaml` - 字段类型错误
- `syntax-error.yaml` - YAML 语法错误

### expressions/
表达式引擎测试数据。

**示例**:
- `arithmetic.yaml` - 算术运算
- `comparison.yaml` - 比较运算
- `logical.yaml` - 逻辑运算
- `functions.yaml` - 函数调用

### matrix/
Matrix 并行执行策略测试数据。

**示例**:
- `simple.yaml` - 简单矩阵
- `multi-dimension.yaml` - 多维矩阵
- `max-parallel.yaml` - 最大并行度控制
- `fail-fast.yaml` - 快速失败

### timeout-retry/
超时和重试策略测试数据。

**示例**:
- `step-timeout.yaml` - Step 级超时
- `job-timeout.yaml` - Job 级超时
- `custom-retry.yaml` - 自定义重试策略
- `non-retryable.yaml` - 不可重试错误

### greeter/
示例工作流（用于快速入门和演示）。

**示例**:
- `morning-greeting.yaml` - 简单问候示例

## benchmark/

性能基准测试数据，包含不同规模的工作流：

- `small.yaml` - 小型工作流（1-5 steps）
- `medium.yaml` - 中型工作流（10-50 steps）
- `large.yaml` - 大型工作流（100-500 steps）
- `xlarge.yaml` - 超大型工作流（1000+ steps）

## 维护指南

### 添加新的测试数据

1. 确定数据的作用域（项目级 vs Epic 级）
2. 选择正确的目录
3. 使用描述性文件名
4. 添加必要的注释说明数据用途

### 清理过时数据

定期检查并删除：
- 不再使用的测试数据
- 重复的测试文件
- 过时的测试场景

### 重构注意事项

如果移动或重命名测试数据：
1. 使用 `grep` 搜索所有引用
2. 更新所有测试文件中的路径
3. 运行完整的测试套件验证
4. 更新相关文档

## 参考文档

- [集成测试](test/integration/README.md)
- [验收测试](test/acceptance/README.md)
- [BMad 测试标准](https://github.com/bmad-sim/bmad)
