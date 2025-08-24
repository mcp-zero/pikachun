#!/bin/bash

# Pikachu'n 一键启动脚本
# 适用于小白用户的快速体验

# 检查是否为开发模式
DEV_MODE=false
if [[ "$1" == "-d" || "$1" == "--dev" ]]; then
    DEV_MODE=true
fi

echo "🚀 Pikachu'n 一键启动脚本"
echo "========================"

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null
then
    echo "❌ 未检测到 Docker，请先安装 Docker"
    echo "请访问 https://docs.docker.com/get-docker/ 获取安装指南"
    exit 1
fi

echo "✅ Docker 已安装"

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null
then
    echo "❌ 未检测到 Docker Compose，请先安装 Docker Compose"
    echo "请访问 https://docs.docker.com/compose/install/ 获取安装指南"
    exit 1
fi

echo "✅ Docker Compose 已安装"

if [ "$DEV_MODE" = true ]; then
    echo "🔧 开发模式：清理Docker缓存并重新构建..."
    
    # 停止并删除现有容器
    echo "🛑 停止并删除现有容器..."
    docker-compose down 2>/dev/null || true
    
    # 删除旧镜像
    echo "🗑️ 删除旧镜像..."
    docker rmi pikachun-pikachun 2>/dev/null || true
    
    # 清理Docker构建缓存
    echo "🧹 清理Docker构建缓存..."
    docker builder prune -a -f
    
    # 重新构建并启动服务
    echo "🚀 重新构建并启动服务..."
    docker-compose up -d --build --force-recreate
else
    # 构建并启动服务
    echo "� 构建并启动服务..."
    docker-compose up -d
fi

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 15

# 将测试数据文件复制到 MySQL 容器
echo "📋 将测试数据文件复制到 MySQL 容器..."
docker cp test-data.sql pikachun-mysql:/app/test-data.sql

# 检查服务状态
echo "🔍 检查服务状态..."
if docker-compose ps | grep -q "Up"; then
    echo "✅ 服务启动成功！"
    
    echo "🌐 访问 Pikachu'n 管理界面：http://localhost:8668"
    echo "📊 MySQL 管理界面（可选）：http://localhost:3306"
    echo "📡 Webhook 测试接收器：http://localhost:9669"
    
    echo ""
    echo "📝 下一步操作："
    echo "1. 打开浏览器访问 http://localhost:8668"
    echo "2. 在 MySQL 中创建表并插入数据以测试 binlog 监听"
    echo "   执行以下命令快速体验："
    echo "   docker exec -it pikachun-mysql mysql -u root -ppikachun123"
    echo "   source /app/test-data.sql"
    echo "3. 查看 Webhook 接收器以验证事件是否正确发送"
    echo ""
    echo "🐳 相关命令："
    echo "   查看日志: docker-compose logs -f"
    echo "   停止服务: docker-compose down"
    echo "   重启服务: docker-compose restart"
    
else
    echo "❌ 服务启动失败，请检查日志"
    docker-compose logs
fi