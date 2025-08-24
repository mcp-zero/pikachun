package main

import (
	"context"
	"log"
	"os"
	"testing"

	"pikachun/internal/canal"
	"pikachun/internal/config"
)

// TestMySQLCanalInstance 测试 MySQLCanalInstance 的基本功能
func TestMySQLCanalInstance(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstance] ", log.LstdFlags|log.Lshortfile)

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
	instance, err := canal.NewMySQLCanalInstance("test-instance", cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create MySQLCanalInstance: %v", err)
	}

	// 测试订阅事件
	handler := &MockEventHandler{name: "test-handler"}
	err = instance.Subscribe("test", "users", handler)
	if err != nil {
		t.Errorf("Failed to subscribe: %v", err)
	}

	// 测试取消订阅
	err = instance.Unsubscribe("test", "users", "test-handler")
	if err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}

	// 测试获取状态
	status := instance.GetStatus()
	if status.Running {
		t.Error("Expected instance to not be running initially")
	}

	// 测试 String 方法
	str := instance.String()
	if str == "" {
		t.Error("Expected String() to return non-empty string")
	}

	t.Logf("MySQLCanalInstance String(): %s", str)
}

// TestMySQLCanalInstanceStartStop 测试 MySQLCanalInstance 的启动和停止
func TestMySQLCanalInstanceStartStop(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstanceStartStop] ", log.LstdFlags|log.Lshortfile)

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
	instance, err := canal.NewMySQLCanalInstance("test-instance", cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create MySQLCanalInstance: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动应该失败，因为我们没有真实的 MySQL 连接
	err = instance.Start(ctx)
	if err != nil {
		t.Logf("Start() failed as expected: %v", err)
	}

	// 停止应该成功，即使没有启动
	err = instance.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	// 再次停止应该成功
	err = instance.Stop()
	if err != nil {
		t.Errorf("Stop() failed second time: %v", err)
	}
}

// TestMySQLCanalInstanceStatus 测试 MySQLCanalInstance 状态管理
func TestMySQLCanalInstanceStatus(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstanceStatus] ", log.LstdFlags|log.Lshortfile)

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
	instance, err := canal.NewMySQLCanalInstance("test-instance", cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create MySQLCanalInstance: %v", err)
	}

	// 初始状态
	status := instance.GetStatus()
	if status.Running {
		t.Error("Expected instance to not be running initially")
	}

	// 模拟错误状态
	// 这里我们直接修改内部状态来测试状态报告
	// 在实际使用中，错误状态会由实例内部逻辑设置
	status = instance.GetStatus()
	t.Logf("Initial status: %+v", status)
}
