# MySQL Binlog 设置指南

## 📋 目录

- [启用 MySQL Binlog](#启用-mysql-binlog)
- [创建复制用户](#创建复制用户)
- [验证设置](#验证设置)
- [测试连接](#测试连接)
- [常见问题](#常见问题)
- [Docker MySQL 快速设置](#docker-mysql-快速设置)

## 启用 MySQL Binlog

### 检查当前 binlog 状态

```sql
-- 检查 binlog 是否启用
SHOW VARIABLES LIKE 'log_bin';

-- 检查 binlog 格式
SHOW VARIABLES LIKE 'binlog_format';

-- 检查 server_id
SHOW VARIABLES LIKE 'server_id';
```

### 配置 MySQL (my.cnf 或 my.ini)

```ini
[mysqld]
# 启用 binlog
log-bin=mysql-bin
binlog-format=ROW
server-id=1

# 可选：设置 binlog 过期时间（天）
expire_logs_days=7

# 可选：设置 binlog 文件大小
max_binlog_size=100M
```

## 创建复制用户

```sql
-- 创建用于 binlog 复制的用户
CREATE USER 'repl_user'@'%' IDENTIFIED BY 'repl_password';

-- 授予复制权限
GRANT REPLICATION SLAVE ON *.* TO 'repl_user'@'%';

-- 刷新权限
FLUSH PRIVILEGES;

-- 或者使用现有的 root 用户（确保有复制权限）
GRANT REPLICATION SLAVE ON *.* TO 'root'@'%';
FLUSH PRIVILEGES;
```

## 验证设置

```sql
-- 查看当前 binlog 位置
SHOW MASTER STATUS;

-- 查看 binlog 文件列表
SHOW BINARY LOGS;

-- 查看用户权限
SHOW GRANTS FOR 'root'@'%';
```

## 测试连接

使用以下命令测试 MySQL 连接：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p
```

## 常见问题

### 问题1: Access denied

```
ERROR 1045 (28000): Access denied for user 'root'@'localhost'
```

**解决方案:**
1. 检查用户名和密码
2. 确保用户有 REPLICATION SLAVE 权限
3. 检查 MySQL 的 bind-address 设置

### 问题2: Binlog 未启用

```
ERROR: The MySQL server is not configured as a master
```

**解决方案:**
1. 在 my.cnf 中添加 binlog 配置
2. 重启 MySQL 服务

### 问题3: Server ID 冲突

```
ERROR: Slave has the same server_id as master
```

**解决方案:**
1. 确保 Pikachun 的 server_id 与 MySQL 的 server_id 不同
2. 修改 config.yaml 中的 server_id

## Docker MySQL 快速设置

如果使用 Docker，可以快速启动一个启用 binlog 的 MySQL：

```bash
docker run -d \
  --name mysql-binlog \
  -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=lidi10 \
  -e MYSQL_DATABASE=test \
  mysql:8.0 \
  --log-bin=mysql-bin \
  --binlog-format=ROW \
  --server-id=1
```

然后连接并设置权限：

```bash
docker exec -it mysql-binlog mysql -u root -p

# 在 MySQL 中执行
GRANT REPLICATION SLAVE ON *.* TO 'root'@'%';
FLUSH PRIVILEGES;
```

## 📚 相关文档

- [故障排除指南](TROUBLESHOOTING.md)
- [Pikachun 配置说明](README.md#配置说明)