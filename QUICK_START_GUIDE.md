# Pikachu'n 快速上手指南

本指南将帮助您快速启动和体验 Pikachu'n 服务，无需任何复杂的配置。

## 🚀 一键启动

### 1. 前提条件

确保您的系统已安装以下软件：
- Docker
- Docker Compose

### 2. 启动服务

```bash
# 克隆项目
git clone https://github.com/lucklidi/pikachun.git
cd pikachun

# 一键启动所有服务
./quick-start.sh
```

### 3. 访问界面

启动成功后，您可以访问以下地址：
- Pikachu'n 管理界面：http://localhost:8668
- Webhook 测试接收器：http://localhost:9669

## 🧪 快速体验

### 1. 查看实时事件流

1. 打开浏览器访问 http://localhost:8668
2. 您将看到实时事件流界面

### 2. 生成测试数据

在终端中执行以下命令：

```bash
# 进入 MySQL 容器
docker exec -it pikachun-mysql mysql -u root -ppikachun123

# 执行测试数据脚本
source /app/test-data.sql

# 退出 MySQL
exit
```

### 3. 观察事件

在 Pikachu'n 管理界面中，您将看到实时的数据库变更事件：
- 用户表的插入、更新、删除操作
- 产品表的插入操作

### 4. 查看 Webhook 接收器

打开新的浏览器标签页访问 http://localhost:9669 webhook 的事件记录。

## 🛠️ 常用操作

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f pikachun
```

### 停止服务

```bash
docker-compose down
```

### 重启服务

```bash
docker-compose restart
```

## 📝 下一步

1. 阅读完整文档了解所有功能
2. 根据需要修改配置文件
3. 将服务集成到您的项目中

## 🆘 常见问题

### 服务启动失败

1. 检查 Docker 是否正在运行
2. 检查端口是否被占用
3. 查看日志获取详细错误信息

### 未收到事件

1. 确认 MySQL 配置正确
2. 检查表是否在监听范围内
3. 查看日志确认连接状态

---

享受使用 Pikachu'n 的体验！如有任何问题，请查看完整文档或提交 issue。