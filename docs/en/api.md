# API Reference

## Overview

Pikachu'n provides a web interface with several endpoints for monitoring and managing the service.

## Endpoints

### GET `/`

Returns the main dashboard page with real-time event monitoring.

**Response:**
- HTML page with event display interface

### GET `/events`

Server-Sent Events endpoint for real-time event streaming.

**Response:**
- Stream of JSON-formatted events
- Content-Type: text/event-stream

**Example Event:**
```json
{
  "id": "12345",
  "database": "testdb",
  "table": "users",
  "action": "INSERT",
  "data": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com"
  },
  "timestamp": "2023-01-01T12:00:00Z"
}
```

### GET `/status`

Returns the current status of the service.

**Response:**
```json
{
  "status": "running",
  "last_event": "2023-01-01T12:00:00Z",
  "events_processed": 12345,
  "webhook_url": "https://example.com/webhook"
}
```

### POST `/webhook`

The endpoint where binlog events are sent as webhooks.

**Request Body:**
```json
{
  "id": "12345",
  "database": "testdb",
  "table": "users",
  "action": "INSERT",
  "data": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com"
  },
  "timestamp": "2023-01-01T12:00:00Z"
}
```

### GET `/config`

Returns the current configuration (sensitive data redacted).

**Response:**
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
    "port": 9669
  }
}
```

### POST `/reload`

Reloads the configuration from file.

**Response:**
```json
{
  "status": "success",
  "message": "Configuration reloaded"
}
```

## Event Format

All events sent to webhooks follow this format:

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
  "timestamp": "ISO8601 timestamp"
}
```

## Error Responses

All API endpoints may return the following error responses:

### 400 Bad Request
```json
{
  "error": "Invalid request parameters"
}
```

### 404 Not Found
```json
{
  "error": "Endpoint not found"
}
```

### 500 Internal Server Error
```json
{
  "error": "Internal server error"
}
```

## Authentication

Currently, the API does not implement authentication. In production environments, it's recommended to:
1. Run the service behind a reverse proxy with authentication
2. Use network-level access controls
3. Implement custom authentication middleware if needed