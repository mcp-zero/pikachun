package canal

import (
	"log"
	"os"
	"testing"
)

// TestMySQLBinlogSlaveLogging 测试 MySQLBinlogSlave 的日志功能
func TestMySQLBinlogSlaveLogging(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLBinlogSlaveLogging] ", log.LstdFlags|log.Lshortfile)

	// 创建事件接收器
	eventSink := NewDefaultEventSink(logger)

	// 创建 MySQL 配置
	mysqlConfig := MySQLConfig{
		Host:     "localhost",
		Port:     3307,
		Username: "test",
		Password: "test",
		ServerID: 12345,
	}

	// 创建 MySQLBinlogSlave
	binlogSlave, err := NewMySQLBinlogSlave(mysqlConfig, eventSink, logger)
	if err != nil {
		t.Fatalf("Failed to create MySQLBinlogSlave: %v", err)
	}

	// 测试获取 binlog 位置（这会触发日志输出）
	pos := binlogSlave.GetBinlogPosition()
	t.Logf("Binlog position: %+v", pos)

	// 测试是否正在运行
	running := binlogSlave.IsRunning()
	t.Logf("Is running: %v", running)

	// 测试获取统计信息
	stats := binlogSlave.GetStats()
	t.Logf("Stats: %+v", stats)

	// 测试 String 方法
	str := binlogSlave.String()
	t.Logf("String representation: %s", str)
}
