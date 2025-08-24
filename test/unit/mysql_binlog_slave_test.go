package main

import (
	"log"
	"os"
	"testing"

	"pikachun/internal/canal"
)

// TestMySQLBinlogSlave 测试 MySQLBinlogSlave 的基本功能
func TestMySQLBinlogSlave(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlave] ", log.LstdFlags|log.Lshortfile)

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

	// 测试添加监听表
	binlogSlave.AddWatchTable("test", "users")
	binlogSlave.AddWatchTable("test", "orders")

	// 测试设置事件类型
	binlogSlave.SetEventTypes([]canal.EventType{canal.EventTypeInsert, canal.EventTypeUpdate})

	// 测试获取 binlog 位置
	pos := binlogSlave.GetBinlogPosition()
	if pos.Name == "" {
		t.Error("Expected binlog position name to be set")
	}

	// 测试是否正在运行（应该返回 false，因为我们还没有启动）
	if binlogSlave.IsRunning() {
		t.Error("Expected binlog slave to not be running")
	}

	// 测试获取统计信息
	stats := binlogSlave.GetStats()
	if stats == nil {
		t.Error("Expected stats to be non-nil")
	}

	// 测试 String 方法
	str := binlogSlave.String()
	if str == "" {
		t.Error("Expected String() to return non-empty string")
	}

	t.Logf("MySQLBinlogSlave String(): %s", str)
}

// TestMySQLBinlogSlaveStartStop 测试 MySQLBinlogSlave 的启动和停止
func TestMySQLBinlogSlaveStartStop(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlaveStartStop] ", log.LstdFlags|log.Lshortfile)

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

	// 启动应该失败，因为我们没有真实的 MySQL 连接
	err = binlogSlave.Start()
	if err == nil {
		t.Error("Expected Start() to fail without real MySQL connection")
	} else {
		t.Logf("Start() failed as expected: %v", err)
	}

	// 停止应该成功，即使没有启动
	err = binlogSlave.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
}

// TestMySQLBinlogSlaveWatchTables 测试监听表的添加和移除
func TestMySQLBinlogSlaveWatchTables(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlaveWatchTables] ", log.LstdFlags|log.Lshortfile)

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
	binlogSlave.AddWatchTable("test", "orders")
	binlogSlave.AddWatchTable("production", "products")

	// 移除一个监听表
	binlogSlave.RemoveWatchTable("test", "orders")

	// 获取统计信息
	stats := binlogSlave.GetStats()
	watchedTables := stats["watched_tables"].(int)
	if watchedTables != 2 {
		t.Errorf("Expected 2 watched tables, got %d", watchedTables)
	}

	t.Logf("Watched tables count: %d", watchedTables)
}

// TestMySQLBinlogSlaveEventTypes 测试事件类型的设置
func TestMySQLBinlogSlaveEventTypes(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlaveEventTypes] ", log.LstdFlags|log.Lshortfile)

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

	// 设置事件类型
	eventTypes := []canal.EventType{canal.EventTypeInsert, canal.EventTypeUpdate, canal.EventTypeDelete}
	binlogSlave.SetEventTypes(eventTypes)

	// 获取统计信息
	stats := binlogSlave.GetStats()
	t.Logf("Stats after setting event types: %v", stats)
}
