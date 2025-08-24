# Deployment

## Overview

This document provides instructions for deploying Pikachu'n in various environments, including standalone deployment, Docker, and Docker Compose.

## Prerequisites

- Go 1.16 or higher (for building from source)
- MySQL 5.7 or higher with binlog enabled
- Access to a webhook endpoint
- Docker (for Docker deployment)

## MySQL Configuration

Before deploying Pikachu'n, ensure your MySQL server is properly configured:

1. Enable binlog in `my.cnf`:
   ```ini
   [mysqld]
   log-bin=mysql-bin
   binlog-format=ROW
   server-id=1
   ```

2. Create a dedicated user for binlog reading:
   ```sql
   CREATE USER 'pikachu'@'%' IDENTIFIED BY 'password';
   GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'pikachu'@'%';
   GRANT SELECT ON your_database.* TO 'pikachu'@'%';
   FLUSH PRIVILEGES;
   ```

## Standalone Deployment

### Building from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/pikachu-n.git
   cd pikachu-n
   ```

2. Build the binary:
   ```bash
   go build -o pikachu-n .
   ```

3. Create a configuration file (`config.yaml`):
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

4. Run the application:
   ```bash
   ./pikachu-n
   ```

### Running as a Service

#### systemd (Linux)

Create a systemd service file at `/etc/systemd/system/pikachu-n.service`:

```ini
[Unit]
Description=Pikachu'n MySQL Binlog Monitor
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

Enable and start the service:
```bash
sudo systemctl enable pikachu-n
sudo systemctl start pikachu-n
```

## Docker Deployment

### Building the Docker Image

1. Build the image:
   ```bash
   docker build -t pikachu-n .
   ```

### Running with Docker

1. Create a configuration file (`config.yaml`) on your host:
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

2. Run the container:
   ```bash
   docker run -d \
     --name pikachu-n \
     -p 8668:8668 \
     -v $(pwd)/config.yaml:/app/config.yaml \
     -e PIKACHUN_MYSQL_PASSWORD=secure_password \
     pikachu-n
   ```

## Docker Compose

Create a `docker-compose.yml` file:

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

Run with:
```bash
docker-compose up -d
```

## Kubernetes Deployment

Create a `deployment.yaml` file:

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

Create a `configmap.yaml`:

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

Create a `secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pikachu-n-secrets
type: Opaque
data:
  mysql-password: cGFzc3dvcmQ=  # base64 encoded "password"
```

Deploy with:
```bash
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
```

## Monitoring and Logging

### Health Checks

Pikachu'n provides a `/status` endpoint for health checks:
```bash
curl http://localhost:8668/status
```

### Logs

When running in Docker, view logs with:
```bash
docker logs pikachu-n
```

For systemd services:
```bash
sudo journalctl -u pikachu-n -f
```

## Security Considerations

1. Never expose the web interface to the public internet without authentication
2. Use environment variables for sensitive configuration
3. Ensure MySQL user has minimal required privileges
4. Use HTTPS for webhook URLs
5. Consider network-level access controls

## Troubleshooting

### Common Issues

1. **Connection to MySQL failed**
   - Check MySQL is running and accessible
   - Verify credentials and permissions
   - Ensure binlog is enabled

2. **Webhook events not being sent**
   - Check webhook URL is correct and accessible
   - Verify network connectivity
   - Check logs for error messages

3. **Permission denied errors**
   - Ensure the user has REPLICATION SLAVE and REPLICATION CLIENT privileges
   - Check file permissions on configuration files

### Debugging

Enable debug logging by setting `log.level` to `debug` in the configuration or by setting the `PIKACHUN_LOG_LEVEL` environment variable.