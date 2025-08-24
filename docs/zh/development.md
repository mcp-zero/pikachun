# 开发指南

## 概述

本文档为想要为 Pikachu'n 贡献代码或为其修改以满足特定需求的开发人员提供信息。

## 项目结构

```
.
├── cmd/
│   └── pikachu-n/
│       └── main.go          # 应用程序入口点
├── configs/
│   └── config.yaml          # 配置文件
├── data/                    # 数据目录，用于检查点
├── docs/                    # 文档
├── internal/
│   ├── config/              # 配置管理
│   ├── server/              # Web 服务器实现
│   ├── service/             # 核心业务逻辑
│   └── web/                 # Web 界面文件
├── pkg/
│   └── mysql/              # MySQL binlog 解析工具
├── scripts/
│   └── build.sh            # 构建脚本
├── static/                 # 静态 Web 资源
├── tests/                  # 测试文件
├── web/                    # Web 界面源文件
├── .gitignore              # Git 忽略规则
├── Dockerfile              # Docker 配置
├── LICENSE                 # 许可证信息
├── README.md               # 项目 README
└── go.mod                  # Go 模块定义
```

## 开始开发

### 前提条件

- Go 1.16 或更高版本
- Docker（用于容器化）
- Node.js 和 npm（用于 Web 界面开发）
- MySQL 5.7 或更高版本

### 设置开发环境

1. 克隆仓库：
   ```bash
   git clone https://github.com/your-username/pikachu-n.git
   cd pikachu-n
   ```

2. 安装 Go 依赖：
   ```bash
   go mod tidy
   ```

3. 设置 MySQL 进行开发：
   ```bash
   docker run -d \
     --name mysql-dev \
     -e MYSQL_ROOT_PASSWORD=rootpassword \
     -e MYSQL_DATABASE=testdb \
     -e MYSQL_USER=pikachu \
     -e MYSQL_PASSWORD=password \
     -p 3306:3306 \
     mysql:8.0
   ```

4. 创建开发配置文件（`config.dev.yaml`）：
   ```yaml
   mysql:
     host: localhost
     port: 3306
     user: pikachu
     password: password
     database: testdb

   webhook:
     url: http://localhost:3000/webhook

   log:
     level: debug
   ```

## 构建项目

### 构建 Go 二进制文件

```bash
go build -o pikachu-n ./cmd/pikachu-n
```

### 使用 Docker 构建

```bash
docker build -t pikachu-n .
```

## 运行测试

### 单元测试

运行所有单元测试：
```bash
go test ./...
```

运行带覆盖率的测试：
```bash
go test -cover ./...
```

运行带覆盖率并生成 HTML 报告的测试：
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 集成测试

集成测试需要运行中的 MySQL 实例：
```bash
# 启动用于测试的 MySQL
docker run -d \
  --name mysql-test \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=testdb \
  -e MYSQL_USER=pikachu \
  -e MYSQL_PASSWORD=password \
  -p 3307:3306 \
  mysql:8.0

# 运行集成测试
PIKACHUN_MYSQL_PORT=3307 go test -tags=integration ./...
```

## 代码结构

### 主应用程序（`cmd/pikachu-n/main.go`）

主入口点初始化配置、设置服务并启动 Web 服务器。

### 配置（`internal/config/`）

处理从 YAML 文件和环境变量加载配置，使用 Viper。

### 核心服务（`internal/service/`）

包含主要业务逻辑：
- MySQL binlog 监控
- 事件处理
- Webhook 分发
- 状态管理

### Web 服务器（`internal/server/`）

使用 Gin 框架实现 HTTP 服务器，包含以下端点：
- 状态监控
- 事件流
- 配置管理

### Web 界面（`internal/web/`）

包含 Web 界面文件和模板。

## 贡献

### 代码风格

- 遵循标准 Go 格式化（`go fmt`）
- 使用有意义的变量和函数名
- 为导出的函数和复杂逻辑添加注释
- 保持函数小而专注

### Git 工作流

1. Fork 仓库
2. 创建功能分支（`git checkout -b feature/your-feature`）
3. 提交更改（`git commit -am 'Add some feature'`）
4. 推送到分支（`git push origin feature/your-feature`）
5. 创建新的 Pull Request

### 提交消息

遵循常规提交格式：
- `feat: 添加新功能`
- `fix: 修复事件处理中的错误`
- `docs: 更新文档`
- `test: 为 webhook 分发器添加单元测试`
- `refactor: 改进配置管理`

## Web 界面开发

Web 界面使用 HTML、CSS 和 JavaScript 构建。源文件在 `web/` 目录中，并在构建期间嵌入到二进制文件中。

### 构建 Web 资源

在开发期间，您可以直接从文件系统提供 Web 资源：
```bash
cd web
python3 -m http.server 8000
```

### 修改 Web 界面

1. 编辑 `web/` 目录中的文件
2. 在本地测试更改
3. 重新构建 Go 二进制文件以嵌入更改

## 添加新功能

### 添加新配置选项

1. 在 `internal/config/config.go` 中的 `Config` 结构体中添加选项
2. 在 `internal/config/config.go` 中添加默认值
3. 更新 `docs/configuration.md` 中的配置文档
4. 如有必要，添加验证

### 添加新 API 端点

1. 在 `internal/server/server.go` 中添加路由
2. 实现处理函数
3. 添加适当的错误处理
4. 更新 `docs/api.md` 中的 API 文档

### 添加新事件类型

1. 修改 `internal/service/service.go` 中的事件处理逻辑
2. 如有必要，更新事件结构
3. 确保 webhook 负载格式保持兼容
4. 为新事件类型添加测试

## 调试

### 日志

应用程序使用结构化日志。通过在配置中将 `log.level` 设置为 `debug` 来启用调试日志。

### 性能分析

添加性能分析端点：
```go
import _ "net/http/pprof"

// 在您的主函数中
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

在 `http://localhost:6060/debug/pprof/` 访问性能分析数据。

## 发布流程

1. 更新 `README.md` 和其他文档中的版本
2. 创建 git 标签（`git tag v1.0.0`）
3. 推送标签（`git push origin v1.0.0`）
4. 在 GitHub 上创建带二进制文件的发布
5. 在 Docker Hub 上更新 Docker 镜像

## 代码生成

代码库的某些部分是生成的：
- 嵌入的 Web 资源
- 协议缓冲区（如果使用）
- 用于测试的模拟实现

重新生成代码：
```bash
go generate ./...
```

## 依赖管理

依赖项使用 Go 模块管理。添加新依赖：
```bash
go get github.com/some/package
```

更新依赖：
```bash
go get -u ./...
go mod tidy
```

## 文档

文档在 `docs/` 目录中维护。添加新功能时：
1. 更新相关文档文件
2. 如有必要，添加新文档文件
3. 确保所有文档都是准确和最新的