package canal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"pikachun/internal/config"
	"pikachun/internal/database"
)

// MySQLCanalInstance 基于真实 MySQL binlog 的 Canal 实例实现
type MySQLCanalInstance struct {
	id          string
	config      MySQLConfig
	eventSink   *DefaultEventSink
	binlogSlave BinlogSlave
	logger      *log.Logger
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	status      InstanceStatus
}

// NewMySQLCanalInstance 创建基于真实 MySQL binlog 的 Canal 实例
func NewMySQLCanalInstance(id string, cfg *config.Config, logger *log.Logger, metaManager MetaManager) (*MySQLCanalInstance, error) {
	logger.Printf("🔧 Creating MySQL Canal Instance (ID: %s)", id)

	// 转换配置
	logger.Printf("🔧 Converting configuration...")
	mysqlConfig := MySQLConfig{
		Host:       cfg.Canal.Host,
		Port:       cfg.Canal.Port,
		Username:   cfg.Canal.Username,
		Password:   cfg.Canal.Password,
		Database:   "", // 可以监听所有数据库
		ServerID:   cfg.Canal.ServerID,
		BinlogFile: cfg.Canal.Binlog.Filename,
		BinlogPos:  cfg.Canal.Binlog.Position,
	}

	logger.Printf("🔧 MySQL Config: Host=%s, Port=%d, Username=%s, ServerID=%d",
		mysqlConfig.Host, mysqlConfig.Port, mysqlConfig.Username, mysqlConfig.ServerID)

	// 创建事件接收器
	logger.Printf("🔧 Creating event sink...")
	eventSink := NewDefaultEventSink(logger)

	// 尝试创建真实的 MySQL binlog slave
	logger.Printf("🔧 Creating MySQL binlog slave...")
	var binlogSlave BinlogSlave
	realSlave, err := NewMySQLBinlogSlaveWithMeta(mysqlConfig, eventSink, logger, metaManager)
	if err != nil {
		logger.Printf("❌ Failed to create real MySQL binlog slave: %v", err)
		return nil, fmt.Errorf("failed to create real MySQL binlog slave: %v", err)
	}
	binlogSlave = realSlave

	// 配置监听的表和事件类型
	logger.Printf("🔧 Configuring binlog slave from config...")
	configureBinlogSlaveFromConfig(binlogSlave, cfg)

	instance := &MySQLCanalInstance{
		id:          id,
		config:      mysqlConfig,
		eventSink:   eventSink,
		binlogSlave: binlogSlave,
		logger:      logger,
		status: InstanceStatus{
			Running:   false,
			Position:  Position{},
			LastEvent: time.Time{},
		},
	}

	logger.Printf("✅ MySQL Canal Instance created successfully (ID: %s)", id)

	return instance, nil
}

// configureBinlogSlaveFromConfig 配置 binlog slave
func configureBinlogSlaveFromConfig(slave BinlogSlave, cfg *config.Config) {
	// 设置监听的事件类型
	var eventTypes []EventType
	for _, et := range cfg.Canal.Watch.EventTypes {
		switch et {
		case "INSERT":
			eventTypes = append(eventTypes, EventTypeInsert)
		case "UPDATE":
			eventTypes = append(eventTypes, EventTypeUpdate)
		case "DELETE":
			eventTypes = append(eventTypes, EventTypeDelete)
		}
	}
	slave.SetEventTypes(eventTypes)

	// 添加监听的表
	for _, db := range cfg.Canal.Watch.Databases {
		for _, table := range cfg.Canal.Watch.Tables {
			slave.AddWatchTable(db, table)
		}
	}

	// 如果没有指定具体的表，则监听所有表
	if len(cfg.Canal.Watch.Databases) == 0 && len(cfg.Canal.Watch.Tables) == 0 {
		// 不添加任何监听表，表示监听所有表
	}
}

// Start 启动 MySQL Canal 实例
func (c *MySQLCanalInstance) Start(ctx context.Context) error {
	c.logger.Printf("🔧 Starting MySQL Canal Instance %s", c.id)
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		c.logger.Printf("⚠️ MySQL canal instance %s is already running", c.id)
		return fmt.Errorf("mysql canal instance %s is already running", c.id)
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	c.logger.Printf("🚀 Starting MySQL Canal Instance: %s", c.id)
	c.logger.Printf("📡 MySQL Config: %s:%d", c.config.Host, c.config.Port)
	c.logger.Printf("🆔 Server ID: %d", c.config.ServerID)
	c.logger.Printf("🏗️ Architecture: Pure MySQL binlog replication")

	// 启动事件接收器
	c.logger.Printf("🔧 Starting event sink...")
	if err := c.eventSink.Start(c.ctx); err != nil {
		c.logger.Printf("❌ Failed to start event sink: %v", err)
		return fmt.Errorf("failed to start event sink: %v", err)
	}
	c.logger.Printf("✅ Event sink started successfully")

	// 启动 MySQL binlog slave
	c.logger.Printf("🔧 Starting MySQL binlog slave...")
	if err := c.binlogSlave.Start(); err != nil {
		c.logger.Printf("❌ Failed to start mysql binlog slave: %v", err)
		c.logger.Printf("🔧 Stopping event sink due to binlog slave start failure...")
		c.eventSink.Stop()
		return fmt.Errorf("failed to start mysql binlog slave: %v", err)
	}
	c.logger.Printf("✅ MySQL binlog slave started successfully")

	c.running = true
	c.status.Running = true
	c.status.Position = c.binlogSlave.GetBinlogPosition()
	c.status.LastEvent = time.Now()

	c.logger.Printf("✅ MySQL Canal Instance %s started successfully", c.id)
	c.logger.Printf("📊 Initial position: %s:%d", c.status.Position.Name, c.status.Position.Pos)
	return nil
}

// Stop 停止 MySQL Canal 实例
func (c *MySQLCanalInstance) Stop() error {
	c.logger.Printf("🛑 Stopping MySQL Canal Instance: %s", c.id)
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		c.logger.Printf("⚠️ MySQL Canal Instance %s is not running", c.id)
		return nil
	}

	c.logger.Printf("🛑 Stopping MySQL Canal Instance: %s", c.id)

	// 停止 MySQL binlog slave
	c.logger.Printf("🔧 Stopping MySQL binlog slave...")
	if err := c.binlogSlave.Stop(); err != nil {
		c.logger.Printf("❌ Error stopping mysql binlog slave: %v", err)
	}

	// 停止事件接收器
	c.logger.Printf("🔧 Stopping event sink...")
	if err := c.eventSink.Stop(); err != nil {
		c.logger.Printf("❌ Error stopping event sink: %v", err)
	}

	if c.cancel != nil {
		c.logger.Printf("🔧 Cancelling context...")
		c.cancel()
	}

	c.running = false
	c.status.Running = false

	c.logger.Printf("✅ MySQL Canal Instance %s stopped", c.id)
	return nil
}

// StopInstance 停止指定实例
func (c *MySQLCanalInstance) StopInstance(instanceID uint) error {
	return nil
}

// UpdateTask 更新任务
func (c *MySQLCanalInstance) UpdateInstance(instanceID uint, task *database.Task) error {
	return nil
}

// Subscribe 订阅事件
func (c *MySQLCanalInstance) Subscribe(schema, table string, handler EventHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 添加到 MySQL binlog slave 的监听表
	c.binlogSlave.AddWatchTable(schema, table)

	// 订阅到事件接收器
	if err := c.eventSink.Subscribe(schema, table, handler); err != nil {
		return fmt.Errorf("failed to subscribe to event sink: %v", err)
	}

	c.logger.Printf("📋 MySQL Canal Instance %s subscribed to %s.%s with handler %s",
		c.id, schema, table, handler.GetName())
	return nil
}

// Unsubscribe 取消订阅
func (c *MySQLCanalInstance) Unsubscribe(schema, table string, handlerName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.eventSink.Unsubscribe(schema, table, handlerName); err != nil {
		return fmt.Errorf("failed to unsubscribe from event sink: %v", err)
	}

	c.logger.Printf("📋 MySQL Canal Instance %s unsubscribed handler %s from %s.%s",
		c.id, handlerName, schema, table)
	return nil
}

// GetStatus 获取实例状态
func (c *MySQLCanalInstance) GetStatus() InstanceStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 更新状态信息
	if c.running && c.binlogSlave != nil {
		c.status.Position = c.binlogSlave.GetBinlogPosition()
		c.status.Running = c.binlogSlave.IsRunning()

		// 获取统计信息
		stats := c.binlogSlave.GetStats()
		if lastEventTime, ok := stats["last_event_time"].(time.Time); ok {
			c.status.LastEvent = lastEventTime
		}
	}

	return c.status
}

// GetID 获取实例ID
func (c *MySQLCanalInstance) GetID() string {
	return c.id
}

// GetConfig 获取配置
func (c *MySQLCanalInstance) GetConfig() MySQLConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// IsRunning 检查是否正在运行
func (c *MySQLCanalInstance) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetStats 获取详细统计信息
func (c *MySQLCanalInstance) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"id":      c.id,
		"running": c.running,
		"config":  c.config,
		"status":  c.status,
	}

	if c.binlogSlave != nil {
		binlogStats := c.binlogSlave.GetStats()
		stats["binlog"] = binlogStats
	}

	return stats
}

// String 实现 Stringer 接口
func (c *MySQLCanalInstance) String() string {
	return fmt.Sprintf("MySQLCanalInstance{id: %s, host: %s:%d, serverID: %d, running: %v, real_mysql: true}",
		c.id, c.config.Host, c.config.Port, c.config.ServerID, c.running)
}
