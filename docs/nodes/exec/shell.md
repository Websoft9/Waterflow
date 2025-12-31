# exec/shell@v1

**分类**: exec (命令执行)  
**版本**: v1  
**状态**: stable

## 概述

exec/shell 节点用于在 Agent 服务器上执行 Shell 命令。支持命令参数、环境变量、工作目录和超时控制。

适用于调用系统命令、工具集成、脚本执行等场景。这是最基础、最常用的节点，90% 的工作流会使用它。

## 使用场景

- **系统管理**: 查看系统状态、管理服务、文件操作
- **工具调用**: 调用 CLI 工具（git, aws, kubectl, terraform）
- **健康检查**: 检查服务端口、进程状态
- **部署任务**: 执行部署脚本、重启服务

## 参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| command | string | ✅ | - | 要执行的命令 |
| args | array | ❌ | [] | 命令参数 |
| env | object | ❌ | {} | 环境变量 |
| workdir | string | ❌ | "" | 工作目录 |
| timeout | string | ❌ | 5m | 超时时间 |

### 参数详细说明

**command** (string, required)
- 要执行的命令名称
- 建议使用绝对路径避免 PATH 问题
- 示例: `"ls"`, `"/usr/bin/git"`, `"echo"`

**args** (array, optional, default: [])
- 命令参数列表
- 按顺序传递给命令
- 参数会自动转义，防止注入攻击
- 示例: `["-la", "/tmp"]`, `["--version"]`

**env** (object, optional, default: {})
- 额外的环境变量
- 格式: key-value 对象
- 继承 Agent 进程的环境变量
- 示例: `{"DEBUG": "true", "API_URL": "https://api.example.com"}`

**workdir** (string, optional, default: "")
- 命令执行的工作目录
- 使用绝对路径
- 目录必须存在，否则执行失败
- 示例: `"/app/data"`, `"/opt/deploy"`

**timeout** (string, optional, default: 5m)
- 命令执行超时时间
- 格式: "30s", "5m", "1h"
- 超时后命令被终止（SIGKILL）
- 建议根据命令复杂度调整（简单命令 30s，复杂操作 10-30m）

## 返回值

| 字段名 | 类型 | 描述 |
|--------|------|------|
| exit_code | int | 退出码 (0 表示成功) |
| stdout | string | 标准输出 |
| stderr | string | 标准错误输出 |
| elapsed_ms | int | 执行耗时 (毫秒) |

## 使用示例

### 示例 1: 基本命令执行

```yaml
steps:
  - name: List files
    uses: exec/shell@v1
    with:
      command: ls
      args: ["-la", "/tmp"]
```

### 示例 2: 检查系统信息

```yaml
steps:
  - name: Get system info
    uses: exec/shell@v1
    with:
      command: uname
      args: ["-a"]
    id: sysinfo
  
  - name: Print system info
    uses: exec/shell@v1
    with:
      command: echo
      args: ["System: ${{ steps.sysinfo.outputs.stdout }}"]
```

### 示例 3: 带环境变量和工作目录

```yaml
steps:
  - name: Build application
    uses: exec/shell@v1
    with:
      command: npm
      args: ["run", "build"]
      env:
        NODE_ENV: production
        API_URL: https://api.example.com
      workdir: /app/frontend
      timeout: 10m
```

### 示例 4: 调用 Git 命令

```yaml
steps:
  - name: Clone repository
    uses: exec/shell@v1
    with:
      command: git
      args: ["clone", "https://github.com/user/repo.git", "/app/repo"]
      timeout: 5m
  
  - name: Get latest commit
    uses: exec/shell@v1
    with:
      command: git
      args: ["log", "-1", "--pretty=%H"]
      workdir: /app/repo
    id: commit
```

### 示例 5: 健康检查

```yaml
steps:
  - name: Check service port
    uses: exec/shell@v1
    with:
      command: curl
      args: ["-f", "http://localhost:8080/health"]
      timeout: 30s
    retry:
      max_attempts: 5
      initial_interval: 2s
```

### 示例 6: 使用 Secrets

```yaml
steps:
  - name: Deploy with credentials
    uses: exec/shell@v1
    with:
      command: ./deploy.sh
      env:
        DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
        API_TOKEN: ${{ secrets.API_TOKEN }}
      workdir: /app/scripts
```

## 常见错误

### 错误 1: 命令不存在

**现象:**
```
Error: executable file not found in $PATH
exit_code: 127
```

**原因:**
命令不在系统 PATH 中，或拼写错误。

**解决方法:**
1. 使用绝对路径:
   ```yaml
   command: /usr/bin/git
   ```
2. 检查命令是否已安装:
   ```bash
   which git
   ```
3. 安装缺失的命令

### 错误 2: 权限不足

**现象:**
```
Error: permission denied
exit_code: 126
```

**原因:**
Agent 用户无权限执行命令或访问文件。

**解决方法:**
1. 检查文件权限:
   ```bash
   ls -l /path/to/file
   ```
2. 添加执行权限:
   ```bash
   chmod +x /path/to/script
   ```
3. 检查目录权限:
   ```bash
   chmod 755 /path/to/directory
   ```

### 错误 3: 命令超时

**现象:**
```
Error: context deadline exceeded
```

**原因:**
命令执行时间超过 timeout 设置。

**解决方法:**
1. 增加超时时间:
   ```yaml
   timeout: 15m
   ```
2. 优化命令执行效率
3. 考虑使用后台执行（detach 模式）

### 错误 4: 工作目录不存在

**现象:**
```
Error: chdir /path: no such file or directory
```

**原因:**
指定的 workdir 目录不存在。

**解决方法:**
1. 创建目录:
   ```yaml
   - name: Create directory
     uses: exec/shell@v1
     with:
       command: mkdir
       args: ["-p", "/app/data"]
   
   - name: Run command
     uses: exec/shell@v1
     with:
       command: ./app
       workdir: /app/data
   ```

### 错误 5: 命令执行失败

**现象:**
```
exit_code: 1
stderr: "some error message"
```

**原因:**
命令执行失败（业务错误，非系统错误）。

**解决方法:**
1. 查看 stderr 输出分析具体错误
2. 使用 `continue-on-error: true` 忽略错误继续执行
3. 使用条件判断处理失败:
   ```yaml
   - name: Try command
     uses: exec/shell@v1
     with:
       command: /usr/bin/test-command
     id: test
     continue-on-error: true
   
   - name: Handle failure
     if: steps.test.outputs.exit_code != 0
     uses: exec/shell@v1
     with:
       command: echo
       args: ["Command failed with: ${{ steps.test.outputs.stderr }}"]
   ```

## 最佳实践

### 1. 使用绝对路径

避免依赖 PATH 环境变量:

```yaml
# ❌ 不推荐
with:
  command: git

# ✅ 推荐
with:
  command: /usr/bin/git
```

### 2. 设置合理超时

根据命令复杂度调整:

```yaml
# 简单命令
timeout: 30s

# 中等复杂
timeout: 5m  # 默认值

# 复杂操作（构建、部署）
timeout: 15-30m
```

### 3. 检查退出码

使用条件判断处理失败:

```yaml
- name: Try command
  uses: exec/shell@v1
  with:
    command: /usr/bin/test-command
  id: test

- name: Success path
  if: steps.test.outputs.exit_code == 0
  uses: exec/shell@v1
  with:
    command: echo
    args: ["Success"]

- name: Failure path
  if: steps.test.outputs.exit_code != 0
  uses: exec/shell@v1
  with:
    command: echo
    args: ["Failed"]
```

### 4. 使用 Secrets 存储敏感信息

不要在 YAML 中硬编码密码或 Token:

```yaml
# ❌ 不安全
env:
  API_TOKEN: "secret123"
  DB_PASSWORD: "password"

# ✅ 安全
env:
  API_TOKEN: ${{ secrets.API_TOKEN }}
  DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
```

### 5. 参数使用数组避免注入

使用 args 数组而非拼接命令字符串:

```yaml
# ❌ 不推荐（可能有注入风险）
command: sh
args: ["-c", "rm -rf " + user_input]

# ✅ 推荐（参数自动转义）
command: rm
args: ["-rf", user_input]
```

### 6. 合理使用工作目录

避免在命令中拼接路径:

```yaml
# ❌ 不推荐
command: sh
args: ["-c", "cd /app && npm run build"]

# ✅ 推荐
command: npm
args: ["run", "build"]
workdir: /app
```

### 7. 捕获和使用输出

使用 id 和输出引用链接步骤:

```yaml
- name: Get version
  uses: exec/shell@v1
  with:
    command: cat
    args: ["/app/VERSION"]
  id: version

- name: Deploy specific version
  uses: exec/shell@v1
  with:
    command: ./deploy.sh
    args: ["${{ steps.version.outputs.stdout }}"]
```

### 8. 使用重试处理临时错误

网络请求等可能临时失败的操作添加重试:

```yaml
- name: Download file
  uses: exec/shell@v1
  with:
    command: curl
    args: ["-O", "https://example.com/file.tar.gz"]
  retry:
    max_attempts: 3
    initial_interval: 5s
    backoff_coefficient: 2.0
```

## 安全注意事项

1. **命令注入防护**: 使用 args 数组传递参数，避免字符串拼接
2. **权限最小化**: Agent 用户使用最小权限运行
3. **敏感信息**: 使用 Secrets 管理密码、密钥、Token
4. **日志脱敏**: 避免在命令输出中打印敏感信息
5. **超时保护**: 设置合理超时避免资源耗尽

## 性能考虑

- **启动开销**: 每次执行创建新进程，约 1-5ms 开销
- **输出大小**: stdout/stderr 限制 10MB，超大输出考虑重定向到文件
- **并发安全**: 多个步骤可并发执行，互不干扰

## 相关节点

- [exec/script](script.md) - 执行脚本文件（支持多行脚本和多种解释器）
- [docker/exec](../docker/exec.md) - 执行 Docker 命令

## 参考文档

- [Story 3.2: Shell 命令执行节点](../../sprint-artifacts/3-2-shell-command-execution-node.md)
- [工作流语法参考](../../yaml-dsl-syntax-reference.md)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: 稳定
