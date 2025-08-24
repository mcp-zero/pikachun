# Pikachun - 真实 MySQL Binlog 从库服务

[![Go Report Card](https://goreportcard.com/badge/github.com/mcp-zero/pikachun)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://github.com/golang/go/releases/tag/go1.24.0)
[![Build Status](https://github.com/mcp-zero/pikachun/workflows/Go/badge.svg)](https://github.com/mcp-zero/pikachun/actions)

Pikachun 是一个用纯 Go 语言编写的 MySQL Binlog 从库服务，通过订阅 MySQL 的 Binlog（二进制日志），实时接收并解析 Binlog 流，捕获数据库的变更事件。

## 🌟 特性亮点

- **🚀 高性能**: 基于 Go 语言开发，性能优异
- **🔌 真实 Binlog 解析**: 使用 `github.com/go-mysql-org/go-mysql` 进行真实的 binlog 解析
- **🔄 断点续传**: 支持 binlog 位置持久化和断点续传
- **🌐 Web 管理界面**: 提供直观的 Web UI 进行管理和监控
- **📡 Webhook 回调**: 支持事件回调通知
- **🔧 灵活配置**: 支持表过滤、事件类型过滤等高级配置
- **📦 易于部署**: 支持 Docker 部署和二进制部署

## 📚 文档

- [快速上手指南](QUICK_START_GUIDE.md) - 小白用户快速体验指南
- [详细文档](docs/zh/) - 完整功能和配置说明

## � 演示

![Web管理界面](docs/pikakun.png)

## 🚀 快速开始

### 1. 小白一键启动（推荐）

```bash
# 克隆项目
git clone https://github.com/lucklidi/pikachun.git
cd pikachun

# 一键启动所有服务（包括 MySQL 和 Webhook 测试接收器）
./quick-start.sh
```

访问 Web 管理界面：http://localhost:8668

### 2. 传统启动方式

```bash
# 克隆项目
git clone https://github.com/lucklidi/pikachun.git
cd pikachun

# 使用 Docker 快速设置 MySQL 环境（可选）
./setup_mysql_docker.sh

# 启动服务
./start.sh
```

访问 Web 管理界面：http://localhost:8668

### 2. 快速体验数据变更监听

一键启动后，您可以使用以下命令快速创建测试数据：

```bash
# 进入 MySQL 容器
docker exec -it pikachun-mysql mysql -u root -ppikachun123

# 在 MySQL 中执行测试数据脚本
source /app/test-data.sql
```

或者直接在 Web 管理界面中查看实时事件流，然后在 MySQL 中执行以下操作：

```sql
USE testdb;

-- 插入数据
INSERT INTO users (name, email) VALUES ('测试用户', 'test@example.com');

-- 更新数据
UPDATE users SET name = '更新用户' WHERE email = 'test@example.com';

-- 删除数据
DELETE FROM users WHERE email = 'test@example.com';
```

### 3. 配置说明

创建 `config.yaml` 文件：

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
  password: "your_password"
  charset: "utf8mb4"
  server_id: 1001
  
  binlog:
    filename: ""
    position: 4
    gtid_enabled: true
    
  watch:
    databases: []
    tables: []
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

## 🛠️ 安装和运行

### 环境要求

- Go 1.24+
- MySQL 5.7+ 或 MySQL 8.0+
- 启用 binlog 的 MySQL 实例

### 编译和运行

```bash
# 克隆项目
git clone https://github.com/mcp-zero/pikachun.git
cd pikachun

# 安装依赖
go mod tidy

# 编译（处理 CGO 编译问题）
CGO_CFLAGS="-Wno-nullability-completeness" go build -o pikachun .

# 运行服务
./pikachun
```

## 🐳 Docker 部署

```bash
# 构建 Docker 镜像
docker build -t pikachun .

# 运行容器
docker run -d \
  --name pikachun \
  -p 8668:8668 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  pikachun
```

## 📦 GitHub Packages 部署

项目支持将 Docker 镜像推送到 GitHub Packages，方便在 GitHub Actions 或其他 CI/CD 流程中使用。

### 构建和推送镜像

1. 确保你已经安装了 Docker 并正在运行。
2. 创建一个 GitHub Personal Access Token (PAT) 并将其设置为环境变量：
   ```bash
   export GITHUB_TOKEN=your_github_token
   ```
3. 运行构建和推送脚本：
   ```bash
   ./build-and-push.sh
   ```

脚本会自动完成以下操作：
- 构建 Docker 镜像
- 给镜像打上 GitHub Packages 标签
- 登录到 GitHub Container Registry
- 推送镜像到 GitHub Packages

推送完成后，镜像将可以在 `ghcr.io/mcp-zero/pikachun:latest` 访问。

## 🧪 测试

### 运行测试

```bash
# 运行单元测试
go test ./test/unit/... -v

# 运行集成测试
cd test/binlog_test
go run main.go
```

### 测试数据

```sql
CREATE DATABASE IF NOT EXISTS test;
USE test;

CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100),
    email VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 执行一些操作来测试 binlog 捕获
INSERT INTO users (name, email) VALUES ('张三', 'zhangsan@example.com');
UPDATE users SET email = 'zhangsan_new@example.com' WHERE id = 1;
DELETE FROM users WHERE id = 1;
```

## 📊 API 接口

### RESTful API

- `GET /api/status` - 获取服务状态
- `GET /api/tasks` - 获取所有监听任务
- `POST /api/tasks` - 创建新的监听任务
- `DELETE /api/tasks/{id}` - 删除监听任务
- `GET /api/events` - 获取最近的事件日志

### WebSocket 接口

- `ws://localhost:8668/ws/events` - 实时事件推送

## 📖 文档

- [MySQL配置指南](docs/zh/setup_mysql.md)
- [故障排除](docs/zh/TROUBLESHOOTING.md)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证。详情请见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [go-mysql](https://github.com/go-mysql-org/go-mysql) - MySQL 协议的 Go 实现
- [Gin](https://github.com/gin-gonic/gin) - HTTP web 框架
- [GORM](https://gorm.io/) - ORM 库

---

**Pikachun** - 让 MySQL Binlog 监听变得简单！ 🚀

## 🛠️ 开发注意事项

在开发过程中修改代码后，需要确保 Docker 镜像包含最新的代码变更。有以下几种方式：

### 方式一：强制重新构建（推荐）
```bash
# 删除旧镜像并重新构建
docker-compose down
docker rmi pikachun_pikachun  # 删除旧镜像
docker-compose up -d --build  # 重新构建并启动

# 或者使用一行命令强制重新构建
docker-compose up -d --build --force-recreate
```

### 方式二：清理构建缓存
```bash
# 清理 Docker 构建缓存
docker builder prune -a

# 重新构建
docker-compose up -d --build
```

### 方式三：在 Dockerfile 中添加版本标识
在 Dockerfile 中添加一个构建参数来强制重新构建：
```dockerfile
# 添加构建参数
ARG BUILD_VERSION=1
RUN echo "Build version: $BUILD_VERSION"

# 在构建时传递不同的版本号
docker-compose build --build-arg BUILD_VERSION=$(date +%s)
```