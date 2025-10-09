# 快速开始

这是一个 1 分钟快速上手指南：当前镜像默认已将 `configs` 打包进镜像，并默认启用配置热重载；使用 `docker-compose` 启动时，本地的 `./configs` 和 `./logs` 会自动挂载到容器中（如果本地不存在会自动创建新文件夹/放入默认配置），方便实时编辑和查看日志。

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

   ```bash
   docker-compose up -d
   # 查看实时日志
   docker-compose logs -f
   ```

   说明：

   - 默认镜像包含已经打包的配置文件，容器启动后会自动把配置目录和日志目录同步到宿主机：
     - 本地 `./configs` <-> 容器 `/app/configs`
     - 本地 `./logs` <-> 容器 `/app/logs`
   - 镜像默认启用热重载（每次收到 webhook 请求时会尝试重新加载配置），因此修改 `./configs/` 目录下的配置后无需重启容器即可生效。

3. 访问健康检查

   服务默认监听在 4594 端口：

   ```bash
   http://localhost:4594/health
   ```

4. 修改配置

   - 编辑 `./configs/` 目录下的配置文件，参考 [README.md](README.md) or [configs](configs/) 目录下的示例配置文件的注释说明。你最可能需要修改的有下面几个内容：
     - [./configs/server.yaml](configs/server.yaml)：修改服务器监听地址，端口和自定义一个 `secret`（如果需要的话，如果测试用可以不设置）
     - [./configs/feishu-bots.yaml](configs/feishu-bots.yaml)：配置飞书机器人的 Webhook URL 和别名（每个可选添加配置模板选择）。
       - 如果不确定 `飞书机器人` 和 `Webhook URL` 是什么，可以参考 [这个文档](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot)，在一个群组中创建一个机器人并复制其 Webhook URL 到这里。
     - [./configs/repos.yaml](configs/repos.yaml)：配置需要监听的 GitHub 仓库和事件，以及对应的通知对象
     - [./configs/templates.jsonc](configs/templates.jsonc)：默认消息模板（可选：创建/使用 `templates.<自定义名称，如「cn」>.jsonc` 自定义模板）
   - 修改后保存，程序会在下一次收到 GitHub Webhook 请求时自动热重载最新配置。

5. 多模板配置（可选）

   如果需要为不同的飞书 bot 配置不同的消息模板（如中英文双语），可以在 `./configs/feishu-bots.yaml` 中指定模板：

   ```yaml
   feishu_bots:
     - alias: 'team-cn'
       url: 'https://open.feishu.cn/open-apis/bot/v2/hook/cn-webhook'
       template: 'cn' # 使用中文模板，如不设置默认使用 templates.jsonc 英文模板
   ```

   也可以根据现有的修改并创建新的模版文件 `templates.<自定义名称>.jsonc`，然后在 `feishu-bots.yaml` 中引用。

6. 添加 GitHub Webhook

   - 进入你想监听的 GitHub 仓库，点击 `Settings` -> `Webhooks` -> `Add webhook`
   - 在 `Payload URL` 中填入你的服务器地址，例如 `http://your-domain-or-ip:4594/webhook`
   - 在 `Content type` 中选什么都可以，都支持
   - 在 `Secret` 中填入你在 `server.yaml` 中配置的 `secret`（如果配置了的话）
   - 选择你想监听的事件类型，可以选 `仅push`，也可以选 `Everything` 然后在这个项目的 [configs/events.yaml](configs/events.yaml) 中更详细地选择你想要监听每个事件什么类型甚至什么分支上的事件；如果想要简单一些，也选择 `Let me select individual events` 然后勾选需要的事件，而在这边的 [./configs/repos.yaml](configs/repos.yaml) 你想监听的项目中选择 `all:`
   - 点击 `Add webhook` 保存，如果前面的步骤没有错误，稍等片刻你应该就可以在群组中收到第一个成功的通知了

7. 简要调试

   - 若没有收到通知，请检查：
     - GitHub Webhook 配置（Payload URL、Secret、事件类型）
     - `docker-compose logs -f` 中的错误日志，如果没有任何日志，请确认 GitHub Webhook 是否有成功发送请求
   - 修改配置后，尝试触发一次 GitHub 的 Webhook 事件，然后查看日志确认是否重载成功

更多配置与高级用法请在启动容器后参见 [README.md](README.md) or [configs](configs/) 目录下的示例配置文件的注释说明 or [internal/handler/](internal/handler/) 目录下的相关文档 or [internal/template/](internal/template/) 目录下的消息模板说明。
