package main

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"pikachun/internal/canal"
)

// TestDefaultEventSink 测试 DefaultEventSink 的基本功能
func TestDefaultEventSink(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDefaultEventSink] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := canal.NewDefaultEventSink(logger)

	// 测试订阅事件
	handler := &MockEventHandler{name: "test-handler"}
	err := eventSink.Subscribe("test", "users", handler)
	if err != nil {
		t.Errorf("Failed to subscribe: %v", err)
	}

	// 测试取消订阅
	err = eventSink.Unsubscribe("test", "users", "test-handler")
	if err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
}

// TestDefaultEventSinkStartStop 测试 DefaultEventSink 的启动和停止
func TestDefaultEventSinkStartStop(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDefaultEventSinkStartStop] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := canal.NewDefaultEventSink(logger)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动事件接收器
	err := eventSink.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start event sink: %v", err)
	}

	// 再次启动应该失败
	err = eventSink.Start(ctx)
	if err == nil {
		t.Error("Expected second start to fail")
	}

	// 停止事件接收器
	err = eventSink.Stop()
	if err != nil {
		t.Errorf("Failed to stop event sink: %v", err)
	}

	// 再次停止应该成功
	err = eventSink.Stop()
	if err != nil {
		t.Errorf("Failed to stop event sink second time: %v", err)
	}
}

// TestDefaultEventSinkEventHandling 测试事件处理
func TestDefaultEventSinkEventHandling(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDefaultEventSinkEventHandling] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := canal.NewDefaultEventSink(logger)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动事件接收器
	err := eventSink.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start event sink: %v", err)
	}

	// 创建事件处理器
	handled := make(chan *canal.Event, 1)
	handler := &MockEventHandler{
		name:    "test-handler",
		handled: handled,
	}

	// 订阅事件
	err = eventSink.Subscribe("test", "users", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// 创建测试事件
	event := &canal.Event{
		ID:        "test-event-1",
		Schema:    "test",
		Table:     "users",
		EventType: canal.EventTypeInsert,
		Timestamp: time.Now(),
		Position: canal.Position{
			Name: "mysql-bin.000001",
			Pos:  12345,
		},
		AfterData: &canal.RowData{
			Columns: []canal.Column{
				{Name: "id", Type: "int", Value: 1, IsNull: false},
				{Name: "name", Type: "varchar", Value: "test user", IsNull: false},
			},
		},
	}

	// 发送事件
	err = eventSink.SendEvent(event)
	if err != nil {
		t.Errorf("Failed to send event: %v", err)
	}

	// 等待事件处理完成
	select {
	case receivedEvent := <-handled:
		if receivedEvent.ID != event.ID {
			t.Errorf("Expected event ID %s, got %s", event.ID, receivedEvent.ID)
		}
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for event to be handled")
	}

	// 停止事件接收器
	err = eventSink.Stop()
	if err != nil {
		t.Errorf("Failed to stop event sink: %v", err)
	}
}

// TestDefaultEventSinkMultipleHandlers 测试多个事件处理器
func TestDefaultEventSinkMultipleHandlers(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDefaultEventSinkMultipleHandlers] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := canal.NewDefaultEventSink(logger)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动事件接收器
	err := eventSink.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start event sink: %v", err)
	}

	// 创建多个事件处理器
	handled1 := make(chan *canal.Event, 1)
	handler1 := &MockEventHandler{
		name:    "handler-1",
		handled: handled1,
	}

	handled2 := make(chan *canal.Event, 1)
	handler2 := &MockEventHandler{
		name:    "handler-2",
		handled: handled2,
	}

	// 订阅事件
	err = eventSink.Subscribe("test", "users", handler1)
	if err != nil {
		t.Fatalf("Failed to subscribe handler1: %v", err)
	}

	err = eventSink.Subscribe("test", "users", handler2)
	if err != nil {
		t.Fatalf("Failed to subscribe handler2: %v", err)
	}

	// 创建测试事件
	event := &canal.Event{
		ID:        "test-event-2",
		Schema:    "test",
		Table:     "users",
		EventType: canal.EventTypeUpdate,
		Timestamp: time.Now(),
		Position: canal.Position{
			Name: "mysql-bin.000001",
			Pos:  23456,
		},
		BeforeData: &canal.RowData{
			Columns: []canal.Column{
				{Name: "id", Type: "int", Value: 1, IsNull: false},
				{Name: "name", Type: "varchar", Value: "old name", IsNull: false},
			},
		},
		AfterData: &canal.RowData{
			Columns: []canal.Column{
				{Name: "id", Type: "int", Value: 1, IsNull: false},
				{Name: "name", Type: "varchar", Value: "new name", IsNull: false},
			},
		},
	}

	// 发送事件
	err = eventSink.SendEvent(event)
	if err != nil {
		t.Errorf("Failed to send event: %v", err)
	}

	// 等待事件处理完成
	handledCount := 0
	timeout := time.After(10 * time.Second)
	for handledCount < 2 {
		select {
		case <-handled1:
			handledCount++
		case <-handled2:
			handledCount++
		case <-timeout:
			t.Errorf("Timeout waiting for events to be handled. Only %d of 2 handlers completed.", handledCount)
			return
		}
	}

	// 停止事件接收器
	err = eventSink.Stop()
	if err != nil {
		t.Errorf("Failed to stop event sink: %v", err)
	}
}
