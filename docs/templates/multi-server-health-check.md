# 多服务器健康检查模板 (Multi-Server Health Check)

## 概述

多服务器健康检查模板通过 SSH 远程执行,并行检查多台服务器的健康状况,收集 CPU、内存、磁盘使用率等关键指标,并生成统一的 Markdown 格式健康报告。这是实现大规模服务器批量巡检的最佳方案。

**模板特点:**
- ✅ **并行执行** - Matrix 策略同时检查所有服务器,节省时间
- ✅ **SSH 远程执行** - 无需在目标服务器安装 Agent
- ✅ **跨平台兼容** - 支持 Ubuntu、CentOS、Debian 等主流 Linux 发行版
- ✅ **自动报告生成** - Python 脚本生成可读性强的 Markdown 报告
- ✅ **阈值告警** - 自动标记超过阈值的服务器
- ✅ **容错设计** - 单台服务器失败不影响其他检查

**工作流程:**
1. Matrix 策略扩展服务器列表为并行 Job
2. 通过 SSH 连接每台服务器
3. 执行 CPU/内存/磁盘使用率检测命令
4. 将结果保存为 JSON 文件
5. 聚合所有 JSON 文件生成 Markdown 报告
6. 标记告警项 (CPU/内存/磁盘超阈值)

---

## 前置条件

### 1. Waterflow 环境

- ✅ Waterflow Server 已部署
- ✅ 至少一个 Agent 已连接 (通常为本地或跳板机)
- ✅ Agent 可访问所有目标服务器网络

**验证环境:**
```bash
# 检查 Server 运行状态
curl http://localhost:8080/health

# 检查 Agent 连接
waterflow node-list
```

### 2. SSH 免密登录配置

**关键要求:** 必须配置从 Agent 服务器到所有目标服务器的 SSH 免密登录

**配置步骤:**

```bash
# 1. 在 Agent 服务器生成 SSH 密钥 (如未生成)
ssh-keygen -t ed25519 -C "waterflow-agent" -f ~/.ssh/id_ed25519 -N ""

# 2. 将公钥复制到目标服务器
ssh-copy-id root@server1
ssh-copy-id root@server2
ssh-copy-id root@server3

# 3. 验证免密登录
ssh root@server1 "echo 'SSH connection successful'"
```

**多服务器批量配置:**

```bash
# 批量复制公钥
for server in server1 server2 server3; do
  ssh-copy-id root@$server
done

# 或使用 expect 自动化输入密码 (需安装 expect)
#!/usr/bin/expect
set password "your-password"
set servers [list "server1" "server2" "server3"]

foreach server $servers {
  spawn ssh-copy-id root@$server
  expect "password:"
  send "$password\r"
  expect eof
}
```

**故障排查:**

如果 `ssh-copy-id` 失败,手动追加公钥:
```bash
# 查看公钥
cat ~/.ssh/id_ed25519.pub

# SSH 到目标服务器
ssh root@server1

# 手动追加公钥
mkdir -p ~/.ssh
echo "ssh-ed25519 AAAAC3... waterflow-agent" >> ~/.ssh/authorized_keys
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

### 3. Python 3 环境

**用途:** 在 Agent 服务器上运行报告生成脚本

```bash
# 验证 Python 安装
python3 --version
# 预期输出: Python 3.x.x

# 如未安装 (Ubuntu/Debian)
sudo apt update && sudo apt install -y python3

# CentOS/RHEL
sudo yum install -y python3
```

**无需安装额外依赖:** 报告生成脚本仅使用 Python 标准库

### 4. 目标服务器要求

| 要求 | 说明 | 验证命令 |
|------|------|----------|
| **Linux 操作系统** | 支持 Ubuntu、CentOS、Debian、RHEL 等 | `uname -a` |
| **SSH 服务运行** | 默认 22 端口 (可自定义) | `systemctl status sshd` |
| **基础命令可用** | `free`, `df`, `top` 或 `mpstat` | `which free df top` |

**可选优化:** 安装 `sysstat` 包获取更精确的 CPU 指标
```bash
# Ubuntu/Debian
sudo apt install -y sysstat

# CentOS/RHEL
sudo yum install -y sysstat
```

---

## 参数说明

### 服务器列表配置

服务器列表定义在 `vars.servers`，然后通过 Matrix 策略引用：

```yaml
vars:
  servers: ["web-1", "web-2", "db-1"]  # 修改为你的服务器列表

jobs:
  health-check:
    strategy:
      matrix:
        server: ${{ vars.servers }}  # 引用 vars 中的服务器列表
```

**支持格式:**
- 主机名: `["web-1", "web-2"]`
- IP 地址: `["192.168.1.10", "192.168.1.11"]`
- FQDN: `["web-1.example.com", "db-1.example.com"]`
- 混合格式: `["web-1", "192.168.1.20", "db.example.com"]`

### 可配置参数 (vars)

| 参数 | 默认值 | 说明 | 示例 |
|------|--------|------|------|
| `servers` | `["web-1", "web-2", "db-1"]` | 服务器列表（数组） | `["server1", "192.168.1.10"]` |
| `ssh_user` | `root` | SSH 登录用户名 | `ubuntu` |
| `cpu_threshold` | `80` | CPU 使用率告警阈值 (%) | `75` |
| `memory_threshold` | `85` | 内存使用率告警阈值 (%) | `80` |
| `disk_threshold` | `90` | 磁盘使用率严重阈值 (%) | `85` |
| `report_path` | `/tmp/health_report.md` | 报告保存路径 | `/var/reports/health.md` |

### runs-on 配置

```yaml
runs-on: localhost  # 在本地或跳板机执行,通过 SSH 连接远程服务器
```

**注意:** 此模板不需要在目标服务器部署 Agent,所有检查通过 SSH 远程执行

---

## 使用示例

### 示例 1: 检查 3 台 Web 服务器

**场景:** 定期检查 Web 服务器集群健康状态

**配置文件: `web-servers-health-check.yaml`**

```yaml
name: Web Servers Health Check
on: push

vars:
  servers:
    - "web-1.example.com"
    - "web-2.example.com"
    - "web-3.example.com"
  ssh_user: "ubuntu"
  cpu_threshold: 80
  memory_threshold: 85
  disk_threshold: 90
  report_path: "/tmp/web_health_report.md"

jobs:
  health-check:
    strategy:
      matrix:
        server: ${{ vars.servers }}
    
    runs-on: localhost
    steps:
      # ... (模板步骤保持不变)
```

**提交工作流:**
```bash
waterflow submit web-servers-health-check.yaml
```

**查看报告:**
```bash
cat /tmp/web_health_report.md
```

**预期报告示例:**
```markdown
# Multi-Server Health Check Report

Generated: 2024-01-15 10:30:00

## Summary
- Total Servers: 3
- Healthy: 2
- Warnings: 1
- Critical: 0

## Server Details

### web-1.example.com
- **Status:** ✅ Healthy
- **CPU:** 45.2%
- **Memory:** 62.8%
- **Disk:** 55%

### web-2.example.com
- **Status:** ⚠️ Warning
- **CPU:** 82.1% (Threshold: 80%)
- **Memory:** 78.3%
- **Disk:** 68%

### web-3.example.com
- **Status:** ✅ Healthy
- **CPU:** 38.5%
- **Memory:** 55.2%
- **Disk:** 43%
```

---

### 示例 2: 生产环境混合服务器检查

**场景:** 检查 Web、数据库、缓存服务器,使用严格阈值

**配置文件: `production-health-check.yaml`**

```yaml
name: Production Environment Health Check
on: push

vars:
  servers:
    - "prod-web-1"
    - "prod-web-2"
    - "prod-db-master"
    - "prod-db-replica"
    - "prod-redis-1"
    - "prod-redis-2"
  ssh_user: "prod-admin"
  cpu_threshold: 70     # 生产环境更严格
  memory_threshold: 75
  disk_threshold: 80
  report_path: "/var/waterflow/reports/prod_health.md"

jobs:
  health-check:
    strategy:
      matrix:
        server: ${{ vars.servers }}
    
    runs-on: bastion-server  # 通过跳板机执行
    steps:
      # ... (模板步骤)
```

**定时执行 (Cron 触发):**

修改触发器为定时任务:
```yaml
on:
  schedule:
    cron: "0 */6 * * *"  # 每 6 小时执行一次
```

**集成告警:**

添加 Slack 通知步骤:
```yaml
jobs:
  # ... (health-check job)
  
  notify:
    needs: health-check
    runs-on: localhost
    steps:
      - name: Send report to Slack
        uses: http/request@v1
        with:
          url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
          method: POST
          body: |
            {
              "text": "🩺 Production Health Check Completed",
              "attachments": [{
                "color": "good",
                "text": "View full report: /var/waterflow/reports/prod_health.md"
              }]
            }
```

---

### 示例 3: IP 地址服务器检查

**场景:** 检查通过 IP 地址访问的服务器 (无 DNS 解析)

**配置文件: `ip-based-health-check.yaml`**

```yaml
name: IP-Based Server Health Check
on: push

vars:
  ssh_user: "root"
  cpu_threshold: 80
  memory_threshold: 85
  disk_threshold: 90

jobs:
  health-check:
    strategy:
      matrix:
        server:
          - "192.168.1.10"
          - "192.168.1.11"
          - "192.168.1.12"
          - "10.0.0.20"
    
    runs-on: localhost
    steps:
      # ... (模板步骤)
```

**使用非标准 SSH 端口:**

修改 SSH 命令:
```yaml
- name: Check CPU usage via SSH
  uses: exec/shell@v1
  with:
    command: |
      ssh -p 2222 -o StrictHostKeyChecking=no ${{ vars.ssh_user }}@${{ matrix.server }} '
        # ... (检测命令)
      '
```

---

## 定制指南

### 1. 添加自定义指标

除 CPU/内存/磁盘外,收集其他指标:

**添加网络流量检查:**

```yaml
- name: Check network traffic
  id: network
  uses: exec/shell@v1
  with:
    command: |
      ssh ${{ vars.ssh_user }}@${{ matrix.server }} '
        # 获取网络流量 (bytes received + transmitted)
        cat /proc/net/dev | grep eth0 | awk "{print \$2 + \$10}"
      '
```

**添加进程数检查:**

```yaml
- name: Check process count
  id: processes
  uses: exec/shell@v1
  with:
    command: |
      ssh ${{ vars.ssh_user }}@${{ matrix.server }} '
        ps aux | wc -l
      '
```

**添加服务状态检查:**

```yaml
- name: Check critical services
  id: services
  uses: exec/shell@v1
  with:
    command: |
      ssh ${{ vars.ssh_user }}@${{ matrix.server }} '
        # 检查 nginx、docker、mysql 服务状态
        for service in nginx docker mysql; do
          systemctl is-active $service 2>/dev/null || echo "$service: inactive"
        done
      '
```

### 2. 自定义报告格式

修改报告生成脚本,添加图表或更多细节:

```yaml
- name: Generate enhanced report
  uses: exec/shell@v1
  with:
    command: |
      python3 << 'EOF'
      import json
      import glob
      from datetime import datetime

      # 读取所有 JSON 文件
      results = []
      for file in glob.glob("/tmp/waterflow-health-check/health_check_*.json"):
          with open(file) as f:
              results.append(json.load(f))

      # 生成增强报告
      with open("/tmp/health_report_enhanced.md", "w") as report:
          report.write("# 服务器健康检查报告\n\n")
          report.write(f"**生成时间:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
          
          # 统计汇总
          total = len(results)
          healthy = sum(1 for r in results if float(r.get('cpu', 0)) < 80)
          
          report.write("## 📊 统计概览\n\n")
          report.write(f"- 总服务器数: **{total}**\n")
          report.write(f"- 健康: **{healthy}** ({healthy/total*100:.1f}%)\n")
          report.write(f"- 告警: **{total - healthy}** ({(total-healthy)/total*100:.1f}%)\n\n")
          
          # 服务器详情表格
          report.write("## 📋 服务器详情\n\n")
          report.write("| 服务器 | CPU | 内存 | 磁盘 | 状态 |\n")
          report.write("|--------|-----|------|------|------|\n")
          
          for r in sorted(results, key=lambda x: x['server']):
              cpu = r.get('cpu', 'N/A')
              mem = r.get('memory', 'N/A')
              disk = r.get('disk', 'N/A')
              status = "✅" if isinstance(cpu, (int, float)) and cpu < 80 else "⚠️"
              report.write(f"| {r['server']} | {cpu}% | {mem}% | {disk}% | {status} |\n")
          
          # 告警详情
          warnings = [r for r in results if isinstance(r.get('cpu'), (int, float)) and float(r['cpu']) >= 80]
          if warnings:
              report.write("\n## ⚠️ 告警详情\n\n")
              for w in warnings:
                  report.write(f"### {w['server']}\n")
                  report.write(f"- CPU 使用率: **{w['cpu']}%** (阈值: 80%)\n")
                  report.write(f"- 建议: 检查高负载进程,考虑扩容\n\n")
      
      print("Enhanced report generated: /tmp/health_report_enhanced.md")
      EOF
```

### 3. 使用 Ansible 提升效率

对于大规模服务器 (100+),考虑结合 Ansible:

```yaml
- name: Health check using Ansible
  uses: exec/shell@v1
  with:
    command: |
      # 创建 Ansible Inventory
      cat > /tmp/inventory <<EOF
      [servers]
      ${{ join(matrix.server, '\n') }}
      EOF
      
      # 执行 Ansible Ad-Hoc 命令
      ansible all -i /tmp/inventory -m shell -a "
        echo CPU: \$(top -bn1 | grep Cpu | awk '{print 100-\$8}');
        echo MEM: \$(free | awk '/Mem:/ {printf(\"%.2f\", \$3/\$2*100)}');
        echo DISK: \$(df -h / | tail -1 | awk '{print \$5}' | tr -d '%')
      " --user ${{ vars.ssh_user }}
```

### 4. 添加硬件信息收集

收集 CPU 型号、内存总量等静态信息:

```yaml
- name: Collect hardware info
  uses: exec/shell@v1
  with:
    command: |
      ssh ${{ vars.ssh_user }}@${{ matrix.server }} '
        echo "CPU Model: $(lscpu | grep "Model name" | cut -d: -f2 | xargs)"
        echo "CPU Cores: $(nproc)"
        echo "Total Memory: $(free -h | awk "/Mem:/ {print \$2}")"
        echo "OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2 | tr -d \")"
      '
```

### 5. 集成监控系统

导出结果到 Prometheus 或其他监控系统:

```yaml
- name: Export to Prometheus Pushgateway
  uses: http/request@v1
  with:
    url: "http://pushgateway:9091/metrics/job/waterflow_health_check/instance/${{ matrix.server }}"
    method: POST
    body: |
      server_cpu_usage{server="${{ matrix.server }}"} ${{ steps.cpu.outputs.result }}
      server_memory_usage{server="${{ matrix.server }}"} ${{ steps.memory.outputs.result }}
      server_disk_usage{server="${{ matrix.server }}"} ${{ steps.disk.outputs.result }}
```

---

## 故障排查

### 常见问题

#### 1. SSH 连接失败

**错误信息:**
```
Permission denied (publickey,password)
```

**原因:** SSH 免密登录未配置或公钥权限错误

**解决方案:**

```bash
# 1. 重新复制公钥
ssh-copy-id -i ~/.ssh/id_ed25519.pub root@server1

# 2. 验证公钥权限 (在目标服务器)
ls -la ~/.ssh/authorized_keys
# 预期: -rw------- (权限 600)

chmod 600 ~/.ssh/authorized_keys
chmod 700 ~/.ssh

# 3. 检查 SSH 配置 (在目标服务器)
sudo grep -E "PubkeyAuthentication|PasswordAuthentication" /etc/ssh/sshd_config
# 确保:
# PubkeyAuthentication yes
# PasswordAuthentication yes  (测试后可改为 no)

sudo systemctl restart sshd
```

**临时解决:** 使用密码登录 (不推荐)
```yaml
vars:
  ssh_user: "root"
  ssh_password: "your-password"  # 使用 Waterflow Secrets 管理

steps:
  - name: Check CPU with password
    uses: exec/shell@v1
    with:
      command: |
        sshpass -p "${{ vars.ssh_password }}" ssh ${{ vars.ssh_user }}@${{ matrix.server }} '...'
```

---

#### 2. 命令返回 "N/A"

**原因:** 目标服务器缺少必要命令或命令执行失败

**诊断:**

```bash
# 手动 SSH 到目标服务器
ssh root@server1

# 测试每个命令
free | awk '/Mem:/ {printf("%.2f", $3/$2 * 100.0)}'
df -P / | tail -1 | awk '{print $5}' | tr -d '%'
top -bn1 | grep -i "cpu" | head -1
```

**解决方案:**

**安装缺失工具:**
```bash
# Ubuntu/Debian
sudo apt install -y sysstat procps

# CentOS/RHEL
sudo yum install -y sysstat procps-ng
```

**修改检测命令:**

如果某些发行版命令输出不同,调整 awk 脚本:
```yaml
- name: Check memory (compatible)
  uses: exec/shell@v1
  with:
    command: |
      ssh ${{ vars.ssh_user }}@${{ matrix.server }} '
        # 兼容多种发行版
        if free | grep -q "Mem:"; then
          free | awk "/Mem:/ {printf(\"%.2f\", \$3/\$2*100)}"
        else
          awk "/MemTotal/{t=\$2} /MemAvailable/{a=\$2} END{printf(\"%.2f\", (t-a)/t*100)}" /proc/meminfo
        fi
      '
```

---

#### 3. 报告生成失败

**错误信息:**
```
FileNotFoundError: [Errno 2] No such file or directory: '/tmp/waterflow-health-check/health_check_*.json'
```

**原因:** 前面的检查步骤未成功保存 JSON 文件

**解决方案:**

**1. 检查 JSON 文件是否存在:**
```bash
ls -la /tmp/waterflow-health-check/
```

**2. 验证保存步骤:**
```yaml
- name: Save health check results
  uses: exec/shell@v1
  with:
    command: |
      # 添加调试输出
      mkdir -p /tmp/waterflow-health-check
      
      JSON_FILE="/tmp/waterflow-health-check/health_check_${{ matrix.server }}.json"
      echo "Saving to: $JSON_FILE"
      
      # 保存 JSON
      cat > $JSON_FILE <<EOF
      {
        "server": "${{ matrix.server }}",
        "cpu": "$CPU_RESULT",
        "memory": "$MEMORY_RESULT",
        "disk": "$DISK_RESULT"
      }
      EOF
      
      # 验证文件创建
      ls -la $JSON_FILE
      cat $JSON_FILE
```

**3. 确保报告生成步骤等待所有检查完成:**

使用 `needs` 依赖:
```yaml
jobs:
  health-check:
    # ... (matrix 并行检查)
  
  generate-report:
    needs: health-check  # 等待所有检查完成
    runs-on: localhost
    steps:
      - name: Generate report
        # ...
```

---

#### 4. 部分服务器检查超时

**现象:** 某些服务器返回 "N/A",其他正常

**原因:** 网络延迟、服务器负载高或防火墙阻塞

**解决方案:**

**1. 增加超时时间:**
```yaml
- name: Check CPU usage via SSH
  uses: exec/shell@v1
  with:
    command: |
      ssh -o ConnectTimeout=30 ${{ vars.ssh_user }}@${{ matrix.server }} '...'
  timeout-minutes: 3  # 增加到 3 分钟
```

**2. 添加重试策略:**
```yaml
- name: Check CPU with retry
  uses: exec/shell@v1
  with:
    command: |
      ssh ${{ vars.ssh_user }}@${{ matrix.server }} '...'
  retry-strategy:
    max-attempts: 3
    initial-interval: 5s
```

**3. 检查网络和防火墙:**
```bash
# 从 Agent 服务器 ping 目标服务器
ping -c 3 server1

# 检查 SSH 端口
telnet server1 22

# 检查防火墙规则 (在目标服务器)
sudo iptables -L -n | grep 22
sudo firewall-cmd --list-all
```

---

#### 5. Matrix 并发限制

**现象:** 大量服务器 (100+) 时部分 Job 等待

**原因:** Waterflow 默认并发限制保护系统资源

**解决方案:**

**1. 增加并发限制 (如 Waterflow 支持):**
```yaml
jobs:
  health-check:
    strategy:
      matrix:
        server: ["server1", "server2", ...]  # 100 台服务器
      max-parallel: 50  # 最多同时运行 50 个 Job
```

**2. 分批检查:**

将服务器分组,分批提交工作流:
```bash
# 脚本: batch_health_check.sh
#!/bin/bash

SERVERS=("server1" "server2" ... "server100")
BATCH_SIZE=20

for ((i=0; i<${#SERVERS[@]}; i+=BATCH_SIZE)); do
  batch=("${SERVERS[@]:i:BATCH_SIZE}")
  
  # 生成临时工作流文件
  cat > /tmp/batch_$i.yaml <<EOF
name: Health Check Batch $i
jobs:
  health-check:
    strategy:
      matrix:
        server: [$(printf '"%s",' "${batch[@]}" | sed 's/,$//')]
    # ... (其他配置)
EOF
  
  # 提交工作流
  waterflow submit /tmp/batch_$i.yaml
done
```

---

## 最佳实践

### 1. 定期自动执行

使用 Cron 触发器定时执行:

```yaml
on:
  schedule:
    - cron: "0 2 * * *"   # 每天凌晨 2 点执行
    - cron: "0 */6 * * *"  # 每 6 小时执行一次
```

### 2. 分级告警

根据严重程度分级处理:

```yaml
vars:
  cpu_warning: 70   # 警告阈值
  cpu_critical: 90  # 严重阈值
  memory_warning: 75
  memory_critical: 90
```

报告生成时标记严重级别:
```python
if cpu > cpu_critical:
    status = "🔴 Critical"
elif cpu > cpu_warning:
    status = "⚠️ Warning"
else:
    status = "✅ Healthy"
```

### 3. 保留历史报告

生成带时间戳的报告:

```yaml
vars:
  report_path: "/var/waterflow/reports/health_$(date +%Y%m%d_%H%M%S).md"
```

定期清理旧报告:
```bash
# 保留最近 30 天
find /var/waterflow/reports/ -name "health_*.md" -mtime +30 -delete
```

### 4. 集成监控系统

将结果导出到 Grafana、Prometheus:

```yaml
- name: Export to monitoring
  uses: http/request@v1
  with:
    url: "http://prometheus-pushgateway:9091/metrics"
    method: POST
    body: |
      server_cpu{server="${{ matrix.server }}"} ${{ steps.cpu.outputs.result }}
```

### 5. 使用 SSH Config 简化配置

在 `~/.ssh/config` 配置别名和选项:

```bash
# ~/.ssh/config
Host web-*
  User ubuntu
  Port 22
  StrictHostKeyChecking no
  ConnectTimeout 10

Host db-*
  User postgres
  Port 2222
  IdentityFile ~/.ssh/db_key
```

工作流中直接使用别名:
```yaml
strategy:
  matrix:
    server: ["web-1", "web-2", "db-1"]
```

### 6. 安全加固

**限制 SSH 权限:**
```bash
# 创建专用用户 (在目标服务器)
sudo useradd -m -s /bin/bash waterflow-monitor
sudo usermod -aG sudo waterflow-monitor

# 仅允许只读命令
# 在 /etc/sudoers.d/waterflow-monitor:
waterflow-monitor ALL=(ALL) NOPASSWD: /usr/bin/top, /usr/bin/free, /bin/df
```

**使用专用密钥:**
```bash
# 为健康检查生成专用密钥
ssh-keygen -t ed25519 -f ~/.ssh/waterflow_health_check -C "health-check-key"
```

---

## 参考资源

- **Waterflow 文档:**
  - [快速开始](../quick-start.md)
  - [Matrix 策略文档](../matrix-strategy.md)
  - [表达式系统](../expression-system.md)

- **节点文档:**
  - [exec/shell 节点](../nodes/exec-shell.md)
  - [条件执行 (if)](../conditional-execution.md)

- **SSH 配置:**
  - [OpenSSH 官方文档](https://www.openssh.com/manual.html)
  - [SSH 免密登录配置](https://www.ssh.com/academy/ssh/copy-id)

- **相关模板:**
  - [单服务器部署](./single-server-deployment.md)
  - [分布式栈部署](./distributed-stack-deployment.md)

---

**上一页:** [单服务器部署模板](./single-server-deployment.md)  
**下一页:** [分布式栈部署模板](./distributed-stack-deployment.md)
