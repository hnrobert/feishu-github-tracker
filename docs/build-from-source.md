# 从源码构建与部署

适用于想查看 / 修改源码，或自行构建镜像的用户。普通部署请看 [快速开始](quickstart.md)。

## 前置条件

- `Git`
- `Docker` 与 `Docker Compose`（容器构建方式），**或** `Go 1.21+`（本地直接编译）

## 克隆仓库

```bash
git clone https://github.com/hnrobert/feishu-github-tracker.git
cd feishu-github-tracker
```

## 方式一：Docker Compose 本地构建镜像

编辑 `docker-compose.yml`，把 `image:` 注释掉、启用 `build:`：

```yaml
services:
  feishu-github-tracker:
    # image: ghcr.io/hnrobert/feishu-github-tracker:latest
    build:
      context: .
      dockerfile: Dockerfile
```

然后构建并启动：

```bash
docker compose up -d --build
docker compose logs -f
```

也可以直接用 Makefile：

```bash
make docker-build   # = docker-compose build
make docker-up      # = docker-compose up -d
```

## 方式二：本地 Go 编译直接运行

```bash
# 编译
go build -o bin/feishu-github-tracker ./cmd/feishu-github-tracker

# 运行（首次会从 example-configs 初始化 ./configs；-reload 启用配置热重载）
CONFIG_DIR=./configs DEFAULT_CONFIG_DIR=./example-configs LOG_DIR=./logs ./bin/feishu-github-tracker -reload
```

或直接 `go run`（等价于 `make run`）：

```bash
CONFIG_DIR=./configs DEFAULT_CONFIG_DIR=./example-configs LOG_DIR=./logs go run ./cmd/feishu-github-tracker -reload
```

`-reload` 会在每次收到 webhook 时重新加载 `./configs/`，修改配置后无需重启即生效。

## 更新源码版本

```bash
git pull

# 容器：重新构建并启动
docker compose up -d --build

# 或本地：重新编译后运行
go build -o bin/feishu-github-tracker ./cmd/feishu-github-tracker
```

`./configs` 是本地运行时配置，首次启动由 `./example-configs` 初始化，升级不会覆盖你已有的配置；更新后的示例可直接在 `example-configs/` 中对比。

---

更多说明见 [快速开始](quickstart.md) 与 [../README.md](../README.md)。
