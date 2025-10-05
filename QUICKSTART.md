# Quick Start Example

这是一个 5 分钟快速开始指南，帮助您快速体验本项目。

## 步骤 1: 克隆项目

```bash
git clone https://github.com/hnrobert/feishu-github-tracker.git
cd feishu-github-tracker
```

## 步骤 2: 获取飞书机器人 Webhook URL

1. 在飞书群组中添加自定义机器人
2. 复制生成的 Webhook URL（格式如：`https://open.feishu.cn/open-apis/bot/v2/hook/xxx...`）

## 步骤 3: 配置

编辑 `configs/feishu-bots.yaml`：

```yaml
feishu_bots:
  - alias: 'my-bot'
    url: '粘贴您的飞书 Webhook URL'
```

编辑 `configs/repos.yaml`，将您的 GitHub 仓库添加进去：

```yaml
repos:
  - pattern: 'your-username/your-repo' # 改成您的仓库
    events:
      push:
      pull_request:
      issues:
    notify_to:
      - my-bot
```

生成一个 GitHub Webhook Secret：

```bash
openssl rand -hex 32
```

编辑 `configs/server.yaml`，将生成的密钥填入：

```yaml
server:
  host: '0.0.0.0'
  port: 4594
  secret: '粘贴刚才生成的密钥'
  log_level: 'info'
  max_payload_size: 5MB
  timeout: 15
```

## 步骤 4: 启动服务

### 使用 Docker Compose（推荐）

```bash
docker-compose up -d
docker-compose logs -f
```

### 或本地运行

```bash
go run ./cmd/feishu-github-tracker
```

服务将在 `http://localhost:4594` 启动。

## 步骤 5: 配置 GitHub Webhook

1. 进入您的 GitHub 仓库
2. 点击 **Settings** > **Webhooks** > **Add webhook**
3. 填写：
   - **Payload URL**: `http://your-server-address:4594/webhook`
   - **Content type**: `application/json`
   - **Secret**: 使用步骤 3 中生成的密钥
   - **Which events**: 选择 "Send me everything" 或选择特定事件
4. 点击 **Add webhook**

## 步骤 6: 测试

### 方法 1: 在 GitHub 上触发事件

- 在仓库中创建一个 Issue
- Push 一些代码
- 创建一个 Pull Request

查看飞书群组，应该会收到通知！

### 方法 2: 使用 GitHub Webhook 重发功能

1. 在 GitHub Webhook 设置页面
2. 点击最近的一个事件
3. 点击 **Redeliver** 按钮
4. 查看飞书群组和服务日志

### 查看日志

```bash
# Docker 部署
docker-compose logs -f

# 本地运行
tail -f log/feishu-github-tracker-*.log
```

## 常见问题

### 1. 没有收到通知？

检查：

- 服务是否正常运行
- GitHub Webhook 是否配置正确
- 在 GitHub Webhook 页面查看 "Recent Deliveries"
- 查看服务日志是否有错误

### 2. 签名验证失败？

确保 `server.yaml` 中的 `secret` 与 GitHub Webhook 中的 Secret 完全一致。

### 3. 端口被占用？

编辑 `docker-compose.yml` 或 `server.yaml` 修改端口。

## 下一步

- 阅读 [CONFIGURATION.md](CONFIGURATION.md) 了解详细配置选项
- 阅读 [DEPLOYMENT.md](DEPLOYMENT.md) 了解生产环境部署
- 自定义消息模板（编辑 `configs/templates.yaml`）
- 添加更多仓库和机器人配置

## 示例场景

### 场景 1: 监控多个仓库的 PR

```yaml
# repos.yaml
repos:
  - pattern: 'myorg/*'
    events:
      pull_request:
        types:
          - opened
          - closed
    notify_to:
      - dev-team
```

### 场景 2: 不同分支通知不同群组

```yaml
# repos.yaml
repos:
  - pattern: 'myorg/production-app'
    events:
      push:
        branches:
          - main
    notify_to:
      - ops-team

  - pattern: 'myorg/production-app'
    events:
      push:
        branches:
          - develop
    notify_to:
      - dev-team
```

注意：由于匹配顺序，这个场景需要调整配置逻辑，或者使用不同的仓库 pattern。更好的方式是在一个配置中处理所有事件，然后通过模板来区分。

### 场景 3: 只关注特定类型的 Issue

```yaml
# repos.yaml
repos:
  - pattern: 'myorg/support-repo'
    events:
      issues:
        types:
          - opened
          - labeled
    notify_to:
      - support-team
```

祝您使用愉快！如有问题，欢迎提 Issue。
