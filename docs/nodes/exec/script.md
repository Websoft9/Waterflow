# exec/script@v1

**分类**: exec (命令执行)  
**版本**: v1  
**状态**: stable

## 概述

exec/script 节点用于在 Agent 服务器上执行脚本文件或内联脚本。支持多种解释器（Bash、Python、Node.js、Ruby），适合执行复杂逻辑和多行脚本。

与 exec/shell 的区别：shell 执行单条命令，script 执行脚本文件或内联脚本内容。

## 使用场景

- **复杂脚本逻辑**: 多行脚本、循环、条件判断
- **多语言支持**: Bash、Python、Node.js、Ruby 脚本
- **无需文件**: 使用 script_content 直接嵌入脚本内容
- **批处理任务**: 数据处理、系统维护、批量操作

## 参数

| 参数名 | 类型 | 必需 | 默认值 | 描述 |
|--------|------|------|--------|------|
| script_path | string | ❌* | - | 脚本文件路径 |
| script_content | string | ❌* | - | 内联脚本内容 |
| interpreter | string | ❌ | bash | 解释器 (bash/sh/python3/node/ruby) |
| args | array | ❌ | [] | 脚本参数 |
| env | object | ❌ | {} | 环境变量 |
| workdir | string | ❌ | "" | 工作目录 |
| timeout | string | ❌ | 5m | 超时时间 |

*注: script_path 和 script_content 必须指定其中一个，不能同时指定。

### 参数详细说明

**script_path** (string, optional)
- 脚本文件路径（绝对路径或相对于 workdir）
- 文件必须存在且可读
- 与 script_content 互斥
- 示例: `"/app/scripts/deploy.sh"`, `"./backup.py"`

**script_content** (string, optional)
- 内联脚本内容
- 自动创建临时文件执行，执行后自动删除
- 与 script_path 互斥
- 示例: `"#!/bin/bash\necho 'Hello'"`

**interpreter** (string, optional, default: bash)
- 脚本解释器
- 支持: bash, sh, python3, python, node, ruby
- 解释器必须已安装在 Agent 服务器
- 示例: `"python3"`, `"node"`

**args** (array, optional)
- 传递给脚本的参数
- Bash: 通过 $1, $2... 访问
- Python: 通过 sys.argv[1], sys.argv[2]... 访问
- 示例: `["arg1", "arg2"]`

其他参数（env, workdir, timeout）与 exec/shell 相同。

## 返回值

| 字段名 | 类型 | 描述 |
|--------|------|------|
| exit_code | int | 退出码 |
| stdout | string | 标准输出 |
| stderr | string | 标准错误输出 |
| elapsed_ms | int | 执行耗时 (毫秒) |
| interpreter_used | string | 实际使用的解释器 |

## 使用示例

### 示例 1: 执行 Bash 脚本文件

```yaml
steps:
  - name: Run deployment script
    uses: exec/script@v1
    with:
      script_path: /app/scripts/deploy.sh
      interpreter: bash
      args: ["production", "v1.2.3"]
      env:
        DEPLOY_USER: admin
      workdir: /app
```

### 示例 2: 内联 Bash 脚本

```yaml
steps:
  - name: Backup database
    uses: exec/script@v1
    with:
      interpreter: bash
      script_content: |
        #!/bin/bash
        echo "Starting backup..."
        mysqldump -u root mydb > /backup/mydb_$(date +%Y%m%d).sql
        echo "Backup completed"
      env:
        MYSQL_PASSWORD: ${{ secrets.DB_PASSWORD }}
```

### 示例 3: 执行 Python 脚本

```yaml
steps:
  - name: Data processing
    uses: exec/script@v1
    with:
      interpreter: python3
      script_content: |
        import sys
        import json
        
        # Read input from args
        input_file = sys.argv[1]
        output_file = sys.argv[2]
        
        # Process data
        with open(input_file, 'r') as f:
            data = json.load(f)
        
        # Transform and save
        result = [item['name'] for item in data]
        with open(output_file, 'w') as f:
            json.dump(result, f)
        
        print(f"Processed {len(result)} items")
      args: ["/data/input.json", "/data/output.json"]
      timeout: 10m
```

### 示例 4: Node.js 脚本

```yaml
steps:
  - name: Run Node script
    uses: exec/script@v1
    with:
      script_path: /app/scripts/process.js
      interpreter: node
      workdir: /app
      env:
        NODE_ENV: production
```

### 示例 5: 带参数的复杂脚本

```yaml
steps:
  - name: System maintenance
    uses: exec/script@v1
    with:
      interpreter: bash
      script_content: |
        #!/bin/bash
        set -e
        
        ACTION=$1
        TARGET=$2
        
        case $ACTION in
          clean)
            echo "Cleaning $TARGET..."
            rm -rf "/tmp/$TARGET"/*
            ;;
          backup)
            echo "Backing up $TARGET..."
            tar -czf "/backup/$TARGET.tar.gz" "$TARGET"
            ;;
          *)
            echo "Unknown action: $ACTION"
            exit 1
            ;;
        esac
      args: ["clean", "cache"]
```

## 常见错误

### 错误 1: 脚本文件不存在

**现象:**
```
Error: script file not found: /path/to/script.sh
```

**解决方法:**
1. 检查文件路径是否正确
2. 使用绝对路径
3. 确认文件在 Agent 服务器上存在

### 错误 2: 解释器未安装

**现象:**
```
Error: interpreter not found: python3
```

**解决方法:**
1. 在 Agent 服务器上安装解释器:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install python3
   
   # CentOS/RHEL
   sudo yum install python3
   ```

### 错误 3: script_path 和 script_content 同时指定

**现象:**
```
Error: script_path and script_content are mutually exclusive
```

**解决方法:**
只指定其中一个参数。

### 错误 4: 脚本权限错误

**现象:**
```
Error: permission denied
```

**解决方法:**
```bash
chmod +x /path/to/script.sh
```

### 错误 5: 脚本执行失败

**现象:**
```
exit_code: 1
stderr: "script error message"
```

**解决方法:**
1. 检查脚本逻辑错误
2. 查看 stderr 输出
3. 添加 `set -e` 在脚本开头（Bash）使脚本在第一个错误处停止

## 最佳实践

### 1. 选择正确的解释器

```yaml
# Bash: 系统管理、文件操作
interpreter: bash

# Python: 数据处理、API 调用
interpreter: python3

# Node.js: JavaScript 逻辑
interpreter: node
```

### 2. 使用 script_content 避免文件依赖

```yaml
# ✅ 推荐: 自包含，无外部文件依赖
script_content: |
  #!/bin/bash
  echo "Embedded script"

# ❌ 需要确保文件存在
script_path: /app/scripts/deploy.sh
```

### 3. 脚本错误处理（Bash）

```yaml
script_content: |
  #!/bin/bash
  set -e  # 遇到错误立即退出
  set -u  # 使用未定义变量时报错
  set -o pipefail  # 管道中任何命令失败都返回失败
  
  echo "Starting task..."
  # 脚本逻辑
```

### 4. Python 脚本最佳实践

```yaml
script_content: |
  #!/usr/bin/env python3
  import sys
  import os
  
  def main():
      try:
          # 脚本逻辑
          print("Success")
          return 0
      except Exception as e:
          print(f"Error: {e}", file=sys.stderr)
          return 1
  
  if __name__ == "__main__":
      sys.exit(main())
```

### 5. 使用参数传递配置

```yaml
# ✅ 推荐: 通过参数传递
script_content: |
  #!/bin/bash
  ENV=$1
  VERSION=$2
  echo "Deploying $VERSION to $ENV"
args: ["production", "v1.2.3"]

# ❌ 硬编码配置
script_content: |
  #!/bin/bash
  ENV="production"  # 不灵活
```

### 6. 合理设置超时

```yaml
# 简单脚本
timeout: 1m

# 数据处理
timeout: 10m

# 长时间运行任务
timeout: 30m
```

## 支持的解释器

| 解释器 | 命令 | 用途 | 常见场景 |
|--------|------|------|----------|
| bash | /bin/bash | Bash 脚本 | 系统管理、部署脚本 |
| sh | /bin/sh | POSIX shell | 兼容性脚本 |
| python3 | /usr/bin/python3 | Python 3 脚本 | 数据处理、API 调用 |
| python | /usr/bin/python | Python 2/3 | 遗留脚本 |
| node | /usr/bin/node | Node.js | JavaScript 逻辑 |
| ruby | /usr/bin/ruby | Ruby 脚本 | Ruby 应用 |

## 相关节点

- [exec/shell](shell.md) - 执行单条 Shell 命令
- [docker/exec](../docker/exec.md) - 执行 Docker 命令

## 参考文档

- [Story 3.3: 脚本文件执行节点](../../sprint-artifacts/3-3-script-file-execution-node.md)

---

**文档版本**: v1  
**最后更新**: 2025-12-31  
**状态**: 稳定
