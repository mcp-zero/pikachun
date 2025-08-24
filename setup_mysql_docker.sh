#!/bin/bash

# MySQL Docker 快速设置脚本
# 用于快速启动一个支持 binlog 的 MySQL 实例

set -e

echo "🐳 Setting up MySQL with Binlog support using Docker..."

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# 停止并删除现有的 MySQL 容器（如果存在）
echo "🧹 Cleaning up existing MySQL container..."
docker stop mysql-binlog 2>/dev/null || true
docker rm mysql-binlog 2>/dev/null || true

# 启动 MySQL 容器
echo "🚀 Starting MySQL container with binlog enabled..."
docker run -d \
  --name mysql-binlog \
  -p 3307:3306 \
  -e MYSQL_ROOT_PASSWORD=lidi10 \
  -e MYSQL_DATABASE=test \
  -e MYSQL_USER=repl_user \
  -e MYSQL_Password=repl_pass \
  mysql:8.0 \
  --log-bin=mysql-bin \
  --binlog-format=ROW \
  --server-id=1 \
  --binlog-do-db=test

echo "⏳ Waiting for MySQL to start..."
sleep 15

# 检查 MySQL 是否启动成功
if ! docker exec mysql-binlog mysqladmin ping -h localhost --silent; then
    echo "❌ MySQL failed to start. Checking logs..."
    docker logs mysql-binlog
    exit 1
fi

echo "✅ MySQL started successfully!"

# 设置复制权限
echo "🔐 Setting up replication privileges..."
docker exec -i mysql-binlog mysql -u root -plidi10 << 'EOF'
-- 为 root 用户授予复制权限
GRANT REPLICATION SLAVE ON *.* TO 'root'@'%';

-- 创建专用的复制用户
CREATE USER IF NOT EXISTS 'repl_user'@'%' IDENTIFIED BY 'repl_pass';
GRANT REPLICATION SLAVE ON *.* TO 'repl_user'@'%';

-- 刷新权限
FLUSH PRIVILEGES;

-- 创建测试数据库和表
CREATE DATABASE IF NOT EXISTS test;
USE test;

CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100),
    email VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入一些测试数据
INSERT INTO users (name, email) VALUES 
('张三', 'zhangsan@example.com'),
('李四', 'lisi@example.com'),
('王五', 'wangwu@example.com');

-- 显示当前状态
SHOW MASTER STATUS;
SHOW VARIABLES LIKE 'log_bin';
SHOW VARIABLES LIKE 'binlog_format';
SHOW VARIABLES LIKE 'server_id';
EOF

echo "✅ MySQL setup completed!"
echo ""
echo "📋 Connection Information:"
echo "   Host: 127.0.0.1"
echo "   Port: 3306"
echo "   Username: root"
echo "   Password: lidi10"
echo "   Database: test"
echo ""
echo "🔧 Alternative replication user:"
echo "   Username: repl_user"
echo "   Password: repl_pass"
echo ""
echo "🧪 Test the connection:"
echo "   mysql -h 127.0.0.1 -P 3306 -u root -plidi10"
echo ""
echo "🚀 Now you can run the binlog test:"
echo "   cd test/binlog_test && go run main.go"