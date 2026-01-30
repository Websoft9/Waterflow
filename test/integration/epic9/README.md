# Epic 9: 安全和合规 - 集成测试

Epic 9 实现了 Waterflow 的安全基础设施和合规特性，包括密钥管理、TLS 加密、审计日志等。

## Epic 概述

Epic 9 包含以下 Story：

| Story | 描述 | 状态 |
|-------|------|------|
| Story 9.1 | HTTPS/TLS 支持 | ✅ Validated |
| Story 9.2 | SecretProvider 接口实现 | ✅ Validated |
| Story 9.3 | 审计日志实现 | ✅ Validated |
| Story 9.4 | 安全最佳实践文档 | ✅ Validated |

## 测试范围

### ⚠️ 重要说明：PRD 追溯

**Epic 9 是技术实现层面的 Story，不直接追溯到 PRD 功能需求（FR）。**

- ❌ PRD 中**明确排除**用户认证和授权功能
- ❌ PRD 中**未定义**安全相关的 NFR（非功能需求）
- ✅ Epic 9 是为了**支持生产环境部署**的技术基础设施

**测试追溯：**
```
test/integration/epic9/story9_2_*.go  →  Story 9.2 AC  →  SecretProvider 实现
test/integration/epic9/story9_3_*.go  →  Story 9.3 AC  →  审计日志实现
```

**不追溯到：**
```
❌ test/integration/epic9/*  ↛  PRD FR/NFR（PRD 中无对应需求）
```

## 目录结构

```
test/integration/epic9/
├── README.md                    # 本文档
├── story9_2_vault_test.go       # Story 9.2: Vault 集成测试
└── story9_3_audit_test.go       # Story 9.3: 审计日志集成测试
```

## Story 测试详情

### Story 9.2: SecretProvider 接口实现

**测试文件**: `story9_2_vault_test.go`

**测试内容**:
- ✅ AC3: Vault 连接和认证
- ✅ AC3: 密钥读取和缓存
- ✅ AC4: 表达式引擎集成 `${{ secrets.key }}`
- ✅ AC6: 日志脱敏验证

**前置条件**:
```bash
# 启动 Vault（Docker）
docker run -d --name vault-dev \
  -p 8200:8200 \
  -e VAULT_DEV_ROOT_TOKEN_ID=mytoken \
  vault:1.15

# 设置环境变量
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=mytoken
```

**运行测试**:
```bash
go test -v ./test/integration/epic9 -run=TestStory9_2
```

### Story 9.3: 审计日志实现

**测试文件**: `story9_3_audit_test.go`

**测试内容**:
- ✅ AC2: 文件审计日志存储和轮转
- ✅ AC3: API 请求审计中间件
- ✅ AC4: 工作流操作审计
- ✅ AC5: 密钥访问审计
- ✅ AC6: 审计日志查询 API

**运行测试**:
```bash
go test -v ./test/integration/epic9 -run=TestStory9_3
```

## 测试命名规范

### 集成测试

- **格式:** `TestStory{Epic}_{Story}_{AC}_{Feature}`
- **示例:**
  - `TestStory9_2_AC3_VaultConnection`
  - `TestStory9_3_AC5_SecretAccessAudit`

## 与其他测试层级的关系

### Unit Tests（已存在）
- `pkg/secrets/vault_provider_test.go` - Vault Provider 单元测试
- `pkg/audit/file_store_test.go` - 审计文件存储单元测试
- `pkg/middleware/audit_test.go` - 审计中间件单元测试

### Integration Tests（本目录）
- Epic 9 各 Story 的 AC 集成验证
- 跨组件交互测试（SecretProvider + 审计）

### E2E Tests（可选）
- `test/e2e/secure_workflow_test.go` - 端到端加密工作流测试

## 注意事项

### 1. 环境依赖

集成测试需要外部服务：
- **Vault**: 密钥管理测试
- **文件系统**: 审计日志写入

### 2. 测试隔离

- 使用临时目录存储审计日志
- 每个测试用例独立的 Vault 路径
- 测试结束后清理资源

### 3. 敏感信息处理

- ⚠️ 测试中不使用真实密钥
- ✅ 使用 `test-secret-*` 前缀
- ✅ 验证日志脱敏功能

## 参考

- [Story 9.1: HTTPS/TLS 支持](../../../docs/sprint-artifacts/9-1-https-tls-support.md)
- [Story 9.2: SecretProvider 接口](../../../docs/sprint-artifacts/9-2-secretprovider-interface.md)
- [Story 9.3: 审计日志实现](../../../docs/sprint-artifacts/9-3-audit-logging.md)
- [Story 9.4: 安全最佳实践](../../../docs/sprint-artifacts/9-4-security-best-practices-documentation.md)
- [测试标准文档](../../../docs/test-review.md)
