# Pikachun MySQL Binlog 故障排除指南

## 📋 目录

- [常见错误及解决方案](#常见错误及解决方案)
  - [Access denied for user](#错误1-access-denied-for-user-rootlocalhost)
  - [Binlog 未启用](#错误2-binlog-未启用)
  - [Server ID 冲突](#错误3-server-id-冲突)
- [使用 Docker 快速设置 MySQL](#使用-docker-快速设置-mysql)
- [测试连接](#测试连接)
- [配置文件示例](#配置文件示例)
- [调试技巧](#调试技巧)
- [获取帮助](#获取帮助)

## 🚨 常见错误及解决方案

### 错误1: Access denied for user 'root'@'localhost'

```
ERROR 1045 (28000): Access denied for user 'root'@'localhost' (using password: NO)
```

**原因分析：**
1. MySQL 用户密码不正确
2. 用户没有足够的权限
3. MySQL 服务未启动
4. 网络连接问题

**解决步骤：**

#### 步骤1: 检查 MySQL 服务状态

```bash
# macOS
brew services list | grep mysql
sudo launchctl list | grep mysql

# Linux
systemctl status mysql
# 或
systemctl status mysqld

# Windows
net start | findstr MySQL
```

#### 步骤2: 启动 MySQL 服务

```bash
# macOS (Homebrew)
brew services start mysql

# macOS (系统安装)
sudo launchctl load -w /Library/LaunchDaemons/com.oracle.oss.mysql.mysqld.plist

# Linux
sudo systemctl start mysql
# 或
sudo systemctl start mysqld

# Windows
net start MySQL80
```

#### 步骤3: 重置 MySQL root 密码

```bash
# 停止 MySQL 服务
sudo mysqld_safe --skip-grant-tables --skip-networking &

# 连接 MySQL（无密码）
mysql -u root

# 在 MySQL 中执行
USE mysql;
UPDATE user SET authentication_string=PASSWORD('lidi10') WHERE User='root';
FLUSH PRIVILEGES;
EXIT;

# 重启 MySQL 服务
sudo killall mysqld
sudo systemctl start mysql
```

#### 步骤4: 创建复制用户并授权

```sql
-- 连接 MySQL
mysql -u root -p

-- 创建复制用户
CREATE USER 'repl_user'@'%' IDENTIFIED BY 'repl_pass';
GRANT REPLICATION SLAVE ON *.* TO 'repl_user'@'%';

-- 为 root 用户添加复制权限
GRANT REPLICATION SLAVE ON *.* TO 'root'@'%';
GRANT REPLICATION SLAVE ON *.* TO 'root'@'localhost';

-- 刷新权限
FLUSH PRIVILEGES;

-- 验证权限
SHOW GRANTS FOR 'root'@'%';
```

### 错误2: Binlog 未启用

```
ERROR: The MySQL server is not configured as a master
```

**解决方案：**

#### 步骤1: 检查 binlog 状态

```sql
SHOW VARIABLES LIKE 'log_bin';
SHOW VARIABLES LIKE 'binlog_format';
SHOW VARIABLES LIKE 'server_id';
```

#### 步骤2: 配置 MySQL binlog

编辑 MySQL 配置文件：

**macOS (Homebrew):** `/usr/local/etc/my.cnf`
**Linux:** `/etc/mysql/my.cnf` 或 `/etc/my.cnf`
**Windows:** `C:\ProgramData\MySQL\MySQL Server 8.0\my.ini`

添加以下配置：

```ini
[mysqld]
# 启用 binlog
log-bin=mysql-bin
binlog-format=ROW
server-id=1

# 可选配置
expire_logs_days=7
max_binlog_size=100M
```

#### 步骤3: 重启 MySQL 服务

```bash
# macOS
brew services restart mysql

# Linux
sudo systemctl restart mysql

# Windows
net stop MySQL80
net start MySQL80
```

### 错误3: Server ID 冲突

```
ERROR: Slave has the same server_id as master
```

**解决方案：**
修改 `config.yaml` 中的 `server_id`，确保与 MySQL 的 `server_id` 不同：

```yaml
canal:
  server_id: 1001  # 确保与 MySQL 的 server_id 不同
```

## 🐳 使用 Docker 快速设置 MySQL

如果本地 MySQL 配置复杂，推荐使用 Docker：

### 步骤1: 安装并启动 Docker

```bash
# macOS
brew install --cask docker
# 启动 Docker Desktop

# Linux
sudo apt-get install docker.io
sudo systemctl start docker

# Windows
# 下载并安装 Docker Desktop
```

### 步骤2: 运行 MySQL 容器

```bash
# 启动 MySQL 容器
docker run -d \
  --name mysql-binlog \
  -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=lidi10 \
  -e MYSQL_DATABASE=test \
  mysql:8.0 \
  --log-bin=mysql-bin \
  --binlog-format=ROW \
  --server-id=1

# 等待启动
sleep 15

# 设置权限
docker exec -i mysql-binlog mysql -u root -plidi10 << 'EOF'
GRANT REPLICATION SLAVE ON *.* TO 'root'@'%';
FLUSH PRIVILEGES;

CREATE DATABASE IF NOT EXISTS test;
USE test;
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100),
    email VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
EOF
```

## 🧪 测试连接

### 步骤1: 测试基本连接

```bash
cd test/connection_test
go run main.go
```

### 步骤2: 测试 binlog 功能

```bash
cd test/binlog_test
go run main.go
```

### 步骤3: 在另一个终端执行 MySQL 操作

```sql
mysql -h 127.0.0.1 -P 3306 -u root -plidi10

USE test;
INSERT INTO users (name, email) VALUES ('测试用户', 'test@example.com');
UPDATE users SET email = 'updated@example.com' WHERE id = 1;
DELETE FROM users WHERE id = 1;
```

## 📋 配置文件示例

### config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: "8668"

database:
  dsn: "./data/pikachun.db"

canal:
  host: "127.0.0.1"
  port: 3306
  username: "root"
  password: "lidi10"
  charset: "utf8mb4"
  server_id: 1001
  
  binlog:
    filename: ""
    position: 4
    gtid_enabled: true
    
  watch:
    databases: ["test"]
    tables: ["users"]
    event_types: ["INSERT", "UPDATE", "DELETE"]
    
  reconnect:
    max_attempts: 10
    interval: "5s"
    
  performance:
    event_buffer_size: 1000
    batch_size: 100

log:
  level: "info"
  file: "./logs/pikachun.log"
```

## 🔍 调试技巧

### 1. 启用详细日志

```bash
# 运行时查看日志
tail -f /tmp/pikachun.log

# 或者直接运行查看输出
./pikachun
```

### 2. 检查 MySQL 状态

```sql
-- 检查 binlog 状态
SHOW MASTER STATUS;
SHOW BINARY LOGS;

-- 检查用户权限
SHOW GRANTS FOR 'root'@'%';

-- 检查变量
SHOW VARIABLES LIKE '%binlog%';
SHOW VARIABLES LIKE '%server_id%';
```

### 3. 网络连接测试

```bash
# 测试端口连通性
telnet 127.0.0.1 3306

# 或使用 nc
nc -zv 127.0.0.1 3306
```

## 📞 获取帮助

如果以上方法都无法解决问题，请：

1. 检查 MySQL 错误日志
2. 确认 MySQL 版本兼容性（推荐 5.7+ 或 8.0+）
3. 检查防火墙设置
4. 确认网络配置

**常用 MySQL 日志位置：**
- macOS: `/usr/local/var/mysql/`
- Linux: `/var/log/mysql/`
- Windows: `C:\ProgramData\MySQL\MySQL Server 8.0\Data\`

```bash
source ~/.gvm/scripts/gvm && gvm use go1.23.0 && CGO_CFLAGS="-Wno-nullability-completeness" go run main.go

source ~/.gvm/scripts/gvm && gvm use go1.23.0 && go run test/webhook_server/main.go

source ~/.gvm/scripts/gvm && gvm use go1.23.0 && CGO_CFLAGS="-Wno-nullability-completeness" go test ./test/unit/... -v
```

## 📚 相关文档

- [MySQL配置指南](setup_mysql.md)
- [Pikachun README](README.md)