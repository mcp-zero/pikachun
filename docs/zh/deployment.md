# 部署指南

## 概述

本文档提供了在各种环境中部署 Pikachu'n 的说明，包括独立部署、Docker 和 Docker Compose。

## 前提条件

- Go 1.16 或更高版本（用于从源码构建）
- MySQL 5.7 或更高版本，并启用 binlog
- 访问 webhook 端点
- Docker（用于 Docker 部署）

## MySQL 配置

在部署 Pikachu'n 之前，请确保您的 MySQL 服务器已正确配置：

1. 在 `my.cnf` 中启用 binlog：
   ```ini
   [mysqld]
   log-bin=mysql-bin
   binlog-format=ROW
   server-id=1
   ```

2. 创建专门用于 binlog 读取的用户：
   ```sql
   CREATE USER 'pikachu'@'%' IDENTIFIED BY 'password';
   GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'pikachu'@'%';
   GRANT SELECT ON your_database.* TO 'pikachu'@'%';
   FLUSH PRIVILEGES;
   ```

## 独立部署

### 从源码构建

1. 克隆仓库：
   ```bash
   git clone https://github.com/your-username/pikachu-n.git
   cd pikachu-n
   ```

2. 构建二进制文件：
   ```bash
   go build -o pikachu-n .
   ```

3. 创建配置文件（`config.yaml`）：
   ```yaml
   mysql:
     host: localhost
     port: 3306
     user: pikachu
     password: password
     database: testdb

   webhook:
     url: https://example.com/webhook
   ```

4. 运行应用程序：
   ```bash
   ./pikachu-n
   ```

### 作为服务运行

#### systemd (Linux)

在 `/etc/systemd/system/pikachu-n.service` 创建 systemd 服务文件：

```ini
[Unit]
Description=Pikachu'n MySQL Binlog 监控器
After=network.target

[Service]
Type=simple
User=pikachu
WorkingDirectory=/opt/pikachu-n
ExecStart=/opt/pikachu-n/pikachu-n
Restart=always
RestartSec=10
Environment=PIKACHUN_MYSQL_PASSWORD=secure_password

[Install]
WantedBy=multi-user.target
```

启用并启动服务：
```bash
sudo systemctl enable pikachu-n
sudo systemctl start pikachu-n
```

## Docker 部署

### 构建 Docker 镜像

1. 构建镜像：
   ```bash
   docker build -t pikachu-n .
   ```

### 使用 Docker 运行

1. 在主机上创建配置文件（`config.yaml`）：
   ```yaml
   mysql:
     host: host.docker.internal
     port: 3306
     user: pikachu
     password: password
     database: testdb

   webhook:
     url: https://example.com/webhook
   ```

2. 运行容器：
   ```bash
   docker run -d \
     --name pikachu-n \
     -p 8668:8668 \
     -v $(pwd)/config.yaml:/app/config.yaml \
     -e PIKACHUN_MYSQL_PASSWORD=secure_password \
     pikachu-n
   ```

## Docker Compose

创建 `docker-compose.yml` 文件：

```yaml
version: '3.8'

services:
  pikachu-n:
    build: .
    container_name: pikachu-n
    ports:
      - "8668:8668"
    volumes:
      - ./config.yaml:/app/config.yaml
    environment:
      - PIKACHUN_MYSQL_PASSWORD=secure_password
    depends_on:
      - mysql
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    container_name: mysql
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: testdb
      MYSQL_USER: pikachu
      MYSQL_PASSWORD: password
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./mysql.cnf:/etc/mysql/conf.d/mysql.cnf
    restart: unless-stopped

volumes:
  mysql_data:
```

运行：
```bash
docker-compose up -d
```

## Kubernetes 部署

创建 `deployment.yaml` 文件：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pikachu-n
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pikachu-n
  template:
    metadata:
      labels:
        app: pikachu-n
    spec:
      containers:
      - name: pikachu-n
        image: your-registry/pikachu-n:latest
        ports:
        - containerPort: 8668
        env:
        - name: PIKACHUN_MYSQL_PASSWORD
          valueFrom:
            secretKeyRef:
              name: pikachu-n-secrets
              key: mysql-password
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: pikachu-n-config
---
apiVersion: v1
kind: Service
metadata:
  name: pikachu-n
spec:
  selector:
    app: pikachu-n
  ports:
  - protocol: TCP
    port: 8668
    targetPort: 8668
```

创建 `configmap.yaml`：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pikachu-n-config
data:
  config.yaml: |
    mysql:
      host: mysql-service
      port: 3306
      user: pikachu
      password: password
      database: testdb
    webhook:
      url: https://example.com/webhook
```

创建 `secret.yaml`：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pikachu-n-secrets
type: Opaque
data:
  mysql-password: cGFzc3dvcmQ=  # base64 编码的 "password"
```

部署：
```bash
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
```

## 监控和日志

### 健康检查

Pikachu'n 提供 `/status` 端点用于健康检查：
```bash
curl http://localhost:8668/status
```

### 日志

在 Docker 中查看日志：
```bash
docker logs pikachu-n
```

对于 systemd 服务：
```bash
sudo journalctl -u pikachu-n -f
```

## 安全考虑

1. 不要将 Web 界面暴露给公共互联网而不进行认证
2. 使用环境变量存储敏感配置
3. 确保 MySQL 用户具有最小必需权限
4. 为 Webhook URL 使用 HTTPS
5. 考虑网络级访问控制

## 故障排除

### 常见问题

1. **无法连接到 MySQL**
   - 检查 MySQL 是否正在运行且可访问
   - 验证凭据和权限
   - 确保已启用 binlog

2. **Webhook 事件未发送**
   - 检查 Webhook URL 是否正确且可访问
   - 验证网络连接
   - 检查日志中的错误消息

3. **权限被拒绝错误**
   - 确保用户具有 REPLICATION SLAVE 和 REPLICATION CLIENT 权限
   - 检查配置文件的文件权限

### 调试

通过在配置中将 `log.level` 设置为 `debug` 或设置 `PIKACHUN_LOG_LEVEL` 环境变量来启用调试日志。