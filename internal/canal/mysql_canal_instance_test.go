package canal

import (
	"log"
	"os"
	"testing"

	"pikachun/internal/config"
)

// TestMySQLCanalInstanceLogging 测试 MySQLCanalInstance 的日志功能
func TestMySQLCanalInstanceLogging(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstanceLogging] ", log.LstdFlags|log.Lshortfile)

	// 创建配置
	cfg := &config.Config{
		Canal: config.CanalConfig{
			Host:     "localhost",
			Port:     3307,
			Username: "test",
			Password: "test",
			ServerID: 12345,
			Watch: config.WatchConfig{
				Databases:  []string{"test"},
				Tables:     []string{"users"},
				EventTypes: []string{"INSERT", "UPDATE", "DELETE"},
			},
		},
	}

	// 创建 MySQLCanalInstance
	instance, err := NewMySQLCanalInstance("test-instance", cfg, logger, nil)
	if err != nil {
		t.Logf("Expected creation to fail without real MySQL connection: %v", err)
	}

	// 测试获取状态（这会触发日志输出）
	status := instance.GetStatus()
	t.Logf("Instance status: %+v", status)

	// 测试 String 方法
	str := instance.String()
	t.Logf("String representation: %s", str)
}
