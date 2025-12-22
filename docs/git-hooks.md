# Git Hooks

本项目已配置本地 Git hooks 来确保代码质量。

## 已启用的 Hooks

### 1. pre-commit（提交前检查）
在每次 `git commit` 之前自动运行：
- ✅ Go 代码格式检查 (`go fmt`)
- ✅ 静态分析 (`go vet`)
- ✅ Lint 检查 (`golangci-lint`，仅检查修改的文件)
- ✅ 快速测试（仅测试修改的包）

**跳过方式**（不推荐）：
```bash
git commit --no-verify
```

### 2. pre-push（推送前检查）
在每次 `git push` 之前自动运行：
- ✅ 完整测试套件 (`make test`)
- ✅ 编译检查 (`make build`)

**跳过方式**（不推荐）：
```bash
git push --no-verify
```

### 3. commit-msg（提交信息验证）
验证提交信息是否符合 [Conventional Commits](https://www.conventionalcommits.org/) 规范。

**格式要求**：
```
<type>[optional scope]: <description>

[optional body]

[optional footer]
```

**允许的类型**：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式调整（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建或辅助工具的变动
- `ci`: CI 配置文件和脚本的变动
- `build`: 影响构建系统或外部依赖的更改
- `perf`: 性能优化
- `revert`: 回滚之前的提交

**示例**：
```bash
feat(api): add user authentication endpoint
fix(parser): handle empty workflow files correctly
docs: update installation guide
refactor(dsl): reorganize matrix and node packages
test(matrix): add integration tests
ci: upgrade GitHub Actions to v4
```

## 安装到其他开发者环境

由于 `.git/hooks/` 不被 Git 跟踪，团队成员需要手动设置。可以：

### 方法 1: 复制脚本（已完成）
```bash
# 钩子已经在 .git/hooks/ 中创建
ls -la .git/hooks/
```

### 方法 2: 使用 Husky（可选，需要 Node.js）
```bash
npm install husky --save-dev
npx husky install
```

### 方法 3: 添加到项目根目录（推荐）
创建 `scripts/install-hooks.sh`:
```bash
#!/bin/bash
cp scripts/hooks/* .git/hooks/
chmod +x .git/hooks/*
```

## 常见问题

### Q: Hook 检查太慢怎么办？
A: 
- `pre-commit` 只检查修改的文件，应该很快
- 如果急需提交可以用 `--no-verify`，但推送前仍会完整检查

### Q: 如何临时禁用 hooks？
A:
```bash
# 临时禁用 pre-commit
git commit --no-verify

# 临时禁用 pre-push
git push --no-verify
```

### Q: Lint 失败如何自动修复？
A:
```bash
golangci-lint run --fix
```

### Q: 格式问题如何修复？
A:
```bash
go fmt ./...
```

## 效果

启用 hooks 后：
1. ❌ **提交前**阻止格式错误的代码
2. ❌ **推送前**阻止测试失败的代码
3. ✅ 减少 CI 失败次数
4. ✅ 提高代码质量
5. ✅ 节省 CI 运行时间

---

**注意**：Hooks 在本地运行，不影响 CI。CI 仍然是最终的质量关卡。
