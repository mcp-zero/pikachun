# API 参考

## 概述

Pikachu'n 提供了一个 Web 界面，包含多个端点用于监控和管理服务。

## 端点

### GET `/`

返回带有实时事件监控的主仪表板页面。

**响应：**
- HTML 页面与事件显示界面

### GET `/events`

服务器发送事件（SSE）端点，用于实时事件流。

**响应：**
- JSON 格式事件流
- Content-Type: text/event-stream

**事件示例：**
```json
{
  "id": "12345",
  "database": "testdb",
  "table": "users",
  "action": "INSERT",
  "data": {
    "id": 1,
    "name": "张三",
    "email": "zhangsan@example.com"
  },
  "timestamp": "2023-01-01T12:00:00Z"
}
```

### GET `/status`

返回服务的当前状态。

**响应：**
```json
{
  "status": "running",
  "last_event": "2023-01-01T12:00:00Z",
  "events_processed": 12345,
  "webhook_url": "https://example.com/webhook"
}
```

### POST `/webhook`

binlog 事件作为 webhook 发送的端点。

**请求体：**
```json
{
  "id": "12345",
  "database": "testdb",
  "table": "users",
  "action": "INSERT",
  "data": {
    "id": 1,
    "name": "张三",
    "email": "zhangsan@example.com"
  },
  "timestamp": "2023-01-01T12:00:00Z"
}
```

### GET `/config`

返回当前配置（敏感数据已隐藏）。

**响应：**
```json
{
  "mysql": {
    "host": "localhost",
    "port": 3306,
    "user": "pikachu",
    "password": "***",
    "database": "testdb"
  },
  "webhook": {
    "url": "https://example.com/webhook",
    "timeout": 30
  },
  "server": {
    "port": 8668
  }
}
```

### POST `/reload`

从文件重新加载配置。

**响应：**
```json
{
  "status": "success",
  "message": "配置已重新加载"
}
```

## 事件格式

所有发送到 webhook 的事件都遵循以下格式：

```json
{
  "id": "string",
  "database": "string",
  "table": "string",
  "action": "INSERT|UPDATE|DELETE",
  "data": {
    "field1": "value1",
    "field2": "value2"
  },
  "timestamp": "ISO8601 时间戳"
}
```

## 错误响应

所有 API 端点都可能返回以下错误响应：

### 400 错误请求
```json
{
  "error": "无效的请求参数"
}
```

### 404 未找到
```json
{
  "error": "端点未找到"
}
```

### 500 服务器内部错误
```json
{
  "error": "服务器内部错误"
}
```

## 认证

目前，API 没有实现认证。在生产环境中，建议：
1. 在反向代理后面运行服务并进行认证
2. 使用网络级访问控制
3. 如需要，实现自定义认证中间件