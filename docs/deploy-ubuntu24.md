# Ubuntu 24 部署指南

本文档说明如何将当前 `Syslog Attendance Console` 部署到一台 Ubuntu 24 服务器。

推荐采用以下结构：

- `MySQL 8.x`
- `Go` 后端作为 `systemd` 服务运行
- `Nginx` 提供前端静态文件并反向代理 `/api/*`

不建议直接使用仓库中的开发版 [docker-compose.yml](/Users/wesleyw/Project/syslog/docker-compose.yml) 上生产，因为它当前使用的是：

- 后端 `go run`
- 前端 `vite dev`

这更适合本地联调，不适合正式环境。

## 1. 服务器准备

更新系统并安装基础依赖：

```bash
sudo apt update
sudo apt install -y nginx mysql-server golang-go nodejs npm
```

如果你需要更高版本的 Node.js，建议使用 `nvm` 或 NodeSource 安装 Node 22。

## 2. 获取代码

```bash
sudo mkdir -p /opt/syslog
sudo chown -R $USER:$USER /opt/syslog
cd /opt/syslog
git clone <你的仓库地址> .
```

## 3. 配置 MySQL

进入 MySQL：

```bash
sudo mysql
```

执行：

```sql
CREATE DATABASE syslog CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'syslog'@'127.0.0.1' IDENTIFIED BY '请替换成强密码';
GRANT ALL PRIVILEGES ON syslog.* TO 'syslog'@'127.0.0.1';
FLUSH PRIVILEGES;
```

## 4. 构建前端

```bash
cd /opt/syslog/frontend
npm ci
npm run build
```

将前端构建产物发布到 Nginx 静态目录：

```bash
sudo mkdir -p /var/www/syslog
sudo cp -r /opt/syslog/frontend/dist/* /var/www/syslog/
```

## 5. 构建后端

```bash
cd /opt/syslog/backend
go mod download
mkdir -p /opt/syslog/bin
go build -o /opt/syslog/bin/syslog-server ./cmd/server
```

## 6. 配置后端环境变量

创建目录：

```bash
sudo mkdir -p /etc/syslog
```

创建文件 `/etc/syslog/syslog.env`：

```bash
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_USER=syslog
MYSQL_PASSWORD=请替换成强密码
MYSQL_DATABASE=syslog
MYSQL_PARAMS=charset=utf8mb4&parseTime=true&loc=Asia/Shanghai&multiStatements=true
SYSLOG_RETENTION_DAYS=30
SYSLOG_UDP_ADDR=:514
```

说明：

- `SYSLOG_UDP_ADDR=:514` 表示直接监听标准 syslog 端口
- 如果你不想让进程占用特权端口，可以改成 `:1514`

## 7. 创建 systemd 服务

创建文件 `/etc/systemd/system/syslog-backend.service`：

```ini
[Unit]
Description=Syslog Attendance Backend
After=network.target mysql.service
Wants=mysql.service

[Service]
User=root
WorkingDirectory=/opt/syslog/backend
EnvironmentFile=/etc/syslog/syslog.env
ExecStart=/opt/syslog/bin/syslog-server
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

加载并启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now syslog-backend
sudo systemctl status syslog-backend
```

查看运行日志：

```bash
journalctl -u syslog-backend -f
```

## 8. 配置 Nginx

创建文件 `/etc/nginx/sites-available/syslog`：

```nginx
server {
    listen 80;
    server_name 你的域名或服务器IP;

    root /var/www/syslog;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
        try_files $uri /index.html;
    }
}
```

启用配置：

```bash
sudo ln -sf /etc/nginx/sites-available/syslog /etc/nginx/sites-enabled/syslog
sudo nginx -t
sudo systemctl reload nginx
```

## 9. 配置防火墙

如果启用了 `ufw`：

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 514/udp
sudo ufw enable
```

如果你使用的是 `1514/udp`，把上面的 `514/udp` 改成 `1514/udp`。

## 10. 首次验证

检查后端 API：

```bash
curl http://127.0.0.1:8080/api/settings
```

打开浏览器访问：

```text
http://你的域名或服务器IP/
```

发送一条测试 syslog：

```bash
SYSLOG_HOST=你的服务器IP SYSLOG_PORT=514 /opt/syslog/scripts/send-sample-syslog.sh connect
```

如果你的服务监听的是 `1514`：

```bash
SYSLOG_HOST=你的服务器IP SYSLOG_PORT=1514 /opt/syslog/scripts/send-sample-syslog.sh connect
```

## 11. 后续更新

更新代码后建议按以下顺序发布：

```bash
cd /opt/syslog
git pull

cd /opt/syslog/frontend
npm ci
npm run build
sudo rm -rf /var/www/syslog/*
sudo cp -r /opt/syslog/frontend/dist/* /var/www/syslog/

cd /opt/syslog/backend
go mod download
go build -o /opt/syslog/bin/syslog-server ./cmd/server

sudo systemctl restart syslog-backend
sudo systemctl reload nginx
```

## 12. 生产建议

- 给 Nginx 配置 HTTPS
- 给 MySQL 做定时备份
- 用单独的系统用户运行后端，而不是长期使用 `root`
- 如果必须监听 `514/udp`，可后续改成给二进制加 `CAP_NET_BIND_SERVICE`，避免整进程以 root 运行
- 如需容器化生产部署，建议另外补正式 `Dockerfile` 和 `docker-compose.prod.yml`
