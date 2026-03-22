# Syslog Attendance Console

基于 Go、React 和 MySQL 的本地考勤控制台，当前已经具备真实的员工档案、日志入站、考勤复核、飞书打卡上报、打卡成功通知，以及前端调试工具闭环。

## 项目结构

- `backend/` Go 后端
- `frontend/` Vite 前端
- `scripts/send-sample-syslog.sh` 发送 connect / disconnect 调试 syslog
- `scripts/reset-dev-data.sh` 清理本地联调过程中残留的历史演示数据
- `docker-compose.yml` 本地 mysql + backend + frontend 运行栈

## 本地启动 Compose

```bash
docker compose up mysql backend frontend
```

## Ubuntu 24 服务器部署

正式环境部署说明见：

- [docs/deploy-ubuntu24.md](/Users/wesleyw/Project/syslog/docs/deploy-ubuntu24.md)

服务端口：

- `mysql`：`3306`
- `backend` HTTP：`http://127.0.0.1:8080`
- `backend` syslog UDP：`127.0.0.1:514/udp`
- `frontend`：`http://127.0.0.1:5173`

说明：

- `backend` 使用官方 `golang:1.22` 镜像，挂载当前仓库并执行 `go run ./cmd/server`
- `backend` 容器内显式注入 `MYSQL_HOST=mysql`，避免宿主机本地默认配置和 compose 网络配置互相冲突
- `frontend` 使用官方 `node:22` 镜像，挂载当前仓库并执行 `npm install && npm run dev`
- `frontend` 容器内显式注入 `VITE_API_PROXY_TARGET=http://backend:8080`，让 Vite 代理把 `/api/*` 转发到 compose 网络里的后端服务，而不是前端容器自己的 `127.0.0.1`
- `frontend` 额外使用独立的容器卷保存 `/workspace/frontend/node_modules`，避免把宿主机上的 macOS 依赖直接挂进 Linux 容器导致 Rollup 原生包缺失
- `frontend` 启动前会额外补装一次 `@rollup/rollup-linux-x64-gnu`，绕过 npm 在 Linux 容器内漏装 Rollup optional dependency 的已知问题
- 当前后端容器内 UDP listener 实际监听 `1514/udp`，通过 compose 映射到宿主机 `514/udp`

如果你之前已经跑过失败的 compose，建议先清理再重启：

```bash
docker compose down -v
docker compose up mysql backend frontend
```

## 本地直接启动后端

如果不走 compose，后端默认会连接 `127.0.0.1:3306` 的 MySQL：

```bash
cd backend
go run ./cmd/server
```

本机直跑后端时，syslog UDP 默认监听 `127.0.0.1:1514/udp` 对应的 `:1514`。如果你的 AP 只能发到 `514/udp`，可以显式改成：

```bash
cd backend
SYSLOG_UDP_ADDR=:514 go run ./cmd/server
```

注意：

- 在 macOS/Linux 上绑定 `514` 这类特权端口通常需要更高权限；如果直接报权限错误，需要用具备权限的方式启动进程
- 如果不想让本机 Go 进程直接抢占 `514`，继续使用 Docker 后端也是可行方案，宿主机 `514/udp` 会映射到容器内 `1514/udp`

需要自定义数据库地址时，再显式设置环境变量：

```bash
cd backend
MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=syslog MYSQL_PASSWORD=syslog MYSQL_DATABASE=syslog go run ./cmd/server
```

## 后端测试

运行最小闭环集成测试：

```bash
cd backend && go test ./tests/integration -run TestSyslogFlow -v
```

运行全部 Go 测试：

```bash
cd backend && go test ./...
```

## 前端测试

```bash
cd frontend && npm test
cd frontend && npm run build
```

## 发送调试 Syslog

默认发送到 `127.0.0.1:514/udp`：

```bash
./scripts/send-sample-syslog.sh
```

前提：本机需要已安装 `nc`/`netcat`。

只发 connect：

```bash
./scripts/send-sample-syslog.sh connect
```

只发 disconnect：

```bash
./scripts/send-sample-syslog.sh disconnect
```

自定义目标：

```bash
SYSLOG_HOST=127.0.0.1 SYSLOG_PORT=514 ./scripts/send-sample-syslog.sh both
```

## 清理本地演示数据

如果你之前用调试页、样例 syslog 或手工维护过演示员工，建议在切真实数据前先清理一遍运行期数据。

默认仅清理日志、事件、考勤和上报记录，保留员工档案与系统设置：

```bash
./scripts/reset-dev-data.sh
```

如果员工档案本身也是演示数据，可以连同员工和设备一起清理：

```bash
./scripts/reset-dev-data.sh --with-employees
```

如果要完全回到空白状态，包括系统设置：

```bash
./scripts/reset-dev-data.sh --with-employees --with-settings --yes
```

## 当前实现边界

- 集成测试当前覆盖服务级闭环：`AP syslog parser -> attendance processor -> day-end -> report service -> 飞书上报/通知`
- 后端主程序已经真实监听 UDP 和 HTTP，调试 syslog 会进入持久化、考勤计算和后续上报链路
- MySQL 是运行时必需组件，员工、日志、考勤、上报与设置都落在数据库中
- 前端已经切到真实 `/api/*`，不再依赖运行时 mock 数据
