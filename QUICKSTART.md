# 快速开始

这是一个 1 分钟快速上手指南：镜像包含不可变的默认配置，并默认启用配置热重载。使用 Compose 启动时，程序会将缺失的默认配置文件初始化到本地 `./configs`，已有文件不会被覆盖；`./configs` 和 `./logs` 随后挂载到容器中，方便实时编辑和查看日志。

1. 准备（可选两种方式）

   - 方式 A：克隆仓库（适合想查看、修改源码或 `Dockerfile` 的用户）

     > 在此之前请确保已安装 `Git`, `Docker` 与 `Docker Compose`。

     ```bash
     git clone https://github.com/hnrobert/feishu-github-tracker.git
     cd feishu-github-tracker
     ```

   - 方式 B：仅下载 `docker-compose.yml`（适合希望快速运行服务的用户）

     > 在此之前请确保已安装 `wget`, `Docker` 与 `Docker Compose`。

     ```bash
     mkdir feishu-github-tracker
     cd feishu-github-tracker
     # 下载仓库中的 docker-compose.yml（示例使用 main 分支）
     wget -O docker-compose.yml https://raw.githubusercontent.com/hnrobert/feishu-github-tracker/main/docker-compose.yml
     ```

2. 启动（推荐）

  <details><summary>docker compose(现代compose)</summary>

   ```bash
   docker compose up -d
   # 查看实时日志
   docker compose logs -f
   ```

  </details>
  <details><summary>docker-compose(早期compose)</summary>

   ```bash
   docker-compose up -d
   # 查看实时日志
   docker-compose logs -f
   ```

   </details>
   说明：

- 首次启动时，程序会将镜像中的默认配置文件复制到本地配置目录；已有配置文件不会被覆盖：
  - 本地 `./configs` <-> 容器 `/app/configs`
  - 本地 `./logs` <-> 容器 `/app/logs`
- 镜像默认启用热重载（每次收到 webhook 请求时会尝试重新加载配置），因此修改 `./configs/` 目录下的配置后无需重启容器即可生效。

1. 访问健康检查

   服务默认监听在 4594 端口：

   ```bash
   http://localhost:4594/health
   ```

2. 修改配置

   - 编辑 `./configs/` 目录下的配置文件，参考 [README.md](README.md) or [configs](configs/) 目录下的示例配置文件的注释说明。你最可能需要修改的有下面几个内容：
     - [./configs/server.yaml](configs/server.yaml)：修改服务器监听地址，端口和自定义一个 `secret`（如果需要的话，如果测试用可以不设置）
     - [./configs/feishu-bots.yaml](configs/feishu-bots.yaml)：配置飞书机器人的 Webhook URL 和别名（每个可选添加配置模板选择）。
       - 如果不确定 `飞书机器人` 和 `Webhook URL` 是什么，可以参考 [这个文档](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot)，在一个群组中创建一个机器人并复制其 Webhook URL 到这里。
     - [./configs/repos.yaml](configs/repos.yaml)：配置需要监听的 GitHub 仓库和事件，以及对应的通知对象
     - [./configs/templates.jsonc](configs/templates.jsonc)：默认消息模板（可选：创建/使用 `templates.<自定义名称，如「cn」>.jsonc` 自定义模板）
   - 修改后保存，程序会在下一次收到 GitHub Webhook 请求时自动热重载最新配置。

3. Web 管理面板（可选）

   除了手改 YAML，本项目还内置一个 Web 管理面板，可在浏览器里增删改：仓库规则、飞书机器人、服务设置、事件配置、消息模板。

   - 面板地址：

     ```bash
     http://localhost:4594/
     ```

     （与 webhook 服务同端口；`/webhook`、`/health` 仍照常工作。）

   - 设置管理员账号：
     - 默认账号：用户名 `admin` / 密码 `admin`。默认配置文件已带 `panel.password: "admin"`；从老版本升级且未配置面板账号时，也会自动使用 `admin` / `admin`。
     - 用户名：默认 `admin`，可在 [./configs/server.yaml](configs/server.yaml) 的 `panel.username` 或环境变量 `PANEL_USERNAME` 中自定义；也可在面板「服务设置」页直接修改。
     - 密码（优先级从高到低）：
       - 环境变量（推荐）：`PANEL_PASSWORD=你的密码`
       - [./configs/server.yaml](configs/server.yaml) 的 `panel.password`（明文）。**若存在这一项则优先使用它**：启动 / reload 时会自动转为 `password_hash`（覆盖原 hash）、删除该明文行，并补回一行 `# password: "admin"` 注释。
       - [./configs/server.yaml](configs/server.yaml) 的 `panel.password_hash`（bcrypt，可用 `htpasswd -bnBC 10 "" 你的密码 | tr -d ':\n' | sed 's/^\$2y/\$2a/'` 生成）
     - 在面板「服务设置」页修改用户名或密码：修改密码需先填写「当前密码」校验通过后才生效；新密码会生成 `password_hash`（同时补回 `# password: "admin"` 注释）。保存后下次登录即用新账号，无需重启。

   - 面板内的修改保存后会自动 reload 生效（无需等待下一次 webhook，也无需重启）；手动编辑 `./configs/` 下的文件则需以 `--reload` 启动或重启进程。端口 / 密钥的改动仍需重启进程。

   - 注意：在「消息模板」页保存 `templates.*.jsonc` 会移除文件中的 `//` 注释并按字母重排键（功能不变）。

4. 多模板配置（可选）

   如果需要为不同的飞书 bot 配置不同的消息模板（如中英文双语），可以在 `./configs/feishu-bots.yaml` 中指定模板：

   ```yaml
   feishu_bots:
     - alias: 'team-cn'
       url: 'https://open.feishu.cn/open-apis/bot/v2/hook/cn-webhook'
       template: 'cn' # 使用中文模板，如不设置默认使用 templates.jsonc 英文模板
   ```

   也可以根据现有的修改并创建新的模版文件 `templates.<自定义名称>.jsonc`，然后在 `feishu-bots.yaml` 中引用。

5. 添加 GitHub Webhook

   - 进入你想监听的 GitHub 仓库，点击 `Settings` -> `Webhooks` -> `Add webhook`
   - 在 `Payload URL` 中填入你的服务器地址，例如 `http://your-domain-or-ip:4594/webhook`
   - 在 `Content type` 中选什么都可以，都支持
   - 在 `Secret` 中填入你在 `server.yaml` 中配置的 `secret`（如果配置了的话）
   - 选择你想监听的事件类型，可以选 `仅push`，也可以选 `Everything` 然后在这个项目的 [configs/events.yaml](configs/events.yaml) 中更详细地选择你想要监听每个事件什么类型甚至什么分支上的事件；如果想要简单一些，也选择 `Let me select individual events` 然后勾选需要的事件，而在这边的 [./configs/repos.yaml](configs/repos.yaml) 你想监听的项目中选择 `all:`
   - 点击 `Add webhook` 保存
   - **✅ 成功提示**：如果前面的步骤没有错误，几秒钟后你会在飞书群组中收到一条 "GitHub Webhook 添加成功" 的通知（这是 GitHub 发送的 ping 事件）。这表示 Webhook 已正确配置并能正常工作！

6. 简要调试

   - 若没有收到通知，请检查：
     - GitHub Webhook 配置（Payload URL、Secret、事件类型）
     - `docker-compose logs -f` 中的错误日志，如果没有任何日志，请确认 GitHub Webhook 是否有成功发送请求
   - 修改配置后，尝试触发一次 GitHub 的 Webhook 事件，然后查看日志确认是否重载成功

更多配置与高级用法请在启动容器后参见 [README.md](README.md) or [configs](configs/) 目录下的示例配置文件的注释说明 or [internal/handler/](internal/handler/) 目录下的相关文档 or [internal/template/](internal/template/) 目录下的消息模板说明。
