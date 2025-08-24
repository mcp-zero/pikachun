/*
 * @Author: lucklidi@126.com
 * @Date: 2025-08-22 22:26:30
 * @LastEditTime: 2025-08-24 10:08:08
 * @LastEditors: lucklidi@126.com
 * @Description:
 * Copyright (c) 2023 by pikachun
 */
package canal

import (
	"context"
	"log"
	"os"
	"testing"
)

// TestEventSinkLogging 测试 EventSink 的日志功能
func TestEventSinkLogging(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestEventSinkLogging] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := NewDefaultEventSink(logger)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动事件接收器（这会触发日志输出）
	err := eventSink.Start(ctx)
	if err != nil {
		t.Logf("Expected start to fail if already started: %v", err)
	}

	// 停止事件接收器（这会触发日志输出）
	err = eventSink.Stop()
	if err != nil {
		t.Errorf("Failed to stop event sink: %v", err)
	}

	// 再次停止应该成功（这会触发日志输出）
	err = eventSink.Stop()
	if err != nil {
		t.Errorf("Failed to stop event sink second time: %v", err)
	}
}

// TestEventSinkSubscribeLogging 测试 EventSink 订阅的日志功能
func TestEventSinkSubscribeLogging(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestEventSinkSubscribeLogging] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := NewDefaultEventSink(logger)

	// 创建一个简单的事件处理器
	handler := &testEventHandler{name: "test-handler"}

	// 订阅事件（这会触发日志输出）
	err := eventSink.Subscribe("test", "users", handler)
	if err != nil {
		t.Errorf("Failed to subscribe: %v", err)
	}

	// 取消订阅（这会触发日志输出）
	err = eventSink.Unsubscribe("test", "users", "test-handler")
	if err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
}

// testEventHandler 简单的事件处理器实现
type testEventHandler struct {
	name string
}

func (h *testEventHandler) Handle(ctx context.Context, event *Event) error {
	return nil
}

func (h *testEventHandler) GetName() string {
	return h.name
}
