# Waterflow Integration Tests

## 概述

本目录包含 Waterflow 的集成测试。集成测试聚焦于**组件间接口和协作**，不测试完整的用户旅程（E2E测试在 `test/e2e/` 目录）。

## 目录结构

```
test/integration/
├── epic1/              # Epic 1: Waterflow 核心工作流引擎
│   ├── story1_1_server_test.go
│   ├── story1_2_api_test.go
│   ├── ...
│   └── README.md       # Epic 1 详细说明
├── epic2/              # Epic 2: (未来扩展)
├── common_test.go      # 跨 Epic 共享的测试工具
├── deprecated/         # 已废弃的测试文件
└── README.md           # 本文件
```

## BMad 测试分层

### 集成测试 (Integration Tests)
- **位置**: `test/integration/`
- **标签**: `//go:build integration`
- **范围**: 组件间接口和数据传递，不包括完整用户流程
- **依赖**: 可使用部分真实服务（如数据库、Temporal Client），但不需要完整环境
- **粒度**: 细粒度，聚焦特定集成点
- **示例**:
  - HTTP 中间件链执行
  - YAML Parser → Go Struct 转换
  - 配置加载和验证逻辑
  - Temporal Client 初始化

### 与 E2E 测试的区别

| 维度 | 集成测试 | E2E 测试 |
|------|----------|----------|
| **测试对象** | 组件接口 | 用户场景 |
| **测试范围** | 部分系统 | 完整系统 |
| **环境要求** | 部分依赖 | Docker Compose 完整栈 |
| **执行速度** | 快 | 慢 |
| **构建标签** | `integration` | `e2e` |

---

## Epic 概览

### Epic 1: Waterflow 核心工作流引擎

**位置**: [epic1/](epic1/)  
**测试数量**: 67 个集成测试  
**覆盖范围**: Story 1.1 - 1.11

详细测试说明见 [epic1/README.md](epic1/README.md)。

---

## 运行测试

### 运行所有集成测试
```bash
go test -tags=integration ./test/integration/...
```

### 运行特定 Epic 的测试
```bash
# Epic 1 所有测试
go test -tags=integration ./test/integration/epic1/...
```

### 运行特定 Story 的测试
```bash
# Epic 1, Story 1.4: 表达式引擎和变量系统
go test -tags=integration ./test/integration/epic1/ -run "TestStory1_4"
```

### 运行特定测试用例
```bash
# Epic 1, Story 1.4, 测试用例 001
go test -tags=integration ./test/integration/epic1/ -run "TestStory1_4_INT_001"
```

### 调试模式
```bash
# 显示详细输出
go test -tags=integration ./test/integration/epic1/... -v

# 显示测试覆盖率
go test -tags=integration ./test/integration/epic1/... -cover
```

### 详细输出
```bash
go test -tags=integration -v ./test/integration/...
```

---

## 测试命名规范

### 文件命名
```
epic{N}_story{M}_{feature}_test.go

示例:
- epic1_story1_1_server_test.go
- epic1_story1_3_dsl_test.go
```

### 测试函数命名
```go
func TestStory{Epic}_{Story}_INT_{Sequence}_{Description}(t *testing.T)

示例:
- TestStory1_1_INT_001_ConfigLoading
- TestStory1_3_INT_002_SchemaValidator
```

### 测试 ID 格式
```
{Epic}.{Story}-INT-{Sequence}

示例:
- 1.1-INT-001
- 1.3-INT-002
```

---

## 测试依赖

### 最小依赖（不需要完整环境）
集成测试设计为**不依赖 Docker Compose 完整栈**，仅需要：
- Go 1.21+
- 被测试的代码包（internal/, pkg/）

### 可选依赖（部分测试需要）
- Temporal Client SDK（Story 1.9 测试）
- Mock 对象（testutil, common_test.go）

---

## 测试状态

### 当前状态
所有测试当前使用 `t.Skip()` 跳过，因为功能尚未实现。

### 实施策略
测试采用 **Test-Driven Development (TDD)** 方式：
1. ✅ **测试框架先行** - 所有测试用例已创建
2. ⏳ **实现驱动** - 当功能实现时，取消 `t.Skip()` 并运行测试
3. ✅ **持续验证** - 每次代码提交后运行集成测试

---

## 测试命名规范

### 文件命名
- **Epic 级别**: `epic{N}/`
- **Story 级别**: `story{StoryNumber}_{feature}_test.go`
- **示例**: `epic1/story1_4_vars_test.go`

### 测试函数命名
- **格式**: `TestStory{Epic}_{Story}_INT_{Sequence}_{Description}`
- **示例**: `TestStory1_4_INT_001_VariableResolutionChain`

### 测试 ID
- **格式**: `{Epic}.{Story}-INT-{Sequence}`
- **示例**: `1.4-INT-001`

---

## 贡献指南

### 实现新功能时
1. 进入对应的 Epic 目录（如 `epic1/`）
2. 找到对应的 Story 测试文件
3. 定位相关的测试用例
4. 删除或注释 `t.Skip()`
5. 实现测试逻辑
6. 运行测试验证实现

### 添加新测试时
1. 遵循命名规范
2. 使用 BDDSpec 框架记录测试意图
3. 在 Epic 的 README 中更新测试覆盖统计
4. 确保测试聚焦组件集成，避免E2E流程

### 添加新 Epic 时
1. 创建 `epic{N}/` 目录
2. 创建 `epic{N}/README.md` 说明文档
3. 按 Story 组织测试文件
4. 在本 README 的 "Epic 概览" 中添加条目

---

## 参考文档

- [Epic 1 集成测试](epic1/README.md)
- [Epic 定义文档](../../docs/epics.md)
- [E2E 测试](../e2e/README.md)
- [测试支持工具](../support/testutil/README.md)
- [BMad 测试标准](https://github.com/bmad-sim/bmad)
