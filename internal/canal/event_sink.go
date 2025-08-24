package canal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DefaultEventSink 默认事件接收器实现
type DefaultEventSink struct {
	mu       sync.RWMutex
	handlers map[string]map[string]EventHandler // schema.table -> handlerName -> handler
	eventCh  chan *Event
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *log.Logger
}

// NewDefaultEventSink 创建默认事件接收器
func NewDefaultEventSink(logger *log.Logger) *DefaultEventSink {
	logger.Printf("🔧 Creating Default Event Sink with buffer size: %d", 1000)

	sink := &DefaultEventSink{
		handlers: make(map[string]map[string]EventHandler),
		eventCh:  make(chan *Event, 1000), // 缓冲区大小
		logger:   logger,
	}

	logger.Printf("✅ Default Event Sink created successfully")
	return sink
}

// Start 启动事件接收器
func (s *DefaultEventSink) Start(ctx context.Context) error {
	s.logger.Printf("🔧 Starting event sink...")
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ctx != nil {
		s.logger.Printf("⚠️ Event sink already started")
		return fmt.Errorf("event sink already started")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// 启动事件处理协程
	s.logger.Printf("🔧 Starting event processing goroutine...")
	s.wg.Add(1)
	go s.processEvents()

	s.logger.Printf("✅ Event sink started")
	return nil
}

// Stop 停止事件接收器
func (s *DefaultEventSink) Stop() error {
	s.logger.Printf("🛑 Stopping event sink...")
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.logger.Printf("🔧 Cancelling context and waiting for goroutines...")
		s.cancel()
		s.logger.Printf("🔧 Waiting for goroutines to finish...")
		s.wg.Wait()
		s.cancel = nil
		s.ctx = nil
		s.logger.Printf("✅ Goroutines stopped")
	}

	s.logger.Printf("✅ Event sink stopped")
	return nil
}
// Subscribe 订阅事件
func (s *DefaultEventSink) Subscribe(schema, table string, handler EventHandler) error {
	s.logger.Printf("📋 Subscribing handler %s for %s.%s", handler.GetName(), schema, table)
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s.%s", schema, table)
	s.logger.Printf("🔧 Creating handler map for key: %s", key)
	if s.handlers[key] == nil {
		s.handlers[key] = make(map[string]EventHandler)
		s.logger.Printf("🆕 Created new handler map for %s", key)
	}

	s.handlers[key][handler.GetName()] = handler
	s.logger.Printf("✅ Subscribed handler %s for %s", handler.GetName(), key)
	s.logger.Printf("📊 Total handlers for %s: %d", key, len(s.handlers[key]))
	return nil
}

// Unsubscribe 取消订阅
func (s *DefaultEventSink) Unsubscribe(schema, table string, handlerName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s.%s", schema, table)
	if handlers, exists := s.handlers[key]; exists {
		delete(handlers, handlerName)
		if len(handlers) == 0 {
			delete(s.handlers, key)
		}
	}

	s.logger.Printf("Unsubscribed handler %s for %s", handlerName, key)
	return nil
}

// SendEvent 发送事件
func (s *DefaultEventSink) SendEvent(event *Event) error {
	s.logger.Printf("📤 Sending event to sink: %s.%s %s", event.Schema, event.Table, event.EventType)
	s.logger.Printf("📋 Event details - ID: %s, Timestamp: %s", event.ID, event.Timestamp.Format(time.RFC3339))

	select {
	case s.eventCh <- event:
		s.logger.Printf("✅ Event sent to channel successfully")
		return nil
	case <-time.After(5 * time.Second):
		s.logger.Printf("❌ Send event timeout after 5 seconds")
		return fmt.Errorf("send event timeout")
	}
}
// processEvents 处理事件
func (s *DefaultEventSink) processEvents() {
	s.logger.Printf("👀 Starting event processing goroutine")
	defer s.wg.Done()
	defer s.logger.Printf("👋 Event processing goroutine stopped")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Printf("🛑 Event processing context cancelled")
			return
		case event := <-s.eventCh:
			s.logger.Printf("📥 Received event from channel: %s.%s %s",
				event.Schema, event.Table, event.EventType)
			s.logger.Printf("📋 Event details - ID: %s, Timestamp: %s", event.ID, event.Timestamp.Format(time.RFC3339))
			s.handleEvent(event)
			s.logger.Printf("✅ Event processing completed")
		}
	}
}

// handleEvent 处理单个事件
func (s *DefaultEventSink) handleEvent(event *Event) {
	s.logger.Printf("🔧 Handling event: %s.%s %s", event.Schema, event.Table, event.EventType)
	s.logger.Printf("📋 Event details - ID: %s, Timestamp: %s", event.ID, event.Timestamp.Format(time.RFC3339))

	s.mu.RLock()
	key := fmt.Sprintf("%s.%s", event.Schema, event.Table)
	s.logger.Printf("📋 Looking up handlers for %s", key)

	handlers := make(map[string]EventHandler)
	if h, exists := s.handlers[key]; exists {
		for name, handler := range h {
			handlers[name] = handler
		}
	}
	s.mu.RUnlock()

	s.logger.Printf("📊 Found %d handlers for event", len(handlers))

	// 并发处理所有订阅的处理器
	var wg sync.WaitGroup
	for name, handler := range handlers {
		s.logger.Printf("🚀 Starting handler %s for event", name)
		wg.Add(1)
		go func(name string, handler EventHandler) {
			defer wg.Done()
			s.logger.Printf("🔄 Handler %s started processing event", name)

			ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
			defer cancel()

			if err := handler.Handle(ctx, event); err != nil {
				s.logger.Printf("❌ Handler %s failed to process event %s: %v", name, event.ID, err)
			} else {
				s.logger.Printf("✅ Handler %s completed processing event", name)
			}
		}(name, handler)
	}

	wg.Wait()
	s.logger.Printf("🎉 All handlers completed for event")
}
