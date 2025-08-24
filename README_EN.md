# Pikachun - Real MySQL Binlog Slave Service

[![Go Report Card](https://goreportcard.com/badge/github.com/mcp-zero/pikachun)](https://goreportcard.com/report/github.com/mcp-zero/pikachun)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://github.com/golang/go/releases/tag/go1.24.0)
[![Build Status](https://github.com/mcp-zero/pikachun/workflows/Go/badge.svg)](https://github.com/mcp-zero/pikachun/actions)

Pikachun is a MySQL Binlog slave service written in pure Go language. By subscribing to MySQL's Binlog (binary log), it receives and parses the Binlog stream in real-time to capture database change events.

## 🌟 Key Features

- **🚀 High Performance**: Developed in Go language for excellent performance
- **🔌 Real Binlog Parsing**: Uses `github.com/go-mysql-org/go-mysql` for authentic binlog parsing
- **🔄 Breakpoint Resume**: Supports binlog position persistence and breakpoint resume
- **🌐 Web Management Interface**: Provides an intuitive Web UI for management and monitoring
- **📡 Webhook Callback**: Supports event callback notifications
- **🔧 Flexible Configuration**: Supports advanced configurations like table filtering and event type filtering
- **📦 Easy Deployment**: Supports Docker deployment and binary deployment

## 📚 Documentation

- [Quick Start Guide](QUICK_START_GUIDE.md) - Quick start guide for beginners
- [Detailed Documentation](docs/en/) - Complete features and configuration instructions

## � Demo

![Web Management Interface](docs/pikakun.png)

## 🚀 Quick Start

### 1. One-Click Startup for Beginners (Recommended)

```bash
# Clone the project
git clone https://github.com/mcp-zero/pikachun.git
cd pikachun

# One-click start all services (including MySQL and Webhook test receiver)
./quick-start.sh
```

Access the Web management interface: http://localhost:8668

### 2. Traditional Startup

```bash
# Clone the project
git clone https://github.com/mcp-zero/pikachun.git
cd pikachun

# Use Docker to quickly set up MySQL environment (optional)
./setup_mysql_docker.sh

# Start the service
./start.sh
```

Access the Web management interface: http://localhost:8668

### 2. Quick Experience with Data Change Monitoring

After one-click startup, you can quickly create test data using the following commands:

```bash
# Enter the MySQL container
docker exec -it pikachun-mysql mysql -u root -ppikachun123

# Execute the test data script in MySQL
source /app/test-data.sql
```

Or directly view the real-time event stream in the Web management interface, then execute the following operations in MySQL:

```sql
USE testdb;

-- Insert data
INSERT INTO users (name, email) VALUES ('Test User', 'test@example.com');

-- Update data
UPDATE users SET name = 'Updated User' WHERE email = 'test@example.com';

-- Delete data
DELETE FROM users WHERE email = 'test@example.com';
```

### 3. Configuration Instructions

Create a `config.yaml` file:

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

## 🛠️ Installation and Running

### Requirements

- Go 1.24+
- MySQL 5.7+ or MySQL 8.0+
- MySQL instance with binlog enabled

### Compile and Run

```bash
# Clone the project
git clone https://github.com/mcp-zero/pikachun.git
cd pikachun

# Install dependencies
go mod tidy

# Compile (handle CGO compilation issues)
CGO_CFLAGS="-Wno-nullability-completeness" go build -o pikachun .

# Run the service
./pikachun
```

## 🐳 Docker Deployment

```bash
# Build Docker image
docker build -t pikachun .

# Run container
docker run -d \
  --name pikachun \
  -p 8668:8668 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/app/logs \
  pikachun
```

## 🧪 Testing

### Run Tests

```bash
# Run unit tests
go test ./test/unit/... -v

# Run integration tests
cd test/binlog_test
go run main.go
```

### Test Data

```sql
CREATE DATABASE IF NOT EXISTS test;
USE test;

CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100),
    email VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Execute some operations to test binlog capture
INSERT INTO users (name, email) VALUES ('John Doe', 'johndoe@example.com');
UPDATE users SET email = 'johndoe_new@example.com' WHERE id = 1;
DELETE FROM users WHERE id = 1;
```

## 📊 API Endpoints

### RESTful API

- `GET /api/status` - Get service status
- `GET /api/tasks` - Get all listening tasks
- `POST /api/tasks` - Create a new listening task
- `DELETE /api/tasks/{id}` - Delete a listening task
- `GET /api/events` - Get recent event logs

### WebSocket Interface

- `ws://localhost:8668/ws/events` - Real-time event push

## 📖 Documentation

- [MySQL Configuration Guide](setup_mysql_en.md)
- [Troubleshooting](docs/troubleshooting.md)

## 🤝 Contributing

Issues and Pull Requests are welcome!

1. Fork the project
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgements

- [go-mysql](https://github.com/go-mysql-org/go-mysql) - Go implementation of MySQL protocol
- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [GORM](https://gorm.io/) - ORM library

---

**Pikachun** - Making MySQL Binlog monitoring simple! 🚀