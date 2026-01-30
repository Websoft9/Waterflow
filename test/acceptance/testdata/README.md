# Epic 1 Acceptance Test Data

本目录包含 Epic 1 验收测试的专属测试数据。

## 内容

### workflows/

Epic 1 验收测试场景使用的工作流 YAML 文件。

**文件列表**:
- `health-check.yaml` - 多服务器健康检查工作流（场景 1）
- `distributed-deploy.yaml` - 分布式应用部署工作流（场景 2）

### mock-app/

用于验收测试的模拟应用。

**内容**:
- `Dockerfile` - 模拟应用容器镜像
- `main.go` - 简单的 HTTP 服务器

## 使用规范

### 在测试中引用

```go
// 工作流文件
workflowPath := filepath.Join("testdata", "workflows", "health-check.yaml")

// Mock 应用
mockAppPath := filepath.Join("testdata", "mock-app")
```

### docker-compose 中引用

```yaml
services:
  mock-app:
    build:
      context: ./testdata/mock-app
      dockerfile: Dockerfile
```

## 场景数据映射

| 场景 | 工作流文件 | 说明 |
|------|-----------|------|
| 场景 1 | `health-check.yaml` | 在 3 台服务器上并发执行健康检查 |
| 场景 2 | `distributed-deploy.yaml` | 分布式应用部署（Web + DB） |

## 添加新场景数据

1. 在 `workflows/` 添加新的工作流 YAML
2. 如需额外资源，在对应子目录创建
3. 更新本 README 文档

## 参考

- [Epic 1 验收测试](../README.md)
- [项目 testdata 说明](../../../../testdata/README.md)
