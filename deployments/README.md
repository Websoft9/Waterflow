# deployments

部署配置文件目录。

## 文件列表

- **docker-compose.yaml** - Docker Compose 编排配置
  - PostgreSQL 数据库
  - Temporal Server 和 Temporal UI
  - Waterflow Server

## 环境

- **development** - 本地开发环境
- **staging** - 预发布环境 (未来)
- **production** - 生产环境 (未来)

## 使用指南

详细的部署说明请参考：[../docs/deployment.md](../docs/deployment.md)

快速启动：

```bash
cd deployments
docker-compose up -d
```
