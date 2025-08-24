package canal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebhookHandler Webhook事件处理器
type WebhookHandler struct {
	name        string
	callbackURL string
	client      *http.Client
	logger      *log.Logger

	// 批处理配置
	batchSize    int
	batchTimeout time.Duration
	eventBuffer  []*Event
	bufferMu     sync.Mutex
	flushTimer   *time.Timer

	// 重试配置
	maxRetries    int
	retryInterval time.Duration

	// 性能统计
	successCount int64
	errorCount   int64
	mu           sync.RWMutex
}

// NewWebhookHandler 创建Webhook处理器
func NewWebhookHandler(name, callbackURL string, logger *log.Logger) *WebhookHandler {
	logger.Printf("🔧 Creating Webhook Handler (Name: %s, URL: %s)", name, callbackURL)

	handler := &WebhookHandler{
		name:          name,
		callbackURL:   callbackURL,
		logger:        logger,
		client:        &http.Client{Timeout: 30 * time.Second},
		batchSize:     10,              // 批处理大小
		batchTimeout:  5 * time.Second, // 批处理超时
		maxRetries:    3,               // 最大重试次数
		retryInterval: time.Second,     // 重试间隔
		eventBuffer:   make([]*Event, 0, 10),
	}

	logger.Printf("✅ Webhook Handler created successfully (Name: %s)", name)
	return handler
}

// GetName 获取处理器名称
func (h *WebhookHandler) GetName() string {
	return h.name
}

// Handle 处理事件（支持批处理）
func (h *WebhookHandler) Handle(ctx context.Context, event *Event) error {
	h.logger.Printf("📥 Webhook handler %s received event: %s.%s %s",
		h.name, event.Schema, event.Table, event.EventType)

	h.bufferMu.Lock()
	defer h.bufferMu.Unlock()

	// 添加事件到缓冲区
	h.eventBuffer = append(h.eventBuffer, event)
	h.logger.Printf("📦 Added event to buffer, current buffer size: %d", len(h.eventBuffer))

	// 检查是否需要立即刷新
	if len(h.eventBuffer) >= h.batchSize {
		h.logger.Printf("📊 Buffer size reached batch size %d, flushing events", h.batchSize)
		return h.flushEvents(ctx)
	}

	// 设置定时器，确保事件不会在缓冲区中停留太久
	if h.flushTimer != nil {
		h.flushTimer.Stop()
	}
	h.flushTimer = time.AfterFunc(h.batchTimeout, func() {
		h.logger.Printf("⏰ Batch timeout reached, flushing events")
		h.bufferMu.Lock()
		defer h.bufferMu.Unlock()
		if len(h.eventBuffer) > 0 {
			// 创建新的context，避免使用已取消的context
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			h.flushEvents(timeoutCtx)
		}
	})

	h.logger.Printf("✅ Event handled by webhook handler %s", h.name)
	return nil
}

// flushEvents 刷新事件缓冲区
func (h *WebhookHandler) flushEvents(ctx context.Context) error {
	h.logger.Printf("🔄 Flushing events buffer, size: %d", len(h.eventBuffer))
	if len(h.eventBuffer) == 0 {
		h.logger.Printf("⚠️ Event buffer is empty, nothing to flush")
		return nil
	}

	// 复制事件并清空缓冲区
	events := make([]*Event, len(h.eventBuffer))
	copy(events, h.eventBuffer)
	h.eventBuffer = h.eventBuffer[:0]
	h.logger.Printf("📋 Copied %d events from buffer", len(events))

	// 停止定时器
	if h.flushTimer != nil {
		h.logger.Printf("⏰ Stopping flush timer")
		h.flushTimer.Stop()
		h.flushTimer = nil
	}

	// 异步发送事件 - 创建新的context避免使用已取消的context
	h.logger.Printf("🚀 Sending %d events asynchronously", len(events))
	sendCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	go func() {
		defer cancel()
		h.sendEventsWithRetry(sendCtx, events)
	}()
	h.logger.Printf("✅ Flush events completed")
	return nil
}

// sendEventsWithRetry 带重试的事件发送
func (h *WebhookHandler) sendEventsWithRetry(ctx context.Context, events []*Event) {
	h.logger.Printf("🔄 Starting send events with retry, events: %d, max retries: %d",
		len(events), h.maxRetries)
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		h.logger.Printf("📤 Sending attempt %d/%d", attempt+1, h.maxRetries+1)
		if attempt > 0 {
			// 指数退避
			backoff := time.Duration(attempt) * h.retryInterval
			h.logger.Printf("⏳ Waiting for backoff: %v", backoff)
			select {
			case <-ctx.Done():
				h.logger.Printf("🛑 Context cancelled during backoff")
				return
			case <-time.After(backoff):
				h.logger.Printf("⏰ Backoff completed")
			}
		}

		if err := h.sendEvents(ctx, events); err != nil {
			lastErr = err
			h.logger.Printf("❌ Attempt %d failed for handler %s: %v", attempt+1, h.name, err)

			h.mu.Lock()
			h.errorCount++
			h.mu.Unlock()

			continue
		}

		// 成功发送
		h.logger.Printf("✅ Successfully sent %d events to %s", len(events), h.callbackURL)
		h.mu.Lock()
		h.successCount += int64(len(events))
		h.mu.Unlock()

		h.logger.Printf("🎉 All events sent successfully on attempt %d", attempt+1)
		return
	}

	// 所有重试都失败了
	h.logger.Printf("💥 Failed to send events after %d attempts to %s: %v",
		h.maxRetries+1, h.callbackURL, lastErr)
}

// sendEvents 发送事件到Webhook
func (h *WebhookHandler) sendEvents(ctx context.Context, events []*Event) error {
	h.logger.Printf("📤 Sending %d events to webhook: %s", len(events), h.callbackURL)

	// 构建请求体
	h.logger.Printf("🔧 Building payload with %d events", len(events))
	payload := map[string]interface{}{
		"events":    events,
		"timestamp": time.Now().Unix(),
		"source":    "canal-pikachun",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		h.logger.Printf("❌ Failed to marshal events: %v", err)
		return fmt.Errorf("failed to marshal events: %v", err)
	}
	h.logger.Printf("✅ Payload marshaled, size: %d bytes", len(jsonData))

	// 创建HTTP请求
	h.logger.Printf("🔧 Creating HTTP request to %s", h.callbackURL)
	req, err := http.NewRequestWithContext(ctx, "POST", h.callbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		h.logger.Printf("❌ Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Canal-Pikachun/1.0")
	req.Header.Set("X-Event-Count", fmt.Sprintf("%d", len(events)))
	h.logger.Printf("📋 Request headers set: Content-Type=application/json, User-Agent=Canal-Pikachun/1.0, X-Event-Count=%d", len(events))

	// 发送请求
	h.logger.Printf("🚀 Sending HTTP request to %s", h.callbackURL)
	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Printf("❌ Failed to send request to %s: %v", h.callbackURL, err)
		return fmt.Errorf("failed to send request to %s: %v", h.callbackURL, err)
	}
	defer resp.Body.Close()
	h.logger.Printf("✅ HTTP request sent to %s, status: %d", h.callbackURL, resp.StatusCode)

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		h.logger.Printf("❌ Webhook %s returned status %d: %s", h.callbackURL, resp.StatusCode, string(body))
		return fmt.Errorf("webhook %s returned status %d: %s", h.callbackURL, resp.StatusCode, string(body))
	}

	h.logger.Printf("🎉 Webhook request to %s successful", h.callbackURL)
	return nil
}

// GetStats 获取处理器统计信息
func (h *WebhookHandler) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"name":          h.name,
		"callback_url":  h.callbackURL,
		"success_count": h.successCount,
		"error_count":   h.errorCount,
		"buffer_size":   len(h.eventBuffer),
	}
}

// DatabaseHandler 数据库处理器
type DatabaseHandler struct {
	name      string
	taskID    uint
	logger    *log.Logger
	dbService EventLogger
	enabled   bool

	mu           sync.RWMutex
	processCount int64
}

// NewDatabaseHandler 创建数据库处理器
// NewDatabaseHandler 创建数据库处理器
func NewDatabaseHandler(name string, taskID uint, logger *log.Logger, dbService EventLogger, enabled bool) *DatabaseHandler {
	logger.Printf("🔧 Creating Database Handler (Name: %s, TaskID: %d, Enabled: %t)", name, taskID, enabled)

	handler := &DatabaseHandler{
		name:      name,
		taskID:    taskID,
		logger:    logger,
		dbService: dbService,
		enabled:   enabled,
	}

	logger.Printf("✅ Database Handler created successfully (Name: %s)", name)
	return handler
}

// GetName 获取处理器名称
func (h *DatabaseHandler) GetName() string {
	return h.name
}

// Handle 处理事件
func (h *DatabaseHandler) Handle(ctx context.Context, event *Event) error {
	h.mu.Lock()
	h.processCount++
	h.mu.Unlock()

	// 检查是否启用了数据库存储功能
	if !h.enabled {
		h.logger.Printf("📥 Database handler %s received event: %s.%s %s (database storage disabled)",
			h.name, event.Schema, event.Table, event.EventType)
		return nil
	}

	// 这里可以将事件保存到数据库
	h.logger.Printf("📥 Database handler %s received event: %s.%s %s",
		h.name, event.Schema, event.Table, event.EventType)

	// 记录事件详情
	if event.BeforeData != nil {
		h.logger.Printf("📋 Before data: %+v", event.BeforeData)
	}
	if event.AfterData != nil {
		h.logger.Printf("📋 After data: %+v", event.AfterData)
	}

	// 实际的数据库保存逻辑
	data := ""
	if event.AfterData != nil {
		// 将AfterData转换为JSON字符串
		dataBytes, _ := json.Marshal(event.AfterData)
		data = string(dataBytes)
	}

	// 调用TaskService的CreateEventLog方法
	err := h.dbService.CreateEventLog(h.taskID, event.Schema, event.Table, string(event.EventType), data, "success", "")
	if err != nil {
		h.logger.Printf("❌ Failed to save event log to database: %v", err)
		return err
	}

	h.logger.Printf("✅ Database handler %s processed event successfully", h.name)
	return nil
}

// GetStats 获取处理器统计信息
func (h *DatabaseHandler) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"name":          h.name,
		"task_id":       h.taskID,
		"process_count": h.processCount,
	}
}
