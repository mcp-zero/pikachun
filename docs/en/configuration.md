# Configuration

## Overview

Pikachu'n can be configured through a YAML configuration file and environment variables. The configuration file is typically named `config.yaml` and should be placed in the same directory as the executable.

## Configuration File

The configuration file uses YAML format. Here's a complete example:

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

## Configuration Options

### MySQL Configuration

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `mysql.host` | string | Yes | - | MySQL server host |
| `mysql.port` | int | Yes | - | MySQL server port |
| `mysql.user` | string | Yes | - | MySQL username |
| `mysql.password` | string | Yes | - | MySQL password |
| `mysql.database` | string | Yes | - | Database name to monitor |
| `mysql.server_id` | int | No | 1001 | Server ID for binlog replication |
| `mysql.flavor` | string | No | mysql | Database flavor (mysql or mariadb) |
| `mysql.heartbeat_period` | int | No | 60 | Heartbeat period in seconds |
| `mysql.read_timeout` | int | No | 300 | Read timeout in seconds |

### Webhook Configuration

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `webhook.url` | string | Yes | - | Webhook URL to send events |
| `webhook.timeout` | int | No | 30 | HTTP request timeout in seconds |
| `webhook.retry_count` | int | No | 3 | Number of retry attempts |
| `webhook.retry_delay` | int | No | 5 | Delay between retries in seconds |

### Server Configuration

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `server.port` | int | No | 8668 | Web server port |
| `server.read_timeout` | int | No | 30 | HTTP read timeout in seconds |
| `server.write_timeout` | int | No | 30 | HTTP write timeout in seconds |

### Log Configuration

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `log.level` | string | No | info | Log level (debug, info, warn, error) |
| `log.format` | string | No | json | Log format (json or text) |

## Environment Variables

All configuration options can be overridden using environment variables. The environment variable names are derived by prefixing `PIKACHUN_` to the configuration path and converting to uppercase with underscores.

For example:
- `mysql.host` becomes `PIKACHUN_MYSQL_HOST`
- `webhook.timeout` becomes `PIKACHUN_WEBHOOK_TIMEOUT`

## Configuration Precedence

Configuration values are applied in the following order (later sources override earlier ones):

1. Default values
2. Configuration file (`config.yaml`)
3. Environment variables

## Example Configurations

### Basic Configuration

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

### Advanced Configuration with Environment Variables

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
  port: 8668

log:
  level: debug
```

With environment variables:
```bash
export PIKACHUN_MYSQL_PASSWORD=secure_password
export PIKACHUN_WEBHOOK_URL=https://secure.example.com/webhook
```

## Validation

The application validates the configuration at startup and will exit with an error if required fields are missing or have invalid values.

## Reloading Configuration

The configuration can be reloaded at runtime by sending a POST request to the `/reload` endpoint or by sending a SIGHUP signal to the process.