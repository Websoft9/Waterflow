# Epic 1 Integration Test Data

本目录包含 Epic 1 集成测试的专属测试数据。

## 内容

### workflows/

Epic 1 集成测试使用的工作流 YAML 文件。

**文件列表**:
- `simple-echo.yaml` - 简单 echo 工作流（Story 1.1-1.2）
- `conditional.yaml` - 条件执行工作流（Story 1.5）
- `matrix.yaml` - Matrix 策略工作流（Story 1.6）
- `retry.yaml` - 重试策略工作流（Story 1.7）
- `job-dependencies.yaml` - Job 依赖关系（Story 1.3）
- `multi-step-outputs.yaml` - 多 Step 输出（Story 1.4）

## 使用规范

### 在测试中引用

```go
// 使用相对路径
testdataPath := filepath.Join("testdata", "workflows", "simple-echo.yaml")

// 或使用绝对路径（从项目根）
testdataPath := "test/integration/epic1/testdata/workflows/simple-echo.yaml"
```

### 添加新文件

1. 文件名应该反映测试场景
2. 添加注释说明对应的 Story
3. 保持 YAML 格式规范

## 与项目级 testdata 的关系

- **Epic 专属数据** → 放在此目录
- **跨 Epic 共享数据** → 放在 `test/integration/testdata/`
- **项目级共享数据** → 放在 `testdata/fixtures/`

## 参考

- [Epic 1 集成测试](../README.md)
- [项目 testdata 说明](../../../../testdata/README.md)
