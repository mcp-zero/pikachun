-- 创建测试数据库和表的脚本
-- 用于快速体验 Pikachu'n 服务

USE testdb;

-- 创建测试表
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 插入测试数据
INSERT INTO users (name, email) VALUES 
('张三', 'zhangsan@example.com'),
('李四', 'lisi@example.com'),
('王五', 'wangwu@example.com');

-- 更新测试数据
UPDATE users SET name = '张三丰' WHERE email = 'zhangsan@example.com';

-- 删除测试数据
DELETE FROM users WHERE email = 'lisi@example.com';

-- 创建另一个测试表
CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入产品测试数据
INSERT INTO products (name, price) VALUES 
('iPhone 15', 999.99),
('MacBook Pro', 1999.99),
('iPad Air', 599.99);