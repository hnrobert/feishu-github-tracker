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

### 前置要求

- Docker 和 Docker Compose（推荐）
- 或者 Go 1.21+（本地运行）

### 使用 Docker Compose（推荐）

#### 使用预构建镜像

```bash
# 拉取最新镜像
docker pull ghcr.io/hnrobert/feishu-github-tracker:latest

# 或使用 docker-compose（会自动拉取镜像）
docker-compose up -d
```

#### 从源码构建

1. **克隆仓库**

   ```bash
   git clone https://github.com/hnrobert/feishu-github-tracker.git
   cd feishu-github-tracker
   ```

2. **配置文件**

   编辑 `configs/` 目录下的配置文件：

   - `server.yaml` - 服务器配置（端口、密钥等）
   - `feishu-bots.yaml` - 飞书机器人 Webhook URL
   - `repos.yaml` - 仓库和事件映射规则
   - `events.yaml` - 事件定义和模板
   - `templates.yaml` - 飞书消息卡片模板

3. **启动服务**

   ```bash
   docker-compose up -d
   ```

4. **查看日志**

   ```bash
   docker-compose logs -f
   ```

5. **配置 GitHub Webhook**

在 GitHub 仓库设置中添加 Webhook：

- Payload URL: `http://your-server-address:4594/webhook`
- Content type: `application/json`
- Secret: 与 `server.yaml` 中的 `secret` 保持一致
- 选择需要的事件类型

### 本地运行

1. **安装依赖**

   ```bash
   go mod download
   ```

2. **构建**

   ```bash
   make build
   ```

3. **运行**

   ```bash
   ./bin/feishu-github-tracker
   ```

   或者直接运行：

   ```bash
   go run ./cmd/feishu-github-tracker
   ```

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
  host: '0.0.0.0' # 监听地址
  port: 4594 # 监听端口
  secret: 'your_secret' # GitHub Webhook 密钥
  log_level: 'info' # 日志级别: debug, info, warn, error
  max_payload_size: 5MB # 最大请求体大小
  timeout: 15 # 请求超时时间（秒）
```

### feishu-bots.yaml

定义飞书机器人及其别名：

```yaml
feishu_bots:
  - alias: 'dev-team'
    url: 'https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxx'

  - alias: 'ops-team'
    url: 'https://open.feishu.cn/open-apis/bot/v2/hook/yyyyyyy'
```

### repos.yaml

配置仓库匹配规则和通知目标：

```yaml
repos:
  # 精确匹配特定仓库
  - pattern: 'CompPsyUnion/motion-vote-backend'
    events:
      push:
        branches:
          - main
          - develop
      pull_request:
        types:
          - opened
          - closed
      issues:
      release:
    notify_to:
      - ops-team
      - dev-team

  # 使用通配符匹配多个仓库
  - pattern: 'CompPsyUnion/experimental-*'
    events:
      all: # 使用预定义的事件集
    notify_to:
      - dev-team

  # 匹配个人所有仓库
  - pattern: 'hnrobert/*'
    events:
      custom: # 使用自定义事件集
    notify_to:
      - ops-team

  # 兜底规则：匹配所有仓库
  - pattern: '*'
    events:
      basic:
    notify_to:
      - org-notify
```

### events.yaml

定义事件模板和具体事件配置：

```yaml
event_sets:
  # 基础事件集
  basic:
    push:
    pull_request:
    issues:
    release:

  # 完整事件集
  all:
    # 包含所有 GitHub 支持的事件...

  # 自定义事件集
  custom:
    push:
      branches:
        - main
        - develop
    pull_request:
      types:
        - opened
        - closed
```

### templates.yaml

定义飞书消息卡片模板。支持为不同事件类型和状态定义多个模板变体：

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

### Docker 命令

```bash
# 拉取最新镜像
docker pull ghcr.io/hnrobert/feishu-github-tracker:latest

# 启动服务（使用预构建镜像）
docker-compose up -d

# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 从源码重新构建并启动
docker-compose build
docker-compose up -d
```

### 可用的镜像标签

从 GitHub Container Registry 拉取：

- `ghcr.io/hnrobert/feishu-github-tracker:latest` - 最新稳定版（main 分支）
- `ghcr.io/hnrobert/feishu-github-tracker:main` - main 分支最新构建
- `ghcr.io/hnrobert/feishu-github-tracker:develop` - develop 分支最新构建
- `ghcr.io/hnrobert/feishu-github-tracker:v1.0.0` - 特定版本（发布时）
- `ghcr.io/hnrobert/feishu-github-tracker:sha-xxxxxxx` - 特定 commit

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
