# internal

项目私有代码,不可被外部项目引用。

## 子包

- **server** - HTTP Server 实现

## 设计原则

- 包含特定于 Waterflow 的业务逻辑
- 可以依赖 pkg/ 中的公共库
- 不能被外部项目 import
