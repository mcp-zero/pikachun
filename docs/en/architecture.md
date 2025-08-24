# Architecture

## Overview

Pikachu'n is a Go-based service that monitors MySQL binlog events and forwards them as JSON payloads to configured webhook URLs. The system is designed to be lightweight, efficient, and reliable.

## Components

### 1. Configuration Manager
- Uses Viper for configuration management
- Supports multiple configuration sources (YAML, environment variables)
- Handles configuration validation and default values

### 2. MySQL Binlog Reader
- Connects to MySQL server using go-mysql library
- Parses binlog events (INSERT, UPDATE, DELETE)
- Handles different data types and special cases (e.g., JSON, BIT)

### 3. Event Processor
- Processes raw binlog events
- Transforms data into structured JSON format
- Handles event filtering based on configuration

### 4. Webhook Dispatcher
- Sends events to configured webhook URLs
- Implements retry mechanism with exponential backoff
- Handles HTTP response validation

### 5. State Manager
- Manages checkpoint persistence
- Implements break-point resume functionality
- Handles state synchronization

### 6. Web Interface
- Built with Gin framework
- Provides monitoring and management endpoints
- Real-time event display

## Data Flow

```
MySQL Binlog → Binlog Reader → Event Processor → Webhook Dispatcher → Webhook URLs
                                          ↓
                                    State Manager
                                          ↓
                                   Web Interface
```

## Design Decisions

### 1. Language Choice
Go was chosen for its:
- Excellent concurrency support
- Strong ecosystem for system programming
- Good performance characteristics
- Simple deployment (single binary)

### 2. Binlog Parsing
The go-mysql library was selected because:
- Purpose-built for MySQL binlog parsing
- Active maintenance and community support
- Good performance and reliability

### 3. Web Framework
Gin was chosen for:
- High performance
- Simple API
- Good middleware support
- Active community

### 4. Configuration Management
Viper was selected for:
- Multiple configuration format support
- Environment variable integration
- Configuration hot-reloading capabilities

## Scalability Considerations

### Horizontal Scaling
- Multiple instances can run with different table filters
- Shared state through external storage (future enhancement)

### Performance Optimization
- Efficient event processing pipeline
- Connection pooling for MySQL and HTTP clients
- Asynchronous event dispatching

## Security Considerations

### Data Protection
- Webhook URLs should use HTTPS
- Sensitive configuration can be passed via environment variables
- Event data is not stored persistently

### Access Control
- Web interface should be protected in production
- Configuration files should have appropriate permissions