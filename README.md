# Feishu GitHub Tracker

[![CI/CD](https://github.com/hnrobert/feishu-github-tracker/actions/workflows/ci.yml/badge.svg)](https://github.com/hnrobert/feishu-github-tracker/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hnrobert/feishu-github-tracker)](go.mod)
[![License](https://img.shields.io/github/license/hnrobert/feishu-github-tracker)](LICENSE)

一个用于接收 GitHub Webhook 并转发到飞书机器人的中间件服务。支持灵活的配置、事件过滤和自定义消息模板。

## ✨ 特性

- 🔄 **自动转发**：接收 GitHub Webhook 事件并转发到飞书机器人
- 🎯 **灵活匹配**：支持通配符模式匹配仓库和分支
- 🎨 **自定义模板**：支持为不同事件类型定制飞书消息卡片
- 🔐 **安全验证**：支持 GitHub Webhook 签名验证
- 📊 **完整日志**：详细的事件处理日志，方便问题排查
- 🐳 **容器化部署**：提供 Docker 和 Docker Compose 支持
- ⚡ **高性能**：使用 Go 编写，轻量高效

## 📋 支持的 GitHub 事件

支持所有主要的 GitHub Webhook 事件，包括但不限于：

- `push` - 代码推送
- `pull_request` - Pull Request 相关
- `issues` - Issue 相关
- `release` - 发布相关
- `discussion` - 讨论相关
- `star`, `fork`, `watch` - 仓库关注相关
- 更多事件详见 [configs/events.yaml](configs/events.yaml)

## 🚀 快速开始

参考 [QUICKSTART.md](./QUICKSTART.md) 了解如何快速部署和测试。

## 📁 项目结构

```text
feishu-github-tracker/
├── cmd/
│   └── feishu-github-tracker/          # 主程序入口
│       └── main.go
├── internal/             # 内部包
│   ├── config/          # 配置加载
│   ├── handler/         # Webhook 处理器
│   ├── matcher/         # 仓库和事件匹配
│   ├── notifier/        # 飞书通知发送
│   └── template/        # 模板处理
├── pkg/
│   └── logger/          # 日志模块
├── configs/             # 配置文件目录
│   ├── server.yaml
│   ├── repos.yaml
│   ├── events.yaml
│   ├── feishu-bots.yaml
│   └── templates.yaml
├── log/                 # 日志文件目录
├── Dockerfile           # Docker 镜像构建
├── docker-compose.yml   # Docker Compose 配置
├── Makefile            # 构建脚本
└── README.md
```

## ⚙️ 配置说明

### server.yaml

服务器基础配置：

```yaml
server:
  host: '0.0.0.0' # Webhook监听主机
  port: 4594 # Webhook监听端口
  secret: 'your_secret' # 用于验证GitHub X-Hub-Signature的密钥
  log_level: 'info' # 可选: debug, info, warn, error
  max_payload_size: 5MB # 限制单次Webhook body大小
  timeout: 15 # 单次请求处理超时 (秒)

# 允许的来源（用于白名单过滤，可选）
allowed_sources:
  - 'github.com'
  - 'api.github.com'
  - 'your-github-enterprise-domain.com'
```

### feishu-bots.yaml

定义飞书机器人及其别名：

```yaml
feishu_bots:
  - alias: 'dev-team' # 可以在 repos.yaml 中通过该别名引用这个链接
    url: 'https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxx'

  - alias: 'ops-team'
    url: 'https://open.feishu.cn/open-apis/bot/v2/hook/yyyyyyy'

  - alias: 'org-notify'
    url: 'https://open.feishu.cn/open-apis/bot/v2/hook/zzzzzzz'
```

### repos.yaml

配置仓库匹配规则和通知目标：

```yaml
repos:
  # 示例：针对特定项目定义更详细监听
  - pattern: 'CompPsyUnion/motion-vote-backend'
    events:
      push: # 直接引用 events.yaml 中的事件
        branches: # 可以进一步细化，覆盖 events.yaml 中的默认配置
          - main
          - develop
      pull_request: # 同理
        types:
          - opened
          - closed
          - reopened
      issues: # 如果不细化，直接监听所有 types
      release:
    notify_to:
      - ops-team # 引用 feishu-bots.yaml 的 alias. 引号可加可不加
      - 'https://open.feishu.cn/open-apis/bot/v2/hook/zzzzzzz' # 这里是 dev-team, 但直接使用完整 URL 也可以。如有冲突 alias 优先

  # 示例：匹配实验性项目（使用 glob 模式）
  - pattern: 'CompPsyUnion/experimental-*'
    events:
      all: # 直接应用 event_sets: 中定义的的模板。如果有命名重合，优先使用自定义模板
    notify_to:
      - dev-team # 引用 feishu-bots.yaml 的 alias

  # 示例：匹配所有个人项目
  - pattern: 'hnrobert/*'
    events:
      custom: # 直接应用 event_sets: 中定义的的模板
    notify_to:
      - ops-team # 引用 feishu-bots.yaml 的 alias

  # 示例：匹配所有仓库（放在最后，作为兜底配置，已经被匹配过的仓库会被拦截，不会用到这里）
  - pattern: '*'
    events:
      basic: # 应用 events.yaml 内 event_sets: 中定义的的模板。可以理解将 basic 里的事件展开添加到该仓库监听
      project: # 也可以同时叠加更多事件。注意后添加的会覆盖先添加的的同类事件配置
    notify_to:
      - org-notify # 引用 feishu-bots.yaml 的 alias
```

### events.yaml

定义事件模板和具体事件配置：

```yaml
event_sets:
  # 基础事件集
  basic:
    push:
    pull_request:
    pull_request_review:
    pull_request_review_comment:
    issues:
    issue_comment:
    discussion:
    discussion_comment:
    release:
    package:

  # 可以自定义事件集
  custom:
    push:
      branches:
        - main
        - develop
    pull_request:
      types:
        - opened
        - closed

  # 完整事件集
  all:
    # 包含所有 GitHub 支持的事件...
```

具体参考 [./configs/events.yaml](./configs/events.yaml) 中的详细内容

### templates.yaml

定义飞书消息卡片模板。支持为不同事件类型和状态定义多个模板变体。当前已经包括了所有你需要的常用事件的模板，你可以根据自己的需要进行修改和扩展。

这里的模板是基于飞书的消息卡片格式设计的，详情请参考 [飞书开放平台文档](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message/create)。

```yaml
templates:
  push:
    payloads:
      - tags: [push, default]
        payload:
          msg_type: interactive
          card:
            # 飞书卡片配置...

      - tags: [push, force]
        payload:
          # Force push 的特殊模板...
```

模板支持占位符替换，如：

- `{{repo_name}}` - 仓库名称
- `{{sender_name}}` - 触发者
- `{{pr_title}}` - PR 标题
- `{{issue_number}}` - Issue 编号
- 更多占位符详见代码中的 `prepareTemplateData` 函数

## 🔧 高级功能

### 事件过滤

支持多级事件过滤：

1. **仓库级别**：使用 glob 模式匹配仓库
2. **事件类型级别**：选择需要的事件类型
3. **分支级别**：为 push/PR 指定分支规则
4. **动作级别**：为事件指定具体的 action（如 opened, closed）

### 模板选择

程序会根据事件的实际情况自动选择最合适的模板：

- Force push 会使用特殊的 force push 模板
- 已合并的 PR 关闭和未合并的 PR 关闭使用不同模板
- Issue 根据标签（bug/feature/task）选择不同样式

### 通知目标

`notify_to` 支持两种方式：

1. **别名引用**：引用 `feishu-bots.yaml` 中定义的 alias
2. **直接 URL**：直接提供完整的飞书 Webhook URL

## 📊 监控和维护

### 健康检查

访问 `/health` 端点检查服务状态：

```bash
curl http://localhost:4594/health
```

### 日志

日志同时输出到控制台和文件：

- 文件位置：`log/feishu-github-tracker-YYYY-MM-DD.log`
- 每天自动创建新的日志文件
- 日志级别可在 `server.yaml` 中配置

## 🛠️ 开发

### 构建

```bash
# 本地构建
make build

# Docker 构建
make docker-build
```

### 测试

```bash
make test
```

### 代码格式化

```bash
make fmt
```

## 📝 环境变量

- `CONFIG_DIR` - 配置文件目录路径（默认：`./config`）
- `LOG_DIR` - 日志文件目录路径（默认：`./log`）
- `TZ` - 时区设置（默认：`Asia/Shanghai`）

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [gobwas/glob](https://github.com/gobwas/glob) - Glob 模式匹配
- [go-yaml/yaml](https://github.com/go-yaml/yaml) - YAML 解析
- [Feishu Open Platform](https://open.feishu.cn/) - 飞书开放平台

## 📮 联系方式

- 作者: hnrobert
- 项目地址: <https://github.com/hnrobert/feishu-github-tracker>
- Issues: <https://github.com/hnrobert/feishu-github-tracker/issues>
