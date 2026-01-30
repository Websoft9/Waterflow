# Stress 目录重构方案

## 问题诊断

### 1. BMad 测试金字塔违规

**当前状态**:
- ✅ 4个 Go 测试文件（符合金字塔规范）
- ❌ 9个 Shell 脚本（违反测试金字塔 - Stress应为Go性能测试）

**违规点**:
```
test/stress/
├── ✅ concurrent_workflows_test.go
├── ✅ event_sourcing_test.go
├── ✅ event_history_integrity_test.go
├── ✅ memory_profile_test.go
├── ❌ concurrent_workflows_test.sh       # 与Go测试重复
├── ❌ server_crash_recovery_test.sh       # 应转为Go测试
├── ❌ resource_leak_test.sh               # 应转为Go测试
├── ❌ timeout_retry_test.sh               # 应转为Go测试
├── ❌ agent_reconnect_test.sh             # 应转为Go测试
├── ❌ temporal_reconnect_test.sh          # 应转为Go测试
├── ❌ mixed_fault_test.sh                 # 应转为Go测试
├── ❌ run_all_tests.sh                    # 用go test -tags=stress替代
└── README.md
```

### 2. Tea 追溯标准违规

**问题**: README.md 声称追溯 AC1-AC5，但 PRD 无 AC 编号定义

**PRD 实际表述**:
- "崩溃测试中 100% 状态恢复" (质量属性表)
- "支持 ≥100 个并发 Agent 连接" (质量属性表)
- "每个节点可配置重试策略" (质量属性表)

**应追溯格式**:
```
PRD#L235: 持久性 - 崩溃测试中 100% 状态恢复
PRD#L238: 可扩展性 - 支持 ≥100 个并发 Agent 连接
PRD#L249: 容错能力 - 自动重试和超时策略
```

### 3. 文件冗余

**重复测试**:
- `concurrent_workflows_test.go` + `concurrent_workflows_test.sh` (同一测试两种实现)

**Shell脚本应转为Go**:
- `server_crash_recovery_test.sh` → 转为 Go (使用 exec.Command 控制进程)
- `resource_leak_test.sh` → 转为 Go (使用 runtime.MemStats)
- `timeout_retry_test.sh` → 转为 Go (使用 context.WithTimeout)

## 重构方案：方案 A（推荐）

### 操作步骤

#### 1. 删除 Shell 脚本（保留 README.md）

```bash
rm test/stress/*.sh
```

**理由**:
- BMad 测试金字塔：Stress 层应为 **Go benchmark/测试**
- Shell 脚本无法集成 `go test` 工具链
- 重复测试增加维护成本

#### 2. 扩展 Go 测试覆盖

**补充测试文件**:
```
test/stress/
├── concurrent_workflows_test.go      # ✅ 已存在 (PRD#L238)
├── event_sourcing_test.go            # ✅ 已存在 (PRD#L235)
├── event_history_integrity_test.go   # ✅ 已存在 (PRD#L235)
├── memory_profile_test.go            # ✅ 已存在
├── server_crash_recovery_test.go     # 🆕 从.sh转换 (PRD#L235)
├── resource_leak_test.go             # 🆕 从.sh转换
├── timeout_retry_test.go             # 🆕 从.sh转换 (PRD#L249)
├── agent_reconnect_test.go           # 🆕 从.sh转换
├── temporal_reconnect_test.go        # 🆕 从.sh转换
├── mixed_fault_test.go               # 🆕 从.sh转换
└── README.md                          # 🔄 更新追溯
```

#### 3. 更新 README.md

**追溯表格**:
| 测试文件 | PRD 追溯 | 验证目标 |
|---------|---------|---------|
| concurrent_workflows_test.go | [PRD#L238](../../docs/prd.md#L238) | 可扩展性 - 支持 ≥100 并发 Agent |
| event_sourcing_test.go | [PRD#L235](../../docs/prd.md#L235) | 持久性 - 100% 状态恢复 |
| server_crash_recovery_test.go | [PRD#L235](../../docs/prd.md#L235) | 持久性 - 进程重启不丢失状态 |
| timeout_retry_test.go | [PRD#L249](../../docs/prd.md#L249) | 容错 - 自动重试和超时策略 |

#### 4. 运行方式统一

**Shell 脚本方式** (删除):
```bash
./run_all_tests.sh
```

**Go 测试方式** (标准):
```bash
go test -tags=stress -v ./test/stress/...
go test -bench=. -benchmem ./test/stress/...
```

### 关键代码转换示例

#### Shell → Go (Server Crash Recovery)

**之前 (server_crash_recovery_test.sh)**:
```bash
kill -9 $SERVER_PID
sleep 5
./bin/server &
```

**之后 (server_crash_recovery_test.go)**:
```go
func TestServerCrashRecovery(t *testing.T) {
    // 追溯: PRD#L235 - 持久性: 崩溃测试中 100% 状态恢复
    cmd := exec.Command("./bin/server")
    cmd.Start()
    
    time.Sleep(5 * time.Second)
    cmd.Process.Kill() // 模拟崩溃
    
    // 验证状态恢复...
}
```

## 重构收益

### 合规性提升

| 维度 | 重构前 | 重构后 |
|-----|-------|-------|
| BMad 测试金字塔 | ❌ 混合 Go + Shell | ✅ 纯 Go 测试 |
| Tea 追溯标准 | ❌ 虚假 AC1-AC5 | ✅ 追溯 PRD#Lxxx |
| 测试工具链 | ❌ 需手动执行 Shell | ✅ `go test` 统一 |
| CI/CD 集成 | ⚠️ 需额外脚本 | ✅ Go 原生支持 |

### 维护成本降低

- **-9 Shell 脚本** → 转为 Go 测试
- **-1 run_all_tests.sh** → 用 `go test ./test/stress/...` 替代
- **统一测试框架** → 所有测试用 testing.T

## 执行建议

**方案 A（推荐）**:
1. ✅ 删除所有 Shell 脚本
2. ✅ 扩展 Go 测试覆盖 Shell 功能
3. ✅ 更新 README.md 追溯 PRD 行号

**方案 B（保守）**:
1. 保留 Shell 脚本但标记为 deprecated
2. 逐步迁移到 Go 测试

**推荐方案 A** 的原因：
- BMad 标准明确要求测试金字塔规范
- Go 测试可完全替代 Shell 脚本功能
- 消除技术债务，避免双重维护
