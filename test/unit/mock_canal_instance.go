package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pikachun/internal/canal"
)

// MockCanalInstance 模拟的Canal实例实现，用于测试
type MockCanalInstance struct {
	id         string
	mu         sync.RWMutex
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
	status     canal.InstanceStatus
	startDelay time.Duration // 启动延迟，用于模拟阻塞
}

// NewMockCanalInstance 创建模拟的Canal实例
func NewMockCanalInstance(id string, startDelay time.Duration) *MockCanalInstance {
	return &MockCanalInstance{
		id:         id,
		startDelay: startDelay,
		status: canal.InstanceStatus{
			Running:   false,
			Position:  canal.Position{Name: "mock-bin.000001", Pos: 4},
			LastEvent: time.Now(),
		},
	}
}

// Start 启动模拟的Canal实例
func (m *MockCanalInstance) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("mock canal instance %s already running", m.id)
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.running = true

	// 模拟启动延迟
	if m.startDelay > 0 {
		time.Sleep(m.startDelay)
	}

	m.status.Running = true
	m.status.LastEvent = time.Now()

	return nil
}

// Stop 停止模拟的Canal实例
func (m *MockCanalInstance) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.running = false
	m.status.Running = false

	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

// Subscribe 订阅事件
func (m *MockCanalInstance) Subscribe(schema, table string, handler canal.EventHandler) error {
	// 模拟订阅操作
	return nil
}

// Unsubscribe 取消订阅
func (m *MockCanalInstance) Unsubscribe(schema, table string, handlerName string) error {
	// 模拟取消订阅操作
	return nil
}

// GetStatus 获取实例状态
func (m *MockCanalInstance) GetStatus() canal.InstanceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.status
}

// GetStats 获取统计信息
func (m *MockCanalInstance) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"id":      m.id,
		"running": m.running,
		"status":  m.status,
	}
}

// GetID 获取实例ID
func (m *MockCanalInstance) GetID() string {
	return m.id
}

// IsRunning 检查是否正在运行
func (m *MockCanalInstance) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}
