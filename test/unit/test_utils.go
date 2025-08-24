/*
 * @Author: lucklidi@126.com
 * @Date: 2025-08-22 22:22:28
 * @LastEditTime: 2025-08-22 22:22:33
 * @LastEditors: lucklidi@126.com
 * @Description:
 * Copyright (c) 2023 by pikachun
 */
package main

import (
	"context"

	"pikachun/internal/canal"
)

// MockEventHandler 模拟事件处理器
type MockEventHandler struct {
	name    string
	handled chan *canal.Event
}

func (h *MockEventHandler) Handle(ctx context.Context, event *canal.Event) error {
	if h.handled != nil {
		select {
		case h.handled <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (h *MockEventHandler) GetName() string {
	return h.name
}
