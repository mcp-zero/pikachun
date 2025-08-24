# Troubleshooting

## Overview

This document provides solutions to common issues you might encounter when using Pikachu'n.

## Common Issues and Solutions

### 1. Connection Issues

#### Problem: Cannot connect to MySQL
**Error Message**: `Error 1045: Access denied for user 'pikachu'@'localhost'`

**Solution**:
1. Verify MySQL credentials in the configuration file
2. Ensure the user exists and has the correct permissions:
   ```sql
   CREATE USER 'pikachu'@'%' IDENTIFIED BY 'password';
   GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'pikachu'@'%';
   GRANT SELECT ON your_database.* TO 'pikachu'@'%';
   FLUSH PRIVILEGES;
   ```
3. Check if MySQL is running and accessible:
   ```bash
   mysql -h localhost -P 3306 -u pikachu -p
   ```

#### Problem: Binlog not enabled
**Error Message**: `ERROR 1227 (42000): Access denied; you need (at least one of) the SUPER privilege(s) for this operation`

**Solution**:
1. Enable binlog in MySQL configuration (`my.cnf`):
   ```ini
   [mysqld]
   log-bin=mysql-bin
   binlog-format=ROW
   server-id=1
   ```
2. Restart MySQL service
3. Verify binlog is enabled:
   ```sql
   SHOW VARIABLES LIKE 'log_bin';
   ```

### 2. Webhook Issues

#### Problem: Webhook events not being sent
**Symptoms**: Events are processed but not received by the webhook endpoint

**Solution**:
1. Check the webhook URL in the configuration
2. Verify the webhook endpoint is accessible:
   ```bash
   curl -X POST -d '{}' https://your-webhook-url.com
   ```
3. Check application logs for error messages
4. Ensure the webhook endpoint returns HTTP 200 OK

#### Problem: Webhook timeout
**Error Message**: `Post "https://your-webhook-url.com": context deadline exceeded`

**Solution**:
1. Increase the webhook timeout in the configuration:
   ```yaml
   webhook:
     timeout: 60  # Increase from default 30 seconds
   ```
2. Optimize the webhook endpoint to respond faster
3. Check network connectivity between Pikachu'n and the webhook server

### 3. Performance Issues

#### Problem: High memory usage
**Symptoms**: Application consumes increasing amounts of memory over time

**Solution**:
1. Check for memory leaks in custom code
2. Monitor the number of concurrent HTTP connections
3. Adjust the retry configuration to prevent connection buildup:
   ```yaml
   webhook:
     retry_count: 3
     retry_delay: 5
   ```
4. Consider implementing rate limiting for event processing

#### Problem: Slow event processing
**Symptoms**: Delay between MySQL events and webhook delivery

**Solution**:
1. Check MySQL binlog reading performance
2. Optimize the webhook endpoint to respond quickly
3. Consider horizontal scaling with multiple instances
4. Monitor system resources (CPU, memory, disk I/O)

### 4. Configuration Issues

#### Problem: Configuration not loading
**Error Message**: `Failed to load configuration: Config File "config" Not Found in "[/path/to/config]"`

**Solution**:
1. Verify the configuration file exists in the correct location
2. Check file permissions:
   ```bash
   ls -la config.yaml
   ```
3. Ensure the configuration file is valid YAML:
   ```bash
   yamllint config.yaml
   ```
4. Use environment variables as an alternative configuration method

#### Problem: Environment variables not taking effect
**Symptoms**: Configuration values from environment variables are ignored

**Solution**:
1. Verify environment variable names follow the correct pattern:
   - `mysql.host` becomes `PIKACHUN_MYSQL_HOST`
   - `webhook.timeout` becomes `PIKACHUN_WEBHOOK_TIMEOUT`
2. Check for typos in environment variable names
3. Ensure environment variables are exported:
   ```bash
   export PIKACHUN_MYSQL_PASSWORD=your_password
   ```

### 5. Deployment Issues

#### Problem: Docker container fails to start
**Error Message**: `standard_init_linux.go:211: exec user process caused "no such file or directory"`

**Solution**:
1. Ensure the binary is compiled for the correct architecture:
   ```bash
   CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pikachu-n .
   ```
2. Check file permissions in the Docker image:
   ```dockerfile
   RUN chmod +x /app/pikachu-n
   ```
3. Verify the entry point in the Dockerfile

#### Problem: Port conflicts
**Error Message**: `listen tcp :8668: bind: address already in use`

**Solution**:
1. Change the server port in the configuration:
   ```yaml
   server:
     port: 8081
   ```
2. Or stop the conflicting service:
   ```bash
   sudo lsof -i :8668
   sudo kill -9 <PID>
   ```

### 6. Data Issues

#### Problem: Missing or incorrect data in events
**Symptoms**: Event data doesn't match the actual database changes

**Solution**:
1. Verify MySQL binlog format is set to ROW:
   ```sql
   SHOW VARIABLES LIKE 'binlog_format';
   ```
2. Check for data type handling issues in the application logs
3. Ensure the MySQL user has SELECT permissions on the relevant tables

#### Problem: Duplicate events
**Symptoms**: Same events are sent multiple times

**Solution**:
1. Check the checkpoint mechanism:
   - Verify the `data/` directory is writable
   - Check for file permission issues
2. Ensure only one instance is running per configuration
3. Review the retry configuration to prevent unnecessary retries

## Debugging Techniques

### 1. Enable Debug Logging

Set the log level to debug in the configuration:
```yaml
log:
  level: debug
```

Or use an environment variable:
```bash
export PIKACHUN_LOG_LEVEL=debug
```

### 2. Monitor Internal Metrics

Check the `/status` endpoint for internal metrics:
```bash
curl http://localhost:8668/status
```

### 3. Use the Test Webhook Endpoint

For development, you can use a simple test webhook receiver:
```bash
# Using Node.js
npx http-server -p 3000

# Using Python
python3 -m http.server 3000

# Using netcat
nc -l 3000
```

### 4. Analyze Binlog Events

Directly inspect MySQL binlog events:
```bash
mysqlbinlog --read-from-remote-server \
  --host=localhost --port=3306 \
  --user=pikachu --password=password \
  --raw binlog-name
```

## Log Analysis

### Common Log Patterns

1. **Successful Event Processing**:
   ```
   INFO[0001] Event processed successfully  action=INSERT database=testdb table=users
   ```

2. **Webhook Delivery**:
   ```
   INFO[0002] Webhook sent successfully     status=200 url=https://example.com/webhook
   ```

3. **Retry Attempts**:
   ```
   WARN[0003] Webhook delivery failed, retrying  attempt=1 error="Post ...: dial tcp: lookup example.com: no such host"
   ```

4. **Configuration Reload**:
   ```
   INFO[0004] Configuration reloaded        source=file
   ```

### Error Log Analysis

1. **Connection Errors**:
   ```
   ERRO[0005] Failed to connect to MySQL    error="dial tcp 127.0.0.1:3306: connect: connection refused"
   ```
   Solution: Check MySQL service status and network connectivity.

2. **Permission Errors**:
   ```
   ERRO[0006] Access denied                 error="Error 1045: Access denied for user 'pikachu'@'localhost'"
   ```
   Solution: Verify MySQL user credentials and permissions.

3. **Data Parsing Errors**:
   ```
   ERRO[0007] Failed to parse binlog event  error="unknown column type"
   ```
   Solution: Check for unsupported data types or update the application to handle new types.

## Performance Monitoring

### Key Metrics to Monitor

1. **Event Processing Rate**: Number of events processed per second
2. **Webhook Delivery Latency**: Time taken to deliver events to webhooks
3. **Memory Usage**: Application memory consumption over time
4. **Error Rate**: Percentage of events that result in errors

### Monitoring Tools

1. **Prometheus Metrics**: If exposed, scrape metrics from `/metrics` endpoint
2. **Log Aggregation**: Use tools like ELK stack to analyze logs
3. **Process Monitoring**: Use system tools like `top`, `htop`, or `ps` to monitor resource usage

## Recovery Procedures

### 1. Recovering from Checkpoint Corruption

If the checkpoint file becomes corrupted:
1. Stop the application
2. Backup the current checkpoint file:
   ```bash
   cp data/checkpoint.json data/checkpoint.json.backup
   ```
3. Either:
   - Delete the checkpoint file to start from the current position
   - Restore from the backup if it's valid
4. Restart the application

### 2. Recovering from MySQL Failures

If MySQL becomes unavailable:
1. The application will retry connections based on configuration
2. Check MySQL logs for the root cause
3. Once MySQL is restored, the application should automatically reconnect
4. Verify events are being processed correctly after recovery

### 3. Rolling Back Configuration Changes

If a configuration change causes issues:
1. Use the `/reload` endpoint with a previous configuration:
   ```bash
   curl -X POST http://localhost:8668/reload
   ```
2. Or restart the application with the previous configuration file
3. Or use environment variables to override problematic settings

## Contact Support

If you're unable to resolve an issue with the information in this document:

1. Check the GitHub issues for similar problems
2. Create a new issue with:
   - Detailed description of the problem
   - Relevant log output (with sensitive information redacted)
   - Configuration details (with sensitive information redacted)
   - Steps to reproduce the issue
3. Include the version of Pikachu'n you're using
4. Include your environment details (OS, MySQL version, etc.)