package main

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"pikachun/internal/canal"
)

// TestMySQLBinlogSlaveBlocking 测试 MySQLBinlogSlave 在处理大量事件时的阻塞情况
func TestMySQLBinlogSlaveBlocking(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlaveBlocking] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := canal.NewDefaultEventSink(logger)

	// 创建 MySQL 配置
	mysqlConfig := canal.MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		Username: "test",
		Password: "test",
		ServerID: 12345,
	}

	// 创建 MySQLBinlogSlave
	binlogSlave, err := canal.NewMySQLBinlogSlave(mysqlConfig, eventSink, logger)
	if err != nil {
		t.Fatalf("Failed to create MySQLBinlogSlave: %v", err)
	}

	// 添加监听表
	binlogSlave.AddWatchTable("test", "users")

	// 设置事件类型
	binlogSlave.SetEventTypes([]canal.EventType{canal.EventTypeInsert, canal.EventTypeUpdate})

	// 启动事件接收器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eventSink.Start(ctx); err != nil {
		t.Fatalf("Failed to start event sink: %v", err)
	}
	defer eventSink.Stop()

	// 模拟大量事件发送到事件接收器
	// 这会测试事件接收器的缓冲区是否会导致阻塞
	go func() {
		for i := 0; i < 200; i++ {
			event := &canal.Event{
				ID:        "test-event-" + string(rune(i)),
				Schema:    "test",
				Table:     "users",
				EventType: canal.EventTypeInsert,
				Timestamp: time.Now(),
				AfterData: &canal.RowData{
					Columns: []canal.Column{
						{Name: "id", Type: "int", Value: i},
						{Name: "name", Type: "varchar", Value: "user" + string(rune(i))},
					},
				},
			}
			eventSink.SendEvent(event)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// 等待一段时间让事件开始发送
	time.Sleep(100 * time.Millisecond)

	// 获取统计信息
	stats := binlogSlave.GetStats()
	t.Logf("Stats after sending events: %v", stats)

	// 检查是否正在运行
	if binlogSlave.IsRunning() {
		t.Log("Binlog slave is running")
	} else {
		t.Log("Binlog slave is not running")
	}
}

// TestMySQLBinlogSlaveSlowEventProcessing 测试 MySQLBinlogSlave 在慢速事件处理时的行为
func TestMySQLBinlogSlaveSlowEventProcessing(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlaveSlowEventProcessing] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := canal.NewDefaultEventSink(logger)

	// 创建 MySQL 配置
	mysqlConfig := canal.MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		Username: "test",
		Password: "test",
		ServerID: 12345,
	}

	// 创建 MySQLBinlogSlave
	binlogSlave, err := canal.NewMySQLBinlogSlave(mysqlConfig, eventSink, logger)
	if err != nil {
		t.Fatalf("Failed to create MySQLBinlogSlave: %v", err)
	}

	// 添加监听表
	binlogSlave.AddWatchTable("test", "users")

	// 设置事件类型
	binlogSlave.SetEventTypes([]canal.EventType{canal.EventTypeInsert, canal.EventTypeUpdate})

	// 启动事件接收器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eventSink.Start(ctx); err != nil {
		t.Fatalf("Failed to start event sink: %v", err)
	}
	defer eventSink.Stop()

	// 模拟慢速事件处理
	go func() {
		for i := 0; i < 50; i++ {
			event := &canal.Event{
				ID:        "test-event-" + string(rune(i)),
				Schema:    "test",
				Table:     "users",
				EventType: canal.EventTypeInsert,
				Timestamp: time.Now(),
				AfterData: &canal.RowData{
					Columns: []canal.Column{
						{Name: "id", Type: "int", Value: i},
						{Name: "name", Type: "varchar", Value: "user" + string(rune(i))},
					},
				},
			}
			eventSink.SendEvent(event)

			// 模拟慢速处理
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 等待一段时间让事件开始发送
	time.Sleep(2 * time.Second)

	// 获取统计信息
	stats := binlogSlave.GetStats()
	t.Logf("Stats after slow event processing: %v", stats)

	// 检查是否正在运行
	if binlogSlave.IsRunning() {
		t.Log("Binlog slave is running")
	} else {
		t.Log("Binlog slave is not running")
	}
}

// TestMySQLBinlogSlaveContextCancellation 测试 MySQLBinlogSlave 在上下文取消时的行为
func TestMySQLBinlogSlaveContextCancellation(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlaveContextCancellation] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := canal.NewDefaultEventSink(logger)

	// 创建 MySQL 配置
	mysqlConfig := canal.MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		Username: "test",
		Password: "test",
		ServerID: 12345,
	}

	// 创建 MySQLBinlogSlave
	binlogSlave, err := canal.NewMySQLBinlogSlave(mysqlConfig, eventSink, logger)
	if err != nil {
		t.Fatalf("Failed to create MySQLBinlogSlave: %v", err)
	}

	// 添加监听表
	binlogSlave.AddWatchTable("test", "users")

	// 设置事件类型
	binlogSlave.SetEventTypes([]canal.EventType{canal.EventTypeInsert, canal.EventTypeUpdate})

	// 启动事件接收器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eventSink.Start(ctx); err != nil {
		t.Fatalf("Failed to start event sink: %v", err)
	}
	defer eventSink.Stop()

	// 创建带超时的上下文
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 启动 binlog 从节点
	startErr := binlogSlave.Start()
	if startErr != nil {
		t.Logf("Start failed as expected without real MySQL connection: %v", startErr)
	}

	// 等待上下文超时
	<-ctx.Done()

	// 停止 binlog 从节点
	stopErr := binlogSlave.Stop()
	if stopErr != nil {
		t.Errorf("Stop failed: %v", stopErr)
	}

	// 获取统计信息
	stats := binlogSlave.GetStats()
	t.Logf("Stats after context cancellation: %v", stats)

	// 检查状态
	if binlogSlave.IsRunning() {
		t.Error("Expected binlog slave to not be running after stop")
	}
}
