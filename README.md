# Syslog Attendance Console

基于 Go、React 和 MySQL 的本地考勤演示骨架，当前已经补齐最小的本地联调入口、样例 syslog 发送脚本，以及一个服务级闭环集成测试。

## 项目结构

- `backend/` Go 后端
- `frontend/` Vite 前端
- `scripts/send-sample-syslog.sh` 发送 connect / disconnect 样例 syslog
- `docker-compose.yml` 本地 mysql + backend + frontend 运行栈

## 本地启动 Compose

```bash
docker compose up mysql backend frontend
```

服务端口：

- `mysql`：`3306`
- `backend` HTTP：`http://127.0.0.1:8080`
- `backend` syslog UDP：`127.0.0.1:5514/udp`
- `frontend`：`http://127.0.0.1:5173`

说明：

- `backend` 使用官方 `golang:1.22` 镜像，挂载当前仓库并执行 `go run ./cmd/server`
- `frontend` 使用官方 `node:22` 镜像，挂载当前仓库并执行 `npm install && npm run dev`
- 当前后端容器内 UDP listener 实际监听 `1514/udp`，通过 compose 映射到宿主机 `5514/udp`

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

## 发送样例 Syslog

默认发送到 `127.0.0.1:5514/udp`：

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
SYSLOG_HOST=127.0.0.1 SYSLOG_PORT=5514 ./scripts/send-sample-syslog.sh both
```

## 当前实现边界

- 集成测试当前验证的是服务级闭环：`AP syslog parser -> attendance processor -> day-end -> report service`
- 当前后端主程序已经监听 UDP 和 HTTP，但 UDP handler 仍是占位实现，发送样例 syslog 主要用于本地联调链路和端口验证，不会形成完整持久化入库
- MySQL 已进入 compose 栈，为后续真实存储与调度接入预留环境；当前测试闭环不依赖数据库
- 前端已经具备 Task 8 的本地 mock 数据流，但尚未接真实后端 API
