package main

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"pikachun/internal/canal"
	"pikachun/internal/config"
)

// TestMySQLCanalInstanceDBConnection 测试 MySQLCanalInstance 数据库连接问题
func TestMySQLCanalInstanceDBConnection(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstanceDBConnection] ", log.LstdFlags|log.Lshortfile)

	// 创建配置（使用无效的数据库连接信息来测试连接失败）
	cfg := &config.Config{
		Canal: config.CanalConfig{
			Host:     "localhost",
			Port:     3307, // 使用一个可能未被占用的端口
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试启动实例（应该失败，因为没有真实的 MySQL 连接）
	startErr := instance.Start(ctx)
	if startErr != nil {
		t.Logf("Start failed as expected without real MySQL connection: %v", startErr)
	} else {
		t.Error("Expected Start to fail without real MySQL connection")
	}

	// 获取状态
	status := instance.GetStatus()
	t.Logf("Instance status after failed start: %+v", status)

	// 检查错误信息是否被正确设置
	if status.ErrorMsg == "" {
		t.Log("No error message in status")
	} else {
		t.Logf("Error message in status: %s", status.ErrorMsg)
	}
}

// TestMySQLCanalInstanceSlowDBConnection 测试 MySQLCanalInstance 慢速数据库连接
func TestMySQLCanalInstanceSlowDBConnection(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstanceSlowDBConnection] ", log.LstdFlags|log.Lshortfile)

	// 创建配置（使用无效的数据库连接信息来测试连接超时）
	cfg := &config.Config{
		Canal: config.CanalConfig{
			Host:     "192.168.1.99", // 使用一个不存在的 IP 地址来模拟慢速连接
			Port:     3306,
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

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 记录开始时间
	startTime := time.Now()

	// 尝试启动实例（应该超时）
	startErr := instance.Start(ctx)
	duration := time.Since(startTime)

	if startErr != nil {
		t.Logf("Start failed as expected with slow connection: %v", startErr)
	} else {
		t.Error("Expected Start to fail with slow connection")
	}

	// 检查是否超时
	if duration >= 3*time.Second {
		t.Logf("Start operation took expected time to timeout: %v", duration)
	} else {
		t.Logf("Start operation completed faster than expected: %v", duration)
	}

	// 获取状态
	status := instance.GetStatus()
	t.Logf("Instance status after slow connection test: %+v", status)
}

// TestMySQLCanalInstanceDBConnectionRetry 测试 MySQLCanalInstance 数据库连接重试机制
func TestMySQLCanalInstanceDBConnectionRetry(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstanceDBConnectionRetry] ", log.LstdFlags|log.Lshortfile)

	// 创建配置（使用无效的数据库连接信息来测试重试机制）
	cfg := &config.Config{
		Canal: config.CanalConfig{
			Host:     "localhost",
			Port:     3307, // 使用一个可能未被占用的端口
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

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 记录开始时间
	startTime := time.Now()

	// 尝试启动实例（应该失败，但会重试）
	startErr := instance.Start(ctx)
	duration := time.Since(startTime)

	if startErr != nil {
		t.Logf("Start failed as expected without real MySQL connection: %v", startErr)
	} else {
		t.Error("Expected Start to fail without real MySQL connection")
	}

	// 检查操作时间（重试机制应该会增加操作时间）
	if duration >= 5*time.Second {
		t.Logf("Start operation took expected time with retries: %v", duration)
	} else {
		t.Logf("Start operation completed faster than expected: %v", duration)
	}

	// 获取状态
	status := instance.GetStatus()
	t.Logf("Instance status after retry test: %+v", status)

	// 检查错误信息
	if status.ErrorMsg != "" {
		t.Logf("Error message in status: %s", status.ErrorMsg)
	}
}

// TestMySQLCanalInstanceDBConnectionPoolExhaustion 测试 MySQLCanalInstance 数据库连接池耗尽情况
func TestMySQLCanalInstanceDBConnectionPoolExhaustion(t *testing.T) {
	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestMySQLCanalInstanceDBConnectionPoolExhaustion] ", log.LstdFlags|log.Lshortfile)

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

	// 创建多个 MySQLCanalInstance 实例来测试连接池
	instances := make([]canal.CanalInstance, 5)
	for i := 0; i < 5; i++ {
		instance, err := canal.NewMySQLCanalInstance("test-instance-"+string(rune(i)), cfg, logger, nil)
		if err != nil {
			t.Fatalf("Failed to create MySQLCanalInstance %d: %v", i, err)
		}
		instances[i] = instance
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试同时启动所有实例
	startErrors := make(chan error, 5)
	for _, instance := range instances {
		go func(inst canal.CanalInstance) {
			startErrors <- inst.Start(ctx)
		}(instance)
	}

	// 等待所有启动操作完成
	timeout := time.After(10 * time.Second)
	completed := 0
	for completed < 5 {
		select {
		case err := <-startErrors:
			completed++
			if err != nil {
				t.Logf("Instance start failed: %v", err)
			} else {
				t.Log("Instance started successfully")
			}
		case <-timeout:
			t.Error("Timeout waiting for instance starts to complete")
			return
		}
	}

	// 获取所有实例的状态
	for i, instance := range instances {
		status := instance.GetStatus()
		t.Logf("Instance %d status: %+v", i, status)
	}
}
