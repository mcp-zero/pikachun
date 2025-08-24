# Development

## Overview

This document provides information for developers who want to contribute to Pikachu'n or modify it for their specific needs.

## Project Structure

```
.
├── cmd/
│   └── pikachu-n/
│       └── main.go          # Application entry point
├── configs/
│   └── config.yaml          # Configuration file
├── data/                    # Data directory for checkpoints
├── docs/                    # Documentation
├── internal/
│   ├── config/              # Configuration management
│   ├── server/              # Web server implementation
│   ├── service/             # Core business logic
│   └── web/                 # Web interface files
├── pkg/
│   └── mysql/              # MySQL binlog parsing utilities
├── scripts/
│   └── build.sh            # Build scripts
├── static/                 # Static web assets
├── tests/                  # Test files
├── web/                    # Web interface source files
├── .gitignore              # Git ignore rules
├── Dockerfile              # Docker configuration
├── LICENSE                 # License information
├── README.md               # Project README
└── go.mod                  # Go module definition
```

## Getting Started

### Prerequisites

- Go 1.16 or higher
- Docker (for containerization)
- Node.js and npm (for web interface development)
- MySQL 5.7 or higher

### Setting up the Development Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/pikachu-n.git
   cd pikachu-n
   ```

2. Install Go dependencies:
   ```bash
   go mod tidy
   ```

3. Set up MySQL for development:
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

4. Create a development configuration file (`config.dev.yaml`):
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

## Building the Project

### Building the Go Binary

```bash
go build -o pikachu-n ./cmd/pikachu-n
```

### Building with Docker

```bash
docker build -t pikachu-n .
```

## Running Tests

### Unit Tests

Run all unit tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

Run tests with coverage and generate an HTML report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require a running MySQL instance:
```bash
# Start MySQL for testing
docker run -d \
  --name mysql-test \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=testdb \
  -e MYSQL_USER=pikachu \
  -e MYSQL_PASSWORD=password \
  -p 3307:3306 \
  mysql:8.0

# Run integration tests
PIKACHUN_MYSQL_PORT=3307 go test -tags=integration ./...
```

## Code Structure

### Main Application (`cmd/pikachu-n/main.go`)

The main entry point initializes configuration, sets up the service, and starts the web server.

### Configuration (`internal/config/`)

Handles configuration loading from YAML files and environment variables using Viper.

### Core Service (`internal/service/`)

Contains the main business logic for:
- MySQL binlog monitoring
- Event processing
- Webhook dispatching
- State management

### Web Server (`internal/server/`)

Implements the HTTP server using Gin framework with endpoints for:
- Status monitoring
- Event streaming
- Configuration management

### Web Interface (`internal/web/`)

Contains the web interface files and templates.

## Contributing

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions small and focused

### Git Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin feature/your-feature`)
5. Create a new Pull Request

### Commit Messages

Follow conventional commit format:
- `feat: Add new feature`
- `fix: Fix bug in event processing`
- `docs: Update documentation`
- `test: Add unit tests for webhook dispatcher`
- `refactor: Improve configuration management`

## Web Interface Development

The web interface is built with HTML, CSS, and JavaScript. The source files are in the `web/` directory and are embedded into the binary during build.

### Building Web Assets

During development, you can serve web assets directly from the file system:
```bash
cd web
python3 -m http.server 8000
```

### Modifying the Web Interface

1. Edit files in the `web/` directory
2. Test changes locally
3. Rebuild the Go binary to embed the changes

## Adding New Features

### Adding a New Configuration Option

1. Add the option to the `Config` struct in `internal/config/config.go`
2. Add default values in `internal/config/config.go`
3. Update the configuration documentation in `docs/configuration.md`
4. Add validation if necessary

### Adding a New API Endpoint

1. Add the route in `internal/server/server.go`
2. Implement the handler function
3. Add appropriate error handling
4. Update the API documentation in `docs/api.md`

### Adding a New Event Type

1. Modify the event processing logic in `internal/service/service.go`
2. Update the event structure if necessary
3. Ensure webhook payload format remains compatible
4. Add tests for the new event type

## Debugging

### Logging

The application uses structured logging. Enable debug logging by setting `log.level` to `debug` in the configuration.

### Profiling

Add profiling endpoints for performance analysis:
```go
import _ "net/http/pprof"

// In your main function
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Access profiling data at `http://localhost:6060/debug/pprof/`.

## Release Process

1. Update version in `README.md` and other documentation
2. Create a git tag (`git tag v1.0.0`)
3. Push the tag (`git push origin v1.0.0`)
4. Create a GitHub release with binaries
5. Update Docker images on Docker Hub

## Code Generation

Some parts of the codebase are generated:
- Embedded web assets
- Protocol buffers (if used)
- Mock implementations for testing

Regenerate code with:
```bash
go generate ./...
```

## Dependency Management

Dependencies are managed with Go modules. To add a new dependency:
```bash
go get github.com/some/package
```

To update dependencies:
```bash
go get -u ./...
go mod tidy
```

## Documentation

Documentation is maintained in the `docs/` directory. When adding new features:
1. Update relevant documentation files
2. Add new documentation files if necessary
3. Ensure all documentation is accurate and up-to-date