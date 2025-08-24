package canal

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

// VitessCanalInstance 基于Vitess的Canal实例实现
type VitessCanalInstance struct {
	id          string
	config      MySQLConfig
	eventSink   *DefaultEventSink
	vitessSlave *VitessBinlogSlave
	logger      *log.Logger
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	status      InstanceStatus
}

// NewVitessCanalInstance 创建基于Vitess的Canal实例
func NewVitessCanalInstance(id string, config MySQLConfig, logger *log.Logger) (*VitessCanalInstance, error) {
	// 创建事件接收器
	eventSink := NewDefaultEventSink(logger)

	// 创建Vitess binlog slave
	vitessSlave, err := NewVitessBinlogSlave(config, eventSink, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create vitess binlog slave: %v", err)
	}

	instance := &VitessCanalInstance{
		id:          id,
		config:      config,
		eventSink:   eventSink,
		vitessSlave: vitessSlave,
		logger:      logger,
		status: InstanceStatus{
			Running:   false,
			Position:  Position{},
			LastEvent: time.Time{},
		},
	}

	return instance, nil
}

// Start 启动Vitess Canal实例
func (c *VitessCanalInstance) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("vitess canal instance %s is already running", c.id)
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	c.logger.Printf("🚀 Starting Vitess Canal Instance: %s", c.id)
	c.logger.Printf("📡 MySQL Config: %s:%d", c.config.Host, c.config.Port)
	c.logger.Printf("🆔 Server ID: %d", c.config.ServerID)
	c.logger.Printf("🏗️ Architecture: Vitess slaveConnection binlog dump")

	// 启动事件接收器
	if err := c.eventSink.Start(c.ctx); err != nil {
		return fmt.Errorf("failed to start event sink: %v", err)
	}

	// 启动Vitess binlog slave
	if err := c.vitessSlave.Start(); err != nil {
		c.eventSink.Stop()
		return fmt.Errorf("failed to start vitess binlog slave: %v", err)
	}

	c.running = true
	c.status.Running = true
	c.status.Position = c.vitessSlave.GetBinlogPosition()
	c.status.LastEvent = time.Now()

	c.logger.Printf("✅ Vitess Canal Instance %s started successfully", c.id)
	return nil
}

// Stop 停止Vitess Canal实例
func (c *VitessCanalInstance) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.logger.Printf("🛑 Stopping Vitess Canal Instance: %s", c.id)

	// 停止Vitess binlog slave
	if err := c.vitessSlave.Stop(); err != nil {
		c.logger.Printf("❌ Error stopping vitess binlog slave: %v", err)
	}

	// 停止事件接收器
	if err := c.eventSink.Stop(); err != nil {
		c.logger.Printf("❌ Error stopping event sink: %v", err)
	}

	if c.cancel != nil {
		c.cancel()
	}

	c.running = false
	c.status.Running = false

	c.logger.Printf("✅ Vitess Canal Instance %s stopped", c.id)
	return nil
}

// Subscribe 订阅事件
func (c *VitessCanalInstance) Subscribe(schema, table string, handler EventHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 添加到Vitess binlog slave的监听表
	c.vitessSlave.AddWatchTable(schema, table)

	// 订阅到事件接收器
	if err := c.eventSink.Subscribe(schema, table, handler); err != nil {
		return fmt.Errorf("failed to subscribe to event sink: %v", err)
	}

	c.logger.Printf("📋 Vitess Canal Instance %s subscribed to %s.%s with handler %s",
		c.id, schema, table, handler.GetName())
	return nil
}

// Unsubscribe 取消订阅
func (c *VitessCanalInstance) Unsubscribe(schema, table string, handlerName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.eventSink.Unsubscribe(schema, table, handlerName); err != nil {
		return fmt.Errorf("failed to unsubscribe from event sink: %v", err)
	}

	c.logger.Printf("📋 Vitess Canal Instance %s unsubscribed handler %s from %s.%s",
		c.id, handlerName, schema, table)
	return nil
}

// GetStatus 获取实例状态
func (c *VitessCanalInstance) GetStatus() InstanceStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 更新状态信息
	if c.running && c.vitessSlave != nil {
		c.status.Position = c.vitessSlave.GetBinlogPosition()
		c.status.Running = c.vitessSlave.IsRunning()
	}

	return c.status
}

// GetID 获取实例ID
func (c *VitessCanalInstance) GetID() string {
	return c.id
}

// GetConfig 获取配置
func (c *VitessCanalInstance) GetConfig() MySQLConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// IsRunning 检查是否正在运行
func (c *VitessCanalInstance) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetStats 获取详细统计信息
func (c *VitessCanalInstance) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"id":      c.id,
		"running": c.running,
		"config":  c.config,
		"status":  c.status,
	}

	if c.vitessSlave != nil {
		// 使用反射来访问VitessBinlogSlave的私有字段
		v := reflect.ValueOf(c.vitessSlave).Elem()

		// 获取processedCount和failedCount字段
		processedCount := int64(0)
		failedCount := int64(0)

		if processedCountField := v.FieldByName("processedCount"); processedCountField.IsValid() {
			processedCount = processedCountField.Int()
		}

		if failedCountField := v.FieldByName("failedCount"); failedCountField.IsValid() {
			failedCount = failedCountField.Int()
		}

		// 使用VitessBinlogSlave的其他方法来获取统计信息
		binlogStats := map[string]interface{}{
			"position":         c.vitessSlave.GetBinlogPosition(),
			"running":          c.vitessSlave.IsRunning(),
			"processed_events": processedCount,
			"failed_events":    failedCount,
		}
		stats["binlog"] = binlogStats
	}

	return stats
}

// String 实现Stringer接口
func (c *VitessCanalInstance) String() string {
	return fmt.Sprintf("VitessCanalInstance{id: %s, host: %s:%d, serverID: %d, running: %v, vitess: true}",
		c.id, c.config.Host, c.config.Port, c.config.ServerID, c.running)
}
