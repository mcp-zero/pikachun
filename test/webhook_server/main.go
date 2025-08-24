package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Event 事件结构体
type Event struct {
	ID        string                 `json:"id"`
	Schema    string                 `json:"schema"`
	Table     string                 `json:"table"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Position  map[string]interface{} `json:"position"`
	AfterData map[string]interface{} `json:"after_data"`
	SQL       string                 `json:"sql"`
}

// WebhookPayload webhook负载结构体
type WebhookPayload struct {
	Events    []Event `json:"events"`
	Timestamp int64   `json:"timestamp"`
	Source    string  `json:"source"`
}

func main() {
	// 设置webhook处理器
	http.HandleFunc("/webhook/test", handleWebhook)
	http.HandleFunc("/health", handleHealth)

	fmt.Println("🚀 Webhook测试服务器启动")
	fmt.Println("📡 监听地址: http://localhost:9669")
	fmt.Println("🎯 Webhook端点: http://localhost:9669/webhook/test")
	fmt.Println("💚 健康检查: http://localhost:9669/health")
	fmt.Println("============================================")

	// 启动服务器
	log.Fatal(http.ListenAndServe(":9669", nil))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// 记录请求信息
	fmt.Printf("\n🔥 收到Webhook请求 [%s]\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("📋 Method: %s\n", r.Method)
	fmt.Printf("📋 URL: %s\n", r.URL.String())
	fmt.Printf("📋 Headers:\n")
	for key, values := range r.Header {
		for _, value := range values {
			fmt.Printf("   %s: %s\n", key, value)
		}
	}

	// 只接受POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("❌ 读取请求体失败: %v\n", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("📊 请求体大小: %d bytes\n", len(body))
	fmt.Printf("📄 原始请求体:\n%s\n", string(body))

	// 解析JSON
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Printf("❌ JSON解析失败: %v\n", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 处理事件
	fmt.Printf("✅ 成功解析JSON\n")
	fmt.Printf("📋 事件数量: %d\n", len(payload.Events))
	fmt.Printf("📋 时间戳: %d\n", payload.Timestamp)
	fmt.Printf("📋 来源: %s\n", payload.Source)

	// 详细显示每个事件
	for i, event := range payload.Events {
		fmt.Printf("\n🎯 事件 #%d:\n", i+1)
		fmt.Printf("   ID: %s\n", event.ID)
		fmt.Printf("   数据库: %s\n", event.Schema)
		fmt.Printf("   表: %s\n", event.Table)
		fmt.Printf("   事件类型: %s\n", event.EventType)
		fmt.Printf("   时间: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   SQL: %s\n", event.SQL)

		// 显示数据变更
		if event.AfterData != nil {
			fmt.Printf("   变更后数据:\n")
			for key, value := range event.AfterData {
				if columns, ok := value.(map[string]interface{}); ok {
					for colName, colValue := range columns {
						fmt.Printf("     %s: %v\n", colName, colValue)
					}
				} else {
					fmt.Printf("     %s: %v\n", key, value)
				}
			}
		}

		// 显示binlog位置
		if event.Position != nil {
			fmt.Printf("   Binlog位置:\n")
			for key, value := range event.Position {
				fmt.Printf("     %s: %v\n", key, value)
			}
		}
	}

	// 返回成功响应
	response := map[string]interface{}{
		"status":    "success",
		"message":   "Events received successfully",
		"processed": len(payload.Events),
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	fmt.Printf("✅ Webhook处理完成，返回成功响应\n")
	fmt.Println("============================================")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "webhook-test-server",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
