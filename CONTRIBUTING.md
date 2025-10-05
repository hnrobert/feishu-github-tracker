# Contributing to Feishu GitHub Tracker

感谢您对本项目的关注！我们欢迎各种形式的贡献。

## 如何贡献

### 报告 Bug

如果您发现了 bug，请创建一个 Issue 并包含：

1. 清晰的标题和描述
2. 复现步骤
3. 预期行为和实际行为
4. 环境信息（Go 版本、操作系统等）
5. 相关的日志输出

### 提出新功能

如果您有新功能的想法：

1. 先创建一个 Issue 讨论该功能
2. 说明功能的用途和价值
3. 如果可能，提供实现思路

### 提交代码

1. **Fork 项目**

2. **创建特性分支**

   ```bash
   git checkout -b feature/amazing-feature
   ```

3. **进行修改**

   - 遵循现有的代码风格
   - 添加必要的测试
   - 更新相关文档

4. **提交更改**

   ```bash
   git add .
   git commit -m "Add amazing feature"
   ```

5. **推送到 Fork**

   ```bash
   git push origin feature/amazing-feature
   ```

6. **创建 Pull Request**

## 开发指南

### 环境准备

```bash
# 克隆项目
git clone https://github.com/hnrobert/feishu-github-tracker.git
cd feishu-github-tracker

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建
make build
```

### 代码规范

- 使用 `go fmt` 格式化代码
- 使用 `go vet` 检查代码
- 所有导出的函数和类型都需要注释
- 遵循 Go 标准项目布局

### 测试

- 为新功能添加单元测试
- 确保所有测试通过：`go test ./...`
- 测试覆盖率应保持或提高

### 提交信息规范

使用清晰的提交信息：

```text
<type>: <subject>

<body>

<footer>
```

类型（type）：

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

示例：

```text
feat: add support for GitHub Actions events

- Add handler for workflow_run events
- Update templates for Actions notifications
- Add tests for new event type

Closes #123
```

## Pull Request 检查清单

提交 PR 前请确认：

- [ ] 代码遵循项目规范
- [ ] 已添加必要的测试
- [ ] 所有测试通过
- [ ] 已更新相关文档
- [ ] 提交信息清晰明确
- [ ] 已解决所有合并冲突

## 行为准则

- 尊重所有贡献者
- 建设性地讨论
- 接受建设性批评
- 关注对项目最有利的事情

## 许可证

提交代码即表示您同意按照项目的 MIT 许可证发布您的贡献。

## 需要帮助？

如有任何问题，欢迎：

- 创建 Issue
- 发起 Discussion
- 联系维护者

再次感谢您的贡献！
