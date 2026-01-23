# 测试 ID 约定

本文档定义了 Waterflow 项目的测试 ID 命名约定。

## 格式

```
{Epic}.{Story}-{TestType}-{Sequence}
```

### 组成部分

| 部分 | 描述 | 示例 |
|------|------|------|
| Epic | 史诗/功能模块标识 | `WF`, `AGENT`, `API`, `NFR` |
| Story | 用户故事编号（可选） | `1`, `2`, `3` |
| TestType | 测试类型 | `UNIT`, `INT`, `E2E`, `PERF` |
| Sequence | 序列号（3位数字） | `001`, `002`, `003` |

### 示例

| Test ID | 含义 |
|---------|------|
| `WF.1-UNIT-001` | 工作流模块，故事1，单元测试#1 |
| `AGENT.2-INT-003` | Agent模块，故事2，集成测试#3 |
| `NFR-PERF-001` | 非功能性需求，性能测试#1 |
| `API.3-E2E-002` | API模块，故事3，端到端测试#2 |

## 测试类型代码

| 代码 | 全称 | 描述 |
|------|------|------|
| `UNIT` | Unit Test | 单元测试 |
| `INT` | Integration Test | 集成测试 |
| `E2E` | End-to-End Test | 端到端测试 |
| `PERF` | Performance Test | 性能测试 |
| `SEC` | Security Test | 安全测试 |
| `STRESS` | Stress Test | 压力测试 |

## 史诗前缀

| 前缀 | 描述 |
|------|------|
| `WF` | 工作流 (Workflow) |
| `AGENT` | Agent 相关 |
| `PLUGIN` | 插件系统 |
| `API` | REST API |
| `NFR` | 非功能性需求 |
| `DSL` | DSL 解析器 |
| `EVENT` | 事件系统 |

## 使用方式

### 方式 1: 在测试注释中标注

```go
// Test ID: WF.1-UNIT-001
// Given: a valid workflow YAML
// When: parsing the workflow
// Then: workflow object is created correctly
func TestParser_Parse_ValidYAML(t *testing.T) {
    // ...
}
```

### 方式 2: 使用 BDDSpec 结构体

```go
func TestWorkflowExecution(t *testing.T) {
    spec := testutil.BDDSpec{
        TestID: testutil.GenerateTestID("WF", "1", testutil.TestTypeUnit, 1),
        Given:  "a valid workflow definition",
        When:   "the workflow is submitted",
        Then:   []string{"workflow starts successfully"},
    }
    spec.Log(t)
    
    // test implementation...
}
```

### 方式 3: 使用 GenerateTestID 函数

```go
testID := testutil.GenerateTestID("WF", "1", "UNIT", 1)
// 输出: "WF.1-UNIT-001"

testID := testutil.GenerateTestID("NFR", "", "PERF", 5)
// 输出: "NFR-PERF-005"
```

## 与 BDD 结合使用

推荐在测试函数注释中同时使用 Test ID 和 BDD 格式：

```go
// Test ID: WF.2-INT-003
//
// Given: a running Waterflow server with Temporal
// When: submitting a multi-step workflow
// Then:
//   - all steps execute in sequence
//   - final status is "completed"
//
// Acceptance Criteria:
//   - AC1: Step execution order is correct
//   - AC2: Step outputs are passed correctly
//   - AC3: Workflow completes within timeout
func TestMultiStepWorkflow_Integration(t *testing.T) {
    testutil.RequireTemporal(t)
    // ...
}
```

## 最佳实践

1. **保持一致性**: 同一模块内的测试使用相同的 Epic 前缀
2. **顺序递增**: 序列号在同一 Epic.Story-TestType 组合内递增
3. **文档关联**: Test ID 应可追溯到需求文档
4. **避免重复**: 每个 Test ID 必须唯一
5. **及时更新**: 删除测试时记录已弃用的 Test ID

## 工具支持

`test/support/testutil/bdd.go` 提供了以下工具：

```go
// 生成 Test ID
testutil.GenerateTestID(epic, story, testType, sequence)

// 测试类型常量
testutil.TestTypeUnit    // "UNIT"
testutil.TestTypeInt     // "INT"
testutil.TestTypeE2E     // "E2E"
testutil.TestTypePerf    // "PERF"
testutil.TestTypeSec     // "SEC"
testutil.TestTypeStress  // "STRESS"

// 史诗前缀常量
testutil.EpicWorkflow    // "WF"
testutil.EpicAgent       // "AGENT"
testutil.EpicPlugin      // "PLUGIN"
testutil.EpicAPI         // "API"
testutil.EpicNFR         // "NFR"
```
