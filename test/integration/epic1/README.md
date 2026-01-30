# Epic 1 集成测试

本目录包含 **Epic 1: Waterflow 核心工作流引擎** 的集成测试。

## 测试覆盖

### 基础设施 (Story 1.1-1.2)
- [story1_1_server_test.go](story1_1_server_test.go) - Waterflow Server 框架
- [story1_2_api_test.go](story1_2_api_test.go) - REST API 服务框架和监控

### DSL 和执行引擎 (Story 1.3-1.8)
- [story1_3_dsl_test.go](story1_3_dsl_test.go) - YAML DSL 解析和验证
- [story1_4_vars_test.go](story1_4_vars_test.go) - 表达式引擎和变量系统
- [story1_5_conditions_test.go](story1_5_conditions_test.go) - 条件执行和控制流
- [story1_6_matrix_test.go](story1_6_matrix_test.go) - Matrix 并行执行策略
- [story1_7_timeout_retry_test.go](story1_7_timeout_retry_test.go) - 超时和重试策略
- [story1_8_temporal_test.go](story1_8_temporal_test.go) - Temporal SDK 集成和工作流执行引擎

### 工作流管理 (Story 1.9-1.11)
- [story1_9_workflow_api_test.go](story1_9_workflow_api_test.go) - 工作流管理 API
- [story1_10_schedule_test.go](story1_10_schedule_test.go) - Schedule API
- [story1_11_webhook_test.go](story1_11_webhook_test.go) - Webhook Trigger

## 测试统计

```
Story 1.1:  7 个集成测试
Story 1.2: 11 个集成测试
Story 1.3:  8 个集成测试
Story 1.4:  7 个集成测试 (变量3 + 表达式4)
Story 1.5:  4 个集成测试
Story 1.6:  4 个集成测试
Story 1.7:  5 个集成测试
Story 1.8:  4 个集成测试
Story 1.9:  7 个集成测试
Story 1.10: 5 个集成测试
Story 1.11: 5 个集成测试

总计: 67 个集成测试
```

## 运行测试

### 运行 Epic 1 所有测试
```bash
cd /data/Waterflow
go test -tags=integration ./test/integration/epic1/...
```

### 运行特定 Story 的测试
```bash
# Story 1.4: 表达式引擎和变量系统
go test -tags=integration ./test/integration/epic1/ -run "TestStory1_4"

# Story 1.9: 工作流管理 API
go test -tags=integration ./test/integration/epic1/ -run "TestStory1_9"
```

### 运行特定测试用例
```bash
# 测试变量解析链
go test -tags=integration ./test/integration/epic1/ -run "TestStory1_4_INT_001"

# 测试工作流提交 API
go test -tags=integration ./test/integration/epic1/ -run "TestStory1_9_INT_001"
```

## 文件命名规范

- **格式**: `story{StoryNumber}_{feature}_test.go`
- **示例**: 
  - `story1_1_server_test.go` - Story 1.1 Server 框架测试
  - `story1_4_vars_test.go` - Story 1.4 变量系统测试

## 测试 ID 规范

- **格式**: `{Epic}.{Story}-INT-{Sequence}`
- **示例**:
  - `1.1-INT-001` - Epic 1, Story 1.1, 集成测试, 序号 001
  - `1.4-INT-003` - Epic 1, Story 1.4, 集成测试, 序号 003

## Story 映射修复

2024-01-30 修复了 Story 1.4-1.9 的测试文件映射问题，详见 [STORY-MAPPING-FIX.md](STORY-MAPPING-FIX.md)。

## 参考文档

- [Epic 1 定义](../../../docs/epics.md#epic-1-waterflow-核心工作流引擎)
- [BMad 测试标准](../README.md#bmad-测试分层)
- [E2E 测试](../../e2e/)
