# MySQL Binlog Setup Guide

## 📋 Table of Contents

- [Enable MySQL Binlog](#enable-mysql-binlog)
- [Create Replication User](#create-replication-user)
- [Verify Setup](#verify-setup)
- [Test Connection](#test-connection)
- [Common Issues](#common-issues)
- [Docker MySQL Quick Setup](#docker-mysql-quick-setup)

## Enable MySQL Binlog

### Check Current Binlog Status

```sql
-- Check if binlog is enabled
SHOW VARIABLES LIKE 'log_bin';

-- Check binlog format
SHOW VARIABLES LIKE 'binlog_format';

-- Check server_id
SHOW VARIABLES LIKE 'server_id';
```

### Configure MySQL (my.cnf or my.ini)

```ini
[mysqld]
# Enable binlog
log-bin=mysql-bin
binlog-format=ROW
server-id=1

# Optional: Set binlog expiration time (days)
expire_logs_days=7

# Optional: Set binlog file size
max_binlog_size=100M
```

## Create Replication User

```sql
-- Create user for binlog replication
CREATE USER 'repl_user'@'%' IDENTIFIED BY 'repl_password';

-- Grant replication privileges
GRANT REPLICATION SLAVE ON *.* TO 'repl_user'@'%';

-- Flush privileges
FLUSH PRIVILEGES;

-- Or use existing root user (ensure it has replication privileges)
GRANT REPLICATION SLAVE ON *.* TO 'root'@'%';
FLUSH PRIVILEGES;
```

## Verify Setup

```sql
-- View current binlog position
SHOW MASTER STATUS;

-- View binlog file list
SHOW BINARY LOGS;

-- View user privileges
SHOW GRANTS FOR 'root'@'%';
```

## Test Connection

Use the following command to test MySQL connection:

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p
```

## Common Issues

### Issue 1: Access denied

```
ERROR 1045 (28000): Access denied for user 'root'@'localhost'
```

**Solution:**
1. Check username and password
2. Ensure user has REPLICATION SLAVE privileges
3. Check MySQL's bind-address setting

### Issue 2: Binlog not enabled

```
ERROR: The MySQL server is not configured as a master
```

**Solution:**
1. Add binlog configuration to my.cnf
2. Restart MySQL service

### Issue 3: Server ID conflict

```
ERROR: Slave has the same server_id as master
```

**Solution:**
1. Ensure Pikachun's server_id is different from MySQL's server_id
2. Modify server_id in config.yaml

## Docker MySQL Quick Setup

If using Docker, you can quickly start a MySQL with binlog enabled:

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

Then connect and set permissions:

```bash
docker exec -it mysql-binlog mysql -u root -p

# Execute in MySQL
GRANT REPLICATION SLAVE ON *.* TO 'root'@'%';
FLUSH PRIVILEGES;
```

## 📚 Related Documentation

- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [Pikachun Configuration Instructions](README_EN.md#configuration-instructions)