# E2E (End-to-End) Tests

本目录包含端到端测试，验证从用户输入到系统输出的完整技术管道。

## 定义

E2E 测试覆盖完整流程：
- **范围:** YAML 提交 → DSL 解析 → Temporal 工作流 → Event History → API 查询
- **环境:** 真实 Temporal Server + PostgreSQL + Waterflow Server + Agent
- **特点:** 无 Mock，验证 Event Sourcing 状态持久化

## 与其他测试层级的区别

| 测试层级 | 范围 | 环境 | 示例 |
|---------|------|------|------|
| **Unit** | 单个函数/类 | 内存 | `pkg/dsl/parser_test.go` |
| **Integration** | 组件交互 | 部分真实环境 | `test/integration/epic1/story1_1_test.go` |
| **E2E** | 完整 Story 管道 | 完整真实环境 | `test/e2e/story1_3_yaml_dsl_test.go` |
| **Acceptance** | PRD 业务场景 | 生产级环境 | `test/acceptance/health_check_test.go` |

## 目录结构

```
test/e2e/
├── README.md                      # 本文档
├── common_test.go                 # E2E 测试基础设施
├── docker-compose.e2e.yaml        # E2E 测试环境配置
├── quick-start.sh                 # 快速启动脚本
├── story1_3_yaml_dsl_test.go      # Story 1.3: YAML DSL 解析和验证
├── story1_4_expressions_test.go   # Story 1.4: 表达式引擎
├── story1_5_conditions_test.go    # Story 1.5: 条件执行
├── story1_6_matrix_test.go        # Story 1.6: Matrix 策略
├── story1_7_timeout_retry_test.go # Story 1.7: 超时和重试
├── story1_8_temporal_integration_test.go # Story 1.8: Temporal 集成
├── story1_9_api_test.go           # Story 1.9: 工作流 API
├── story1_10_schedule_api_test.go # Story 1.10: 定时调度 API
├── story1_11_webhook_test.go      # Story 1.11: Webhook 触发器
├── reports/                       # 测试报告输出目录
└── testdata/
    └── workflows/                 # 测试用 YAML 工作流
```

## 运行测试

### 方式一：使用 Make (推荐)

```bash
# 完整运行：启动环境、运行测试、清理
make e2e-test

# 仅运行测试（环境已启动）
make e2e-test-only
```

### 方式二：使用脚本

```bash
# 完整运行
./scripts/run-e2e-tests.sh

# 保留环境用于调试
./scripts/run-e2e-tests.sh --keep-env

# 运行特定测试
./scripts/run-e2e-tests.sh --test TestStory1_3_E2E_001
```

### 方式三：手动运行

```bash
# 1. 启动 E2E 测试环境
docker compose -f test/e2e/docker-compose.e2e.yaml up -d

# 2. 等待服务就绪
# Server: http://localhost:18080/health
# Temporal: localhost:17233

# 3. 运行测试
SERVER_URL=http://localhost:18080 \
TEMPORAL_HOST=localhost:17233 \
go test -v -tags=e2e ./test/e2e/...

# 4. 清理环境
docker compose -f test/e2e/docker-compose.e2e.yaml down -v
```

## 测试命名规范

### 文件命名

- 格式: `story{epic}_{story}_{feature}_test.go`
- 示例: `story1_3_yaml_dsl_test.go` (Epic 1, Story 3)

### 测试函数命名

- 格式: `TestStory{Epic}_{Story}_E2E_{Sequence}_{Description}`
- 示例: `TestStory1_3_E2E_001_ComplexYAMLWorkflowExecution`

### Test ID 格式

- 格式: `{Epic}.{Story}-E2E-{Sequence}`
- 示例: `1.3-E2E-001`, `1.3-E2E-002`

## 编写 E2E 测试

E2E 测试验证完整的技术管道，应该：

1. **使用真实环境** - Temporal, PostgreSQL, Server
2. **验证端到端流程** - YAML → 执行 → Event History → API 查询
3. **验证 Event Sourcing** - 确保状态正确持久化
4. **清理测试数据** - 每个测试独立，不影响其他测试

## 环境要求

### 依赖服务

- Docker 和 Docker Compose
- Temporal Server (通过 docker-compose 启动)
- PostgreSQL (Event Sourcing 存储)

### 端口

- **Server:** 18080 (HTTP)
- **Temporal:** 17233 (gRPC)
- **PostgreSQL:** 15432

## 辅助文件

### common_test.go

提供 E2E 测试基础设施：
- `E2ETestSuite` - 测试环境管理
- `setupE2EEnvironment()` - 环境初始化
- `teardownE2EEnvironment()` - 环境清理
- `waitForServerReady()` - 等待服务就绪

### docker-compose.e2e.yaml

E2E 测试专用 Docker Compose 配置：
- 独立端口避免冲突
- 持久化 Event History
- 网络隔离

### quick-start.sh

快速启动 E2E 测试环境的便捷脚本。

## 参考

- [测试标准文档](../../docs/test-review.md)
- [Epic 1 定义](../../docs/epics.md#epic-1)
