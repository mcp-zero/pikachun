# 配置说明

## 概述

Pikachu'n 可以通过 YAML 配置文件和环境变量进行配置。配置文件通常命名为 `config.yaml`，应放置在可执行文件所在的目录中。

## 配置文件

配置文件使用 YAML 格式。以下是一个完整的示例：

```yaml
mysql:
  host: localhost
  port: 3306
  user: pikachu
  password: password
  database: testdb
  server_id: 1001
  flavor: mysql
  heartbeat_period: 60
  read_timeout: 300

webhook:
  url: https://example.com/webhook
  timeout: 30
  retry_count: 3
  retry_delay: 5

server:
  port: 8668
  read_timeout: 30
  write_timeout: 30

log:
  level: info
  format: json
```

## 配置选项

### MySQL 配置

| 选项 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `mysql.host` | string | 是 | - | MySQL 服务器主机 |
| `mysql.port` | int | 是 | - | MySQL 服务器端口 |
| `mysql.user` | string | 是 | - | MySQL 用户名 |
| `mysql.password` | string | 是 | - | MySQL 密码 |
| `mysql.database` | string | 是 | - | 要监控的数据库名 |
| `mysql.server_id` | int | 否 | 1001 | binlog 复制的服务器 ID |
| `mysql.flavor` | string | 否 | mysql | 数据库类型（mysql 或 mariadb） |
| `mysql.heartbeat_period` | int | 否 | 60 | 心跳周期（秒） |
| `mysql.read_timeout` | int | 否 | 300 | 读取超时（秒） |

### Webhook 配置

| 选项 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `webhook.url` | string | 是 | - | 发送事件的 Webhook URL |
| `webhook.timeout` | int | 否 | 30 | HTTP 请求超时（秒） |
| `webhook.retry_count` | int | 否 | 3 | 重试次数 |
| `webhook.retry_delay` | int | 否 | 5 | 重试间隔（秒） |

### 服务器配置

| 选项 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `server.port` | int | 否 | 8668 | Web 服务器端口 |
| `server.read_timeout` | int | 否 | 30 | HTTP 读取超时（秒） |
| `server.write_timeout` | int | 否 | 30 | HTTP 写入超时（秒） |

### 日志配置

| 选项 | 类型 | 必需 | 默认值 | 描述 |
|------|------|------|--------|------|
| `log.level` | string | 否 | info | 日志级别（debug, info, warn, error） |
| `log.format` | string | 否 | json | 日志格式（json 或 text） |

## 环境变量

所有配置选项都可以通过环境变量覆盖。环境变量名称通过在配置路径前加上 `PIKACHUN_` 前缀并转换为大写和下划线来派生。

例如：
- `mysql.host` 变为 `PIKACHUN_MYSQL_HOST`
- `webhook.timeout` 变为 `PIKACHUN_WEBHOOK_TIMEOUT`

## 配置优先级

配置值按以下顺序应用（后面的源会覆盖前面的源）：

1. 默认值
2. 配置文件（`config.yaml`）
3. 环境变量

## 示例配置

### 基本配置

```yaml
mysql:
  host: localhost
  port: 3306
  user: pikachu
  password: password
  database: testdb

webhook:
  url: https://example.com/webhook
```

### 高级配置与环境变量

```yaml
mysql:
  host: localhost
  port: 3306
  user: pikachu
  password: password
  database: testdb
  server_id: 1001
  heartbeat_period: 30

webhook:
  url: https://example.com/webhook
  timeout: 60
  retry_count: 5
  retry_delay: 10

server:
  port: 9000

log:
  level: debug
```

配合环境变量：
```bash
export PIKACHUN_MYSQL_PASSWORD=secure_password
export PIKACHUN_WEBHOOK_URL=https://secure.example.com/webhook
```

## 验证

应用程序在启动时验证配置，如果必需字段缺失或值无效，将退出并显示错误。

## 重新加载配置

可以通过向 `/reload` 端点发送 POST 请求或向进程发送 SIGHUP 信号在运行时重新加载配置。