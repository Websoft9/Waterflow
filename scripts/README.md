# Waterflow Scripts

构建、部署、备份和维护脚本集合。

## 部署脚本

### install-server.sh

自动化安装 Waterflow Server（二进制部署）。

**用途**:
- 创建系统用户和目录结构
- 下载最新的 Waterflow Server 二进制文件
- 创建默认配置文件
- 安装并启用 systemd 服务

**使用方法**:
```bash
# 使用默认配置安装
sudo ./scripts/install-server.sh

# 自定义安装目录
sudo INSTALL_DIR=/usr/local/waterflow ./scripts/install-server.sh
```

**环境变量**:
- `INSTALL_DIR`: 安装目录（默认: `/opt/waterflow`）
- `CONFIG_DIR`: 配置目录（默认: `/etc/waterflow`）
- `LOG_DIR`: 日志目录（默认: `/var/log/waterflow`）
- `DATA_DIR`: 数据目录（默认: `/var/lib/waterflow`）

**要求**:
- 需要 root 权限
- 系统支持 systemd
- 已安装 curl, tar

---

## 备份恢复脚本

### backup-database.sh

备份 Temporal PostgreSQL 数据库。

**使用方法**:
```bash
# 使用默认配置备份
./scripts/backup-database.sh

# 自定义备份目录
BACKUP_DIR=/data/backups ./scripts/backup-database.sh
```

**环境变量**:
- `BACKUP_DIR`: 备份目录（默认: `/data/waterflow/backups`）
- `RETENTION_DAYS`: 备份保留天数（默认: 7）
- `COMPOSE_FILE`: docker-compose 文件路径

**备份文件格式**:
```
waterflow_db_YYYYMMDD_HHMMSS.sql.gz
```

### backup-configs.sh

备份配置文件和环境变量。

**使用方法**:
```bash
./scripts/backup-configs.sh
```

**备份内容**:
- `/etc/waterflow/` - 配置目录
- `/opt/waterflow/deployments/.env` - 环境变量
- `/opt/waterflow/deployments/docker-compose.yaml` - Docker Compose 配置

**备份文件格式**:
```
waterflow_configs_YYYYMMDD_HHMMSS.tar.gz
```

### restore-database.sh

从备份恢复数据库。

**使用方法**:
```bash
./scripts/restore-database.sh /data/waterflow/backups/waterflow_db_20260109_120000.sql.gz
```

**警告**:
- 恢复前会停止 Waterflow 和 Temporal 服务
- 当前数据库数据将被覆盖
- 需要用户确认

### restore-configs.sh

从备份恢复配置文件。

**使用方法**:
```bash
./scripts/restore-configs.sh /data/waterflow/backups/waterflow_configs_20260109_120000.tar.gz
```

**警告**:
- 当前配置文件将被覆盖
- 恢复后需要重启服务

### cleanup-old-backups.sh

清理旧备份文件。

**使用方法**:
```bash
# 清理 7 天前的备份
./scripts/cleanup-old-backups.sh

# 清理 30 天前的备份
./scripts/cleanup-old-backups.sh --days 30

# 模拟运行（不实际删除）
./scripts/cleanup-old-backups.sh --dry-run
```

**选项**:
- `-d, --days DAYS`: 保留天数（默认: 7）
- `-p, --path PATH`: 备份目录路径
- `-n, --dry-run`: 模拟运行
- `-h, --help`: 显示帮助

**自动化清理**:
```bash
# 添加到 crontab（每天凌晨 2 点执行）
0 2 * * * /opt/waterflow/scripts/cleanup-old-backups.sh --days 7
```

---

## 升级脚本

### upgrade-docker.sh

升级 Docker Compose 部署的 Waterflow。

**使用方法**:
```bash
./scripts/upgrade-docker.sh
```

**升级流程**:
1. 获取当前版本
2. 创建升级前备份（数据库和配置）
3. 停止服务
4. 拉取最新 Docker 镜像
5. 启动服务
6. 验证升级结果
7. 清理旧镜像

**环境变量**:
- `DEPLOY_DIR`: 部署目录（默认: `/opt/waterflow/deployments`）
- `BACKUP_DIR`: 备份目录（默认: `/data/waterflow/backups`）

**回滚**:
如果升级失败，脚本会自动提示回滚步骤。

### upgrade-binary.sh

升级二进制部署的 Waterflow。

**使用方法**:
```bash
sudo ./scripts/upgrade-binary.sh
```

**升级流程**:
1. 备份当前二进制文件
2. 停止 systemd 服务
3. 下载最新二进制文件
4. 安装新版本
5. 启动服务
6. 验证升级结果

**环境变量**:
- `INSTALL_DIR`: 安装目录（默认: `/opt/waterflow`）
- `DOWNLOAD_URL`: 下载地址

**要求**:
- 需要 root 权限
- 服务由 systemd 管理

---

## 开发脚本

### cleanup.sh

清理开发环境和测试数据。

### deploy-greeter.sh

部署 Greeter 示例工作流。

### verify-agent.sh

验证 Agent 安装和配置。

### test-deployment.sh

测试部署配置。

---

## 监控脚本

### logs.sh

查看 Waterflow 日志。

**使用方法**:
```bash
./scripts/logs.sh
```

### monitor_resources.sh

监控系统资源使用情况。

---

## 测试脚本

### test-retry-ui.sh

测试重试功能和 UI。

### test-template-api.sh

测试模板 API。

---

## 脚本使用最佳实践

### 1. 定期备份

**每日自动备份**（推荐）:
```bash
# /etc/cron.d/waterflow-backup
0 2 * * * root /opt/waterflow/scripts/backup-database.sh
0 2 * * * root /opt/waterflow/scripts/backup-configs.sh
```

### 2. 备份保留策略

```bash
# 每周清理一次，保留 30 天备份
0 3 * * 0 root /opt/waterflow/scripts/cleanup-old-backups.sh --days 30
```

### 3. 升级前准备

```bash
# 1. 创建备份
./scripts/backup-database.sh
./scripts/backup-configs.sh

# 2. 执行升级
./scripts/upgrade-docker.sh  # 或 upgrade-binary.sh

# 3. 验证
curl http://localhost:8080/health
```

### 4. 测试恢复流程

定期测试备份恢复，确保备份可用：

```bash
# 在测试环境恢复备份
./scripts/restore-database.sh /path/to/backup.sql.gz
./scripts/restore-configs.sh /path/to/configs.tar.gz
```

---

## 故障排查

### 备份失败

**检查数据库连接**:
```bash
docker-compose exec postgresql psql -U temporal -d temporal -c "SELECT 1"
```

**检查磁盘空间**:
```bash
df -h /data/waterflow/backups
```

### 升级失败

**查看日志**:
```bash
# Docker Compose
docker-compose logs waterflow

# Systemd
sudo journalctl -u waterflow-server -n 100
```

**回滚到备份版本**:
```bash
# Docker: 使用之前的镜像标签
docker-compose down
docker-compose up -d

# Binary: 从备份恢复
cp /data/waterflow/backups/binaries_YYYYMMDD_HHMMSS/* /opt/waterflow/bin/
systemctl restart waterflow-server
```

---

## 更多信息

- [部署文档](../docs/deployment.md)
- [故障排查文档](../docs/troubleshooting.md)
- [监控指南](../deployments/monitoring/README.md)
