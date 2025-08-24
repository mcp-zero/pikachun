package canal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DefaultCanalInstance Canal实例的默认实现
type DefaultCanalInstance struct {
	instanceID  string
	parser      Parser
	eventSink   EventSink
	metaManager MetaManager
	logger      *log.Logger

	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// 性能监控
	lastEventTime time.Time
	eventCount    int64
	errorCount    int64
}

// NewDefaultCanalInstance 创建默认Canal实例
func NewDefaultCanalInstance(
	instanceID string,
	parser Parser,
	eventSink EventSink,
	metaManager MetaManager,
	logger *log.Logger,
) *DefaultCanalInstance {
	return &DefaultCanalInstance{
		instanceID:  instanceID,
		parser:      parser,
		eventSink:   eventSink,
		metaManager: metaManager,
		logger:      logger,
	}
}

// Start 启动Canal实例
func (c *DefaultCanalInstance) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("canal instance %s already running", c.instanceID)
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// 启动事件接收器
	if err := c.eventSink.Start(c.ctx); err != nil {
		c.running = false
		return fmt.Errorf("failed to start event sink: %v", err)
	}

	// 设置解析器的事件接收器
	c.parser.SetEventSink(c.eventSink)

	// 恢复位置信息
	if pos, err := c.metaManager.LoadPosition(c.instanceID); err != nil {
		c.logger.Printf("Failed to load position for instance %s: %v", c.instanceID, err)
	} else if pos.Name != "" {
		if err := c.parser.SetPosition(pos); err != nil {
			c.logger.Printf("Failed to set position for instance %s: %v", c.instanceID, err)
		} else {
			c.logger.Printf("Restored position for instance %s: %+v", c.instanceID, pos)
		}
	}

	// 启动解析器
	if err := c.parser.Start(c.ctx); err != nil {
		c.eventSink.Stop()
		c.running = false
		return fmt.Errorf("failed to start parser: %v", err)
	}

	// 启动位置保存协程
	c.wg.Add(1)
	go c.positionSaver()

	// 启动健康检查协程
	c.wg.Add(1)
	go c.healthChecker()

	c.logger.Printf("Canal instance %s started successfully", c.instanceID)
	return nil
}

// Stop 停止Canal实例
func (c *DefaultCanalInstance) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.running = false

	// 停止解析器
	if err := c.parser.Stop(); err != nil {
		c.logger.Printf("Failed to stop parser: %v", err)
	}

	// 停止事件接收器
	if err := c.eventSink.Stop(); err != nil {
		c.logger.Printf("Failed to stop event sink: %v", err)
	}

	// 取消上下文并等待协程结束
	if c.cancel != nil {
		c.cancel()
		c.wg.Wait()
	}

	// 保存最终位置
	if pos := c.parser.GetPosition(); pos.Name != "" {
		if err := c.metaManager.SavePosition(c.instanceID, pos); err != nil {
			c.logger.Printf("Failed to save final position: %v", err)
		}
	}

	c.logger.Printf("Canal instance %s stopped", c.instanceID)
	return nil
}

// Subscribe 订阅事件
func (c *DefaultCanalInstance) Subscribe(schema, table string, handler EventHandler) error {
	return c.eventSink.Subscribe(schema, table, handler)
}

// Unsubscribe 取消订阅
func (c *DefaultCanalInstance) Unsubscribe(schema, table string, handlerName string) error {
	return c.eventSink.Unsubscribe(schema, table, handlerName)
}

// GetStatus 获取实例状态
func (c *DefaultCanalInstance) GetStatus() InstanceStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := InstanceStatus{
		Running:   c.running,
		Position:  c.parser.GetPosition(),
		LastEvent: c.lastEventTime,
	}

	return status
}

// positionSaver 定期保存位置信息
func (c *DefaultCanalInstance) positionSaver() {
	defer c.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 每10秒保存一次位置
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.savePosition()
		}
	}
}

// savePosition 保存当前位置
func (c *DefaultCanalInstance) savePosition() {
	pos := c.parser.GetPosition()
	if pos.Name == "" {
		return
	}

	if err := c.metaManager.SavePosition(c.instanceID, pos); err != nil {
		c.logger.Printf("Failed to save position for instance %s: %v", c.instanceID, err)
		c.mu.Lock()
		c.errorCount++
		c.mu.Unlock()
	}
}

// healthChecker 健康检查
func (c *DefaultCanalInstance) healthChecker() {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.checkHealth()
		}
	}
}

// checkHealth 执行健康检查
func (c *DefaultCanalInstance) checkHealth() {
	c.mu.RLock()
	lastEvent := c.lastEventTime
	errorCount := c.errorCount
	c.mu.RUnlock()

	// 检查是否长时间没有事件（可能表示连接断开）
	if !lastEvent.IsZero() && time.Since(lastEvent) > 5*time.Minute {
		c.logger.Printf("Warning: No events received for %v in instance %s",
			time.Since(lastEvent), c.instanceID)
	}

	// 检查错误率
	if errorCount > 0 {
		c.logger.Printf("Instance %s has %d errors", c.instanceID, errorCount)
	}
}

// updateEventStats 更新事件统计
func (c *DefaultCanalInstance) updateEventStats() {
	c.mu.Lock()
	c.lastEventTime = time.Now()
	c.eventCount++
	c.mu.Unlock()
}
