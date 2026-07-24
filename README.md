# Feishu GitHub Tracker

[![CI/CD](https://github.com/hnrobert/feishu-github-tracker/actions/workflows/ci.yml/badge.svg)](https://github.com/hnrobert/feishu-github-tracker/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hnrobert/feishu-github-tracker)](go.mod)
[![License](https://img.shields.io/github/license/hnrobert/feishu-github-tracker)](LICENSE)

一个用于接收 GitHub Webhook 并转发到飞书机器人的中间件服务。支持灵活的配置、事件过滤和自定义消息模板。

![logo](./assets/images/banner.png)

## 写在前面

### 为什么有这个项目

首先，众所周知，飞书在目前没有一个官方的 GitHub 集成（至少在国内是这样，也许之前有，后来因为种种原因总之是没了）。虽然可以通过 GitMaya 等第三方服务实现，但不是不完善（比如 GitMaya 2024 年初还在更新的，结果现在是不可用状态），要不就是操作比较复杂（胡言乱语无法理解）或者通过 `workflow` 实现（太麻烦），要不就是过于简单，无法满足实际需求。

所以，我还是决定直接搓一个给大伙用了。我这边的主要目标是：

- 简单易用：配置简单，Docker Compose 开箱即用，基于 GitHub 的 Webhook 实现
- 可视化管理：内置 Web 管理面板，浏览器里即可增删改仓库规则、飞书机器人、事件、模板等，无需手改 YAML
- 灵活可定制：支持多种事件过滤和自定义消息模板，只要替换现有的 `configs/templates.jsonc` 就可以满足大部分的模版定制需求。
- 高效稳定：使用 Go 语言编写，性能优越
- 安全可靠：支持签名验证，防止伪造请求
- 开源免费：MIT 许可证，欢迎自开分支或者贡献回来（plz）

### TODO

- [x] 已添加基于 `html/template` 的 `web` 管理面板（白色 + `#4EACF8`/`#071C37` 主题，中英双语），可在浏览器中查看日志 / 修改仓库规则、飞书机器人、服务设置、事件与消息模板。详见 [docs/quickstart.md](docs/quickstart.md#5-web-管理面板可选)。
- [ ] 计划添加更多的事件模板（目前已经包含了大部分常用事件的模板，后续会根据反馈继续完善）

## 支持的 GitHub 事件

目前支持所有的 GitHub Webhook 事件

- 详见 [configs/events.yaml](configs/events.yaml)
- 对应的处理方法以及文档详见 [internal/handler/](internal/handler/)
- 默认提供的消息模板详见 [configs/templates.jsonc](configs/templates.jsonc)
- 也可以自定义模板，使用我们 `handler` 提供的的 `占位符变量` ([详见文档](internal/handler/README.md)) 以及 `template` 提供的 `模板引擎的语法` `过滤器` `条件块` 等功能 ([详见文档](internal/template/README.md)) 对发出消息的格式做相应的修改

### Webhook 设置提醒

当你在 GitHub 上添加 Webhook 时（无论是仓库级别还是组织级别），GitHub 会发送一个 **ping 事件**来测试 Webhook 配置。本服务会：

1. **自动识别 ping 事件**：无需在 `repos.yaml` 中特别配置
2. **智能匹配通知目标**：
   - 对于组织级 webhook：自动发送到配置了该组织所有仓库的飞书 bot, 即仅 `org-name/*` 模式匹配的仓库
   - 对于仓库级 webhook：自动发送到配置了该仓库的飞书 bot
3. **发送成功通知**：向飞书发送一条友好的 Webhook 设置成功消息，包含：
   - GitHub 禅语（zen message）
   - Hook ID 和类型
   - 仓库或组织信息

这样你就能立即确认 Webhook 已正确配置并能正常工作。

### 消息演示

<details open>
<summary>Misc</summary>

<img width="675" height="500" alt="image" src="https://github.com/user-attachments/assets/5f47b742-f004-4162-9ae6-9872554dc784" />

<img width="675" height="783" alt="image" src="https://github.com/user-attachments/assets/7f5a3b31-ecc5-4403-9bc5-96baa433d4bf" />

</details>

<details>
<summary>支持双语 / Bilingual support（所有卡片都有对应）</summary>

- 支持中英文双语快速切换
- 所有事件卡片均有对应的中英文版本
- 模板可通过 `configs/templates.cn.jsonc` 和 `configs/templates.en.jsonc` 自定义

<img width="675" height="521" alt="image" src="https://github.com/user-attachments/assets/21143037-9132-42c9-b3b4-3b0d6075ede9" />

</details>

<details>
<summary>Workflow 通知 / Workflow notifications</summary>

<img width="675" height="737" alt="image" src="https://github.com/user-attachments/assets/583e6e94-86b2-4597-943e-17f879b089cc" />

</details>

<details>
<summary>Release 通知 / Release notifications</summary>

<img width="675" height="794" alt="image" src="https://github.com/user-attachments/assets/0fb7870e-ae87-4785-9ae9-595f07fe4040" />

</details>

<details>
<summary>Issue 相关 / Issue related</summary>

<img width="675" height="783" alt="image" src="https://github.com/user-attachments/assets/476027ea-8ac9-49f8-a72e-ec377e4f8786" />

</details>

<details>
<summary>PR 相关 / PR related</summary>

<img width="675" height="786" alt="image" src="https://github.com/user-attachments/assets/0078b45c-52a5-45e2-90f1-7d09017f149b" />

</details>

<details>
<summary>其他事件 / Other events（Star、Watch 等）</summary>

<img width="675" height="380" alt="image" src="https://github.com/user-attachments/assets/7c720de2-ffa9-4bb1-9cdb-156eef435c90" />

</details>

## 快速开始

参考 [docs/quickstart.md](./docs/quickstart.md) 了解如何快速自建服务器部署和测试；从源码构建见 [docs/build-from-source.md](./docs/build-from-source.md)。

### 体验/使用现成服务（适合自己部署成本/难度较高的用户）

当然我们也有部署好的服务（<https://feishu-github-tracker.hnrobert.space/webhook>，直接 call 是没效果的，需要在我的服务器上更新配置才能转发到你那里 😈）可供大家使用或者尝试（下方邮件联系我获取试用和少量技术支持）。由于当前的服务是部署在个人服务器上需要运维成本，且配置当前依然需要人力维护，如果想要长期使用，需联系我付费 25¥ (含**永久使用权** + **1 年的不限次数配置更新&技术支持**，1 年后如需要更新配置或者技术支持，也请续费 10¥/年~~辛苦费~~)。

当然为了鼓励大家参与开源，**如果你能提一个有效的 PR（修复 bug 或者添加功能（请先提 issue））且最终被合并，或者提出一个详细的包含修复方法的 bug report issue**，试用款可以全额退回，后续也不需要支出任何额外费用。

如有想法可以邮件联系我：[hnrobert@qq.com](mailto:hnrobert@qq.com)

## 项目结构

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
│   └── templates.jsonc
├── logs/                 # 日志文件目录
├── Dockerfile           # Docker 镜像构建
├── docker-compose.yml   # Docker Compose 配置
├── Makefile            # 构建脚本
└── README.md
```

## 配置说明

### server.yaml

服务器基础配置：

```yaml
server:
  host: '0.0.0.0' # Webhook监听主机
  port: 4594 # Webhook监听端口
  # secret: 'your_secret' # 可选：全局 Webhook 密钥（fallback），用于验证 GitHub X-Hub-Signature。留空/注释掉则不启用全局校验（可改用每条 repos 匹配各自的 secret）。若某条 repos 匹配单独配置了 secret，则该规则优先使用自己的 secret，否则回退到这里
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

  - alias: 'org-cn-notify'
    url: 'https://open.feishu.cn/open-apis/bot/v2/hook/aaaaaaa'
    template: 'cn' # 可选：指定使用的消息模板，默认为 'default'
```

**多模板支持**：

从 v1.1.0 开始，支持为不同的飞书 bot 配置不同的消息模板。这在以下场景特别有用：

- 中英文双语团队，需要发送不同语言的通知
- 不同团队需要不同格式的消息
- 测试环境和生产环境使用不同的消息格式

配置方法：

1. 在 `feishu-bots.yaml` 中为 bot 指定 `template` 字段（可选）
2. 在 `configs/` 目录下创建对应的模板文件，命名格式为 `templates.<name>.jsonc`

例如：

- `templates.jsonc` - 默认模板（必需）
- `templates.cn.jsonc` - 中文模板
- `templates.en.jsonc` - 英文模板
- `templates.simple.jsonc` - 简化模板

如果某个 bot 没有指定 `template` 字段，或指定的模板文件不存在，将自动使用 `templates.jsonc` 作为默认模板。

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
        branches:
          - main
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

### templates.jsonc

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

模板支持 `占位符替换` ，如：

- `{{repo_name}}` - 仓库名称
- `{{sender_name}}` - 触发者
- `{{pr_title}}` - PR 标题
- `{{issue_number}}` - Issue 编号

以及一些 `tag` 的判断，如：

- `[push, force]` - 仅当是 force push 时使用该模板
- `[pull_request, closed, merged]` - 仅当 PR 被合并时

更多 `占位符` 和 `tag` 相关说明详见我们 `handler` 提供的的 `占位符变量` ([详见文档](internal/handler/README.md))

## 高级功能

### 事件过滤

支持多级事件过滤：

1. **仓库级别**：使用 glob 模式匹配仓库
2. **事件类型级别**：选择需要的事件类型
3. **分支级别**：为 push/PR 指定分支规则
4. **动作级别**：为事件指定具体的 action（如 opened, closed）

### 多规则匹配

默认情况下，仓库事件只使用 `repos.yaml` 中第一条匹配的规则，适合使用精确规则覆盖通配符、将 `*` 作为兜底规则的配置方式。

若同一个仓库需要按事件发送到不同飞书机器人，可在 `server.yaml` 的 `server:` 下开启：

```yaml
match_all_rules: true
```

开启后会按 `repos.yaml` 的顺序评估所有匹配规则：不订阅当前事件的规则会跳过，订阅该事件的规则会使用自己的 `notify_to` 发送通知。相同目标在同一次 webhook 内只会收到一条消息，发送仍按顺序执行。例如：

```yaml
repos:
  - pattern: "acme/widget"
    events:
      release:
    notify_to: [release-bot]
  - pattern: "acme/widget"
    events:
      all:
    notify_to: [activity-bot]
  - pattern: "acme/widget"
    events:
      reviewer:
    notify_to: [review-bot]
```

在这个例子中，`issue_comment` 会通知 `activity-bot` 和 `review-bot`，而 `release` 会通知 `release-bot` 和 `activity-bot`。组织级 Webhook 保持原有行为。

### 模板选择

程序会根据事件的实际情况自动选择最合适的模板：

- Force push 会使用特殊的 force push 模板
- 已合并的 PR 关闭和未合并的 PR 关闭使用不同模板
- Issue 根据标签（bug/feature/task）选择不同样式

### 通知目标

`notify_to` 支持两种方式：

1. **别名引用**：引用 `feishu-bots.yaml` 中定义的 alias
2. **直接 URL**：直接提供完整的飞书 Webhook URL

### Webhook 密钥（可选 / per-rule secret）

默认情况下，所有 Webhook 用 `server.yaml` 中的全局 `server.secret` 校验签名。如果不同仓库/组织需要各自独立的密钥，可以在 `repos.yaml` 的某条匹配上单独配置 `secret`：

```yaml
repos:
  - pattern: 'acme/widget'
    events:
      push:
    notify_to:
      - dev-team
    secret: 'this-repo-only-secret' # 可选：仅该校验该仓库 Webhook 的签名
```

校验规则：匹配到该仓库/组织的 Webhook，会尝试用「该规则的 `secret`」与「全局 `server.secret`」两者校验，任一通过即可；若两者都为空则跳过校验。这样不同 GitHub 端的 Webhook 可以使用各自独立的密钥。

## 监控和维护

### 健康检查

访问 `/health` 端点检查服务状态：

```bash
curl http://localhost:4594/health
```

### 日志

日志同时输出到控制台和文件：

- 文件位置：`logs/feishu-github-tracker-YYYY-MM-DD.log`
- 每天自动创建新的日志文件
- 日志级别可在 `server.yaml` 中配置

## 开发

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

## 环境变量

- `CONFIG_DIR` - 配置文件目录路径（默认：`./configs`）
- `LOG_DIR` - 日志文件目录路径（默认：`./logs`）
- `TZ` - 时区设置（默认：`Asia/Shanghai`）

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 致谢

- [gobwas/glob](https://github.com/gobwas/glob) - Glob 模式匹配
- [go-yaml/yaml](https://github.com/go-yaml/yaml) - YAML 解析
- [Feishu Open Platform](https://open.feishu.cn/) - 飞书开放平台

## 联系方式

- 作者: hnrobert
- 项目地址: <https://github.com/hnrobert/feishu-github-tracker>
- Issues: <https://github.com/hnrobert/feishu-github-tracker/issues>
