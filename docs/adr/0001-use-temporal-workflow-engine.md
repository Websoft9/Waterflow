# ADR-0001: 使用 Temporal 作为工作流引擎

**状态:** ✅ 已采纳  
**日期:** 2025-12-13  
**决策者:** 架构团队  

## 背景

Waterflow 需要一个可靠的工作流引擎来支持:
- 长时运行的工作流(数小时到数天)
- 进程崩溃后的状态恢复
- 分布式任务调度
- 自动重试和容错

考虑的技术方案:
1. 自研工作流引擎
2. 使用 Temporal
3. 使用 Cadence
4. 使用 Apache Airflow

## 决策

采用 **Temporal** 作为底层工作流引擎。

## 理由

### Temporal 的优势:

1. **持久化执行**
   - Event Sourcing 模式,状态完全持久化
   - 进程重启后自动恢复执行
   - 无需额外状态存储

2. **成熟的分布式调度**
   - Task Queue 机制实现服务器组路由
   - 自动负载均衡
   - Worker 心跳和健康检查

3. **强大的容错能力**
   - 自动重试机制
   - 超时控制
   - 补偿逻辑支持

4. **活跃的社区和企业支持**
   - Uber 开源并在生产环境大规模使用
   - 文档完善,社区活跃
   - 持续维护和更新

### 与其他方案对比:

| 方案 | 优点 | 缺点 | 决策 |
|------|------|------|------|
| **Temporal** | 成熟稳定,功能完整,持久化执行 | 学习曲线,部署复杂度 | ✅ 选择 |
| Cadence | Temporal 的前身,成熟 | 社区活跃度不如 Temporal | ❌ |
| 自研 | 完全可控 | 开发成本高,难以达到生产级 | ❌ |
| Airflow | Python 生态,适合数据流 | 不适合通用工作流,重量级 | ❌ |

## 后果

### 正面影响:

✅ **快速实现** - 专注业务逻辑,不需要实现调度器  
✅ **生产就绪** - Temporal 已在大规模生产环境验证  
✅ **功能完整** - 持久化/重试/超时开箱即用  
✅ **可观测性** - Event History 提供完整执行链路  

### 负面影响:

⚠️ **学习曲线** - 团队需要学习 Temporal 概念(Workflow/Activity/确定性)  
⚠️ **依赖外部服务** - 需要部署和维护 Temporal Server  
⚠️ **架构约束** - 必须遵循 Temporal 的编程模型  

### 风险缓解:

- 团队完成 Temporal 官方教程
- 构建 PoC 验证核心场景
- 制定 Temporal 最佳实践文档

## 替代方案

### 方案 A: 自研工作流引擎

**被拒绝原因:**
- 开发成本高(至少 6 个月)
- 难以达到 Temporal 的成熟度
- 需要自己解决状态持久化/分布式调度等复杂问题

### 方案 B: Apache Airflow

**被拒绝原因:**
- 主要为数据流设计,不适合通用工作流
- Python 技术栈,与 Go 不匹配
- 部署和维护复杂

## 参考资料

- [Temporal 官方文档](https://docs.temporal.io/)
- [Temporal 架构分析](../analysis/temporal-architecture-analysis.md)
- [Temporal vs Cadence 对比](https://docs.temporal.io/docs/cadence-to-temporal/)
