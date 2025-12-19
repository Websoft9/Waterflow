# YAML Schema Integration Guide

本文档介绍如何在 IDE 中集成 Waterflow YAML Schema，以获得自动补全和实时验证支持。

## VS Code

### 1. 安装 YAML 扩展

```bash
code --install-extension redhat.vscode-yaml
```

或在 VS Code 扩展市场搜索 "YAML" 并安装 Red Hat 的 YAML 扩展。

### 2. 配置 Schema

在项目根目录创建或编辑 `.vscode/settings.json`：

```json
{
  "yaml.schemas": {
    "./schema/workflow-schema.json": [
      "*.waterflow.yaml",
      ".waterflow/*.yaml"
    ]
  }
}
```

### 3. 使用

创建 `.waterflow.yaml` 文件即可自动获得：
- 字段自动补全
- 实时语法验证
- 字段文档 Hover 提示

## IntelliJ IDEA / GoLand

### 1. 配置 JSON Schema

1. 打开 Settings/Preferences (Ctrl+Alt+S / Cmd+,)
2. 导航到: **Languages & Frameworks → Schemas and DTDs → JSON Schema Mappings**
3. 点击 "+" 添加新映射

### 2. 配置映射

- **Schema file or URL:** 选择项目中的 `schema/workflow-schema.json`
- **Schema version:** JSON Schema version 7
- **File path pattern:** `*.waterflow.yaml`

### 3. 应用并保存

点击 "OK" 应用更改。

## 在线 Schema

生产环境可以使用在线 Schema URL：

```
https://waterflow.dev/schema/v1/workflow.json
```

VS Code 配置示例：

```json
{
  "yaml.schemas": {
    "https://waterflow.dev/schema/v1/workflow.json": "*.waterflow.yaml"
  }
}
```

## Schema 特性

### 支持的自动补全

- 顶层字段：`name`, `on`, `jobs`, `env`
- Job 配置：`runs-on`, `timeout-minutes`, `needs`, `steps`
- Step 配置：`uses`, `with`, `timeout-minutes`, `if`
- 触发器类型：`push`, `pull_request`, `schedule`, `webhook`

### 实时验证

- 必填字段检查
- 字段类型验证
- 格式验证（如 `uses` 必须匹配 `node@version` 格式）
- 数值范围验证（如 timeout 1-1440 分钟）

## 示例

创建 `example.waterflow.yaml`:

```yaml
name: Build and Test
on: push
jobs:
  build:
    runs-on: linux-amd64
    timeout-minutes: 30
    steps:
      - uses: checkout@v1
        with:
          repository: https://github.com/websoft9/waterflow
      - uses: run@v1
        with:
          command: go test ./...
```

IDE 会自动提示可用的字段和验证配置。

## 故障排除

### VS Code 未显示自动补全

1. 检查 YAML 扩展是否已安装
2. 确认 `.vscode/settings.json` 配置正确
3. 重新加载 VS Code 窗口 (Cmd+Shift+P → "Reload Window")

### Schema 未被识别

1. 检查文件扩展名是否匹配配置的模式
2. 查看 schema 文件路径是否正确
3. 尝试在文件顶部添加 schema 注释：
   ```yaml
   # yaml-language-server: $schema=./schema/workflow-schema.json
   ```

## 参考

- [JSON Schema](https://json-schema.org/)
- [VS Code YAML Extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)
- [Waterflow 文档](https://waterflow.dev/docs)
