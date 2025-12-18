# cmd/server

Server 应用程序入口点。

## 职责

- 解析命令行参数
- 加载配置
- 初始化日志系统
- 创建和启动 HTTP Server
- 处理优雅关闭信号

## 使用

```bash
# 使用默认配置
./server

# 指定配置文件
./server --config /path/to/config.yaml

# 指定端口
./server --port 9090

# 指定日志级别
./server --log-level debug
```
