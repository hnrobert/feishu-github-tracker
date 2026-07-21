# 快速开始

镜像包含不可变的默认配置，并默认启用配置热重载。使用 Compose 启动时，程序会把缺失的默认配置文件初始化到本地 `./configs`（已有文件不会被覆盖）；`./configs` 和 `./logs` 会挂载到容器里，方便实时编辑与查看日志。

> 想从源码构建 / 改代码部署？请看 [从源码构建](build-from-source.md)。

## 1. 下载 docker-compose.yml

> 前置：已安装 `wget`（或手动下载）、`Docker` 与 `Docker Compose`。

```bash
mkdir feishu-github-tracker && cd feishu-github-tracker
# 下载仓库 main 分支的 docker-compose.yml
wget -O docker-compose.yml https://raw.githubusercontent.com/hnrobert/feishu-github-tracker/main/docker-compose.yml
```

## 2. 启动

<details open>
<summary>docker compose（现代 Compose）</summary>

```bash
docker compose up -d
docker compose logs -f
```

</details>

<details>
<summary>docker-compose（早期 Compose）</summary>

```bash
docker-compose up -d
docker-compose logs -f
```

</details>

说明：

- 首次启动会把镜像内的默认配置复制到本地配置目录（已有配置不会被覆盖）：
  - 本地 `./configs` ↔ 容器 `/app/configs`
  - 本地 `./logs` ↔ 容器 `/app/logs`
- 镜像默认启用热重载：每次收到 webhook 时会重新加载配置，所以改完 `./configs/` 无需重启即生效。

## 3. 访问健康检查

服务默认监听 `4594` 端口：

```md
http://localhost:4594/health
```

## 4. 修改配置

编辑 `./configs/` 下的配置文件，参考 [../README.md](../README.md) 或 [../configs](../configs/) 下示例文件的注释。最常需要改的有：

- [../configs/server.yaml](../configs/server.yaml)：监听地址、端口、`secret`（测试可不设）
- [../configs/feishu-bots.yaml](../configs/feishu-bots.yaml)：飞书机器人的 Webhook URL 与别名
  - 不清楚「飞书机器人 / Webhook URL」是什么？参考 [飞书文档](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot)，在群里建一个机器人并复制其 Webhook URL
- [../configs/repos.yaml](../configs/repos.yaml)：要监听的 GitHub 仓库、事件及通知对象
- [../configs/templates.jsonc](../configs/templates.jsonc)：默认消息模板（可选：创建 `templates.<名称>.jsonc` 自定义模板）

修改后保存，程序会在下一次收到 GitHub Webhook 时自动热重载。

## 5. Web 管理面板（可选）

除了手改 YAML，还内置一个 Web 管理面板，可在浏览器里增删改：仓库规则、飞书机器人、服务设置、事件配置、消息模板。

- 面板地址：`http://localhost:4594/`（与 webhook 同端口；`/webhook`、`/health` 仍照常工作）
- 默认账号：用户名 `admin` / 密码 `admin`（默认配置已带；老版本升级且未配置面板账号时，也自动用 `admin`/`admin`）
  - 用户名：可在 [../configs/server.yaml](../configs/server.yaml) 的 `panel.username`、环境变量 `PANEL_USERNAME`，或面板「服务设置」页修改
  - 密码（优先级从高到低）：
    - 环境变量（推荐）：`PANEL_PASSWORD=你的密码`
    - `panel.password`（明文）：**存在则优先使用**；启动 / reload 时会自动转为 `password_hash`（覆盖原 hash）、删除该明文行并补回 `# password: "admin"` 注释
    - `panel.password_hash`（bcrypt；可用 `htpasswd -bnBC 10 "" 你的密码 | tr -d ':\n' | sed 's/^\$2y/\$2a/'` 生成）
  - 修改密码需先填「当前密码」校验通过后才生效；保存后下次登录即用新账号，无需重启
- 面板内修改保存后会自动 reload 生效（无需等下一次 webhook 或重启）；手动编辑 `./configs/` 则需以 `--reload` 启动或重启进程。端口 / 密钥的改动仍需重启
- 注意：在「消息模板」页保存 `templates.*.jsonc` 会移除文件中的 `//` 注释并按字母重排键（功能不变）

## 6. 多模板配置（可选）

为不同飞书 bot 配置不同消息模板（如中英文双语）：

```yaml
feishu_bots:
  - alias: 'team-cn'
    url: 'https://open.feishu.cn/open-apis/bot/v2/hook/cn-webhook'
    template: 'cn' # 不设置则默认用 templates.jsonc
```

也可基于现有模板创建 `templates.<名称>.jsonc`，再在 `feishu-bots.yaml` 中引用。

## 7. 添加 GitHub Webhook

- 进入要监听的 GitHub 仓库 → `Settings` → `Webhooks` → `Add webhook`
- Payload URL：你的服务器地址，如 `http://your-domain-or-ip:4594/webhook`
- Content type：选什么都支持
- Secret：填你在 `server.yaml` 配置的 `secret`（如果配了）
- 事件类型：可选 `Let me select individual events` 勾选需要的事件，并在 [../configs/repos.yaml](../configs/repos.yaml) 对应仓库里用 `all:` 等做更细控制；详见 [../configs/events.yaml](../configs/events.yaml)
- 点击 `Add webhook`
- ✅ 配置无误的话，几秒后飞书群会收到一条「GitHub Webhook 添加成功」通知（GitHub 发送的 ping 事件），说明 Webhook 已生效

## 8. 简要调试

没收到通知？检查：

- GitHub Webhook 配置（Payload URL、Secret、事件类型）
- `docker compose logs -f` 里的错误日志；若完全没日志，确认 GitHub 是否真的发请求过来了
- 改完配置后，触发一次 GitHub 事件，看日志确认是否重载成功

## 9. 更新版本

镜像与配置是分离的：你的 `./configs`、`./logs` 是挂载卷，升级镜像不会覆盖它们。

```bash
docker compose pull      # 拉取最新镜像
docker compose up -d     # 用新镜像重建容器（本地配置保留）
```

升级后注意：

- 你已有的 `./configs/*.yaml`、`templates.*.jsonc` 都会保留；镜像只会在文件缺失时补上默认配置
- 新版本引入的新默认配置项，也只在你对应文件缺失时才会自动补入
- 管理面板账号：若你从老版本升级且没配置过面板账号，默认登录 `admin` / `admin`（见 [§5](#5-web-管理面板可选)）

> 从源码部署的更新方式见 [从源码构建 · 更新源码版本](build-from-source.md#更新源码版本)。

---

更多配置与高级用法见 [../README.md](../README.md)、[../configs](../configs/) 示例注释，或 [../internal/handler](../internal/handler/)、[../internal/template](../internal/template/) 下的文档。
