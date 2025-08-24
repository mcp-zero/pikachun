#!/bin/bash

# Pikachun 启动脚本
# 作者: Pikachun Team
# 版本: 1.0.0

set -e  # 遇到错误时退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的信息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 检查Go环境
check_go_environment() {
    if ! command_exists go; then
        print_error "未找到Go环境，请先安装Go 1.23+"
        exit 1
    fi
    
    GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | cut -d' ' -f1)
    print_info "当前Go版本: $GO_VERSION"
    
    if [[ "$GO_VERSION" < "go1.23" ]]; then
        print_warning "建议使用Go 1.23或更高版本以获得最佳性能"
    fi
}

# 检查配置文件
check_config() {
    if [ ! -f "config.yaml" ]; then
        print_warning "未找到config.yaml配置文件，将使用默认配置"
        print_info "建议创建config.yaml文件以自定义配置"
    else
        print_success "找到配置文件 config.yaml"
    fi
}

# 检查数据目录
check_data_directory() {
    if [ ! -d "data" ]; then
        print_info "创建数据目录..."
        mkdir -p data
        print_success "数据目录创建成功"
    else
        print_success "数据目录已存在"
    fi
}

# 检查日志目录
check_logs_directory() {
    if [ ! -d "logs" ]; then
        print_info "创建日志目录..."
        mkdir -p logs
        print_success "日志目录创建成功"
    else
        print_success "日志目录已存在"
    fi
}

# 编译项目
build_project() {
    print_info "正在编译 Pikachun..."
    
    # 处理 CGO 编译问题
    if CGO_CFLAGS="-Wno-nullability-completeness -Wno-error" go build -o pikachun main.go; then
        print_success "编译成功！"
    else
        print_error "编译失败，请检查错误信息"
        exit 1
    fi
}

# 开发模式：清理Docker缓存并重新构建
dev_mode() {
    print_info "进入开发模式..."
    
    # 检查Docker环境
    if ! command_exists docker; then
        print_error "未找到Docker，请先安装Docker"
        exit 1
    fi
    
    if ! command_exists docker-compose; then
        print_error "未找到docker-compose，请先安装docker-compose"
        exit 1
    fi
    
    print_info "停止并删除现有容器..."
    docker-compose down 2>/dev/null || true
    
    print_info "删除旧镜像..."
    docker rmi pikachun-pikachun 2>/dev/null || true
    
    print_info "清理Docker构建缓存..."
    docker builder prune -a -f
    
    print_info "重新构建并启动服务..."
    if docker-compose up -d --build --force-recreate; then
        print_success "开发环境启动成功！"
        print_info "Web管理界面: http://localhost:8668"
        print_info "Webhook测试接收器: http://localhost:9669"
    else
        print_error "开发环境启动失败，请检查错误信息"
        exit 1
    fi
}

# 启动服务
start_service() {
    print_info "启动 Pikachun 服务..."
    print_info "Web管理界面: http://localhost:8668"
    print_info "按 Ctrl+C 停止服务"
    print_info "================================"
    
    # 启动服务
    ./pikachun
}

# 显示帮助信息
show_help() {
    echo "Pikachun 启动脚本"
    echo ""
    echo "用法:"
    echo "  ./start.sh [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示帮助信息"
    echo "  -b, --build    仅编译项目"
    echo "  -c, --check    检查环境配置"
    echo "  -d, --dev      开发模式（清理Docker缓存并重新构建）"
    echo ""
    echo "示例:"
    echo "  ./start.sh          # 编译并启动服务"
    echo "  ./start.sh --build  # 仅编译项目"
    echo "  ./start.sh --check  # 检查环境配置"
    echo "  ./start.sh --dev    # 开发模式，清理Docker缓存并重新构建"
}

# 主函数
main() {
    # 解析命令行参数
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        -b|--build)
            check_go_environment
            check_config
            check_data_directory
            check_logs_directory
            build_project
            print_success "项目编译完成"
            exit 0
            ;;
        -c|--check)
            check_go_environment
            check_config
            check_data_directory
            check_logs_directory
            print_success "环境检查完成"
            exit 0
            ;;
        "")
            # 默认行为：检查环境并启动服务
            check_go_environment
            check_config
            check_data_directory
            check_logs_directory
            build_project
            start_service
            ;;
        *)
            print_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"