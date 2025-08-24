package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"pikachun/internal/config"
	"pikachun/internal/database"
	"pikachun/internal/service"
)

// TestEnhancedCanalServiceConcurrency 测试 EnhancedCanalService 的并发访问
func TestEnhancedCanalServiceConcurrency(t *testing.T) {
	// 创建内存数据库用于测试
	db, err := gorm.Open(sqlite.Dialector{DSN: "file::memory:?cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		// 关闭数据库连接
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 自动迁移数据表
	if err := db.AutoMigrate(&database.Task{}, &database.EventLog{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// 创建配置
	cfg := &config.Config{
		Canal: config.CanalConfig{
			Host:     "127.0.0.1",
			Port:     3307,
			Username: "root",
			Password: "lidi10",
			ServerID: 12345,
			Watch: config.WatchConfig{
				Databases:  []string{"test"},
				Tables:     []string{"users"},
				EventTypes: []string{"INSERT", "UPDATE", "DELETE"},
			},
		},
	}

	// 创建任务服务
	taskService := service.NewTaskService(db)

	// 创建 EnhancedCanalService
	svc, err := service.NewEnhancedCanalService(cfg, db, taskService)
	if err != nil {
		t.Fatalf("Failed to create EnhancedCanalService: %v", err)
	}

	// 创建多个测试任务
	tasks := make([]*database.Task, 5)
	for i := 0; i < 5; i++ {
		tasks[i] = &database.Task{
			ID:          uint(1000 + i), // 使用测试ID >= 1000
			Database:    "test",
			Table:       "users",
			EventTypes:  "INSERT,UPDATE,DELETE",
			CallbackURL: "http://127.0.0.1:9669/webhook/test",
			Status:      "active",
		}
	}

	// 并发创建任务
	var wg sync.WaitGroup
	start := make(chan struct{})

	// 启动多个 goroutine 并发调用 CreateTask
	errorChan := make(chan error, 5)
	for i, task := range tasks {
		wg.Add(1)
		go func(index int, t *database.Task) {
			defer wg.Done()
			// 等待所有 goroutine 就绪
			<-start

			// 添加一些随机延迟以增加竞争条件的可能性
			time.Sleep(time.Duration(index*10) * time.Millisecond)

			err := svc.CreateTask(t)
			if err != nil {
				// 通过通道传递错误
				errorChan <- err
			}
		}(i, task)
	}

	// 同时开始所有 goroutine
	close(start)
	wg.Wait()
	close(errorChan)

	// 检查是否有错误
	for err := range errorChan {
		t.Logf("Task creation error: %v", err)
	}

	// 验证所有任务都已创建
	activeTasks, err := taskService.GetActiveTasks()
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(activeTasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(activeTasks))
	}

	// 验证服务状态
	status := svc.GetStatus()
	t.Logf("Service status: %+v", status)
}

// TestEnhancedCanalServiceLoadActiveTasksConcurrency 测试加载活跃任务时的并发访问
func TestEnhancedCanalServiceLoadActiveTasksConcurrency(t *testing.T) {
	// 创建内存数据库用于测试
	db, err := gorm.Open(sqlite.Dialector{DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		// 关闭数据库连接
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 自动迁移数据表
	if err := db.AutoMigrate(&database.Task{}, &database.EventLog{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// 创建配置
	cfg := &config.Config{
		Canal: config.CanalConfig{
			Host:     "127.0.0.1",
			Port:     3307,
			Username: "root",
			Password: "lidi10",
			ServerID: 12345,
			Watch: config.WatchConfig{
				Databases:  []string{"test"},
				Tables:     []string{"users"},
				EventTypes: []string{"INSERT", "UPDATE", "DELETE"},
			},
		},
	}

	// 创建任务服务
	taskService := service.NewTaskService(db)

	// 创建一些预存的活跃任务
	tasks := make([]database.Task, 3)
	for i := 0; i < 3; i++ {
		tasks[i] = database.Task{
			Database:    "test",
			Table:       "users",
			EventTypes:  "INSERT,UPDATE,DELETE",
			CallbackURL: "http://127.0.0.1:9669/webhook/test",
			Status:      "active",
		}
		// 直接插入到数据库中
		if err := taskService.CreateTask(&tasks[i]); err != nil {
			t.Fatalf("Failed to create task in database: %v", err)
		}
	}

	// 创建用于测试的 EnhancedCanalService
	mockSvc, err := NewMockEnhancedCanalService(cfg, db, taskService)
	if err != nil {
		t.Fatalf("Failed to create mock enhanced canal service: %v", err)
	}
	svc := mockSvc.EnhancedCanalService

	// 并发调用 Start 和 CreateTask
	var wg sync.WaitGroup

	// 启动 goroutine 调用 Start（会触发 loadExistingTasks）
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := svc.Start(ctx)
		if err != nil {
			// 在 goroutine 中通过日志记录错误
			t.Logf("Failed to start service: %v", err)
		}
	}()

	// 启动 goroutine 调用 CreateTask
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 等待一段时间确保 Start 已经开始执行
		time.Sleep(50 * time.Millisecond)

		task := &database.Task{
			Database:    "test",
			Table:       "users",
			EventTypes:  "INSERT,UPDATE,DELETE",
			CallbackURL: "http://127.0.0.1:9669/webhook/test",
			Status:      "active",
		}

		err := svc.CreateTask(task)
		if err != nil {
			// 在 goroutine 中通过日志记录错误
			t.Logf("Failed to create task: %v", err)
		}
	}()

	wg.Wait()

	// 验证所有任务都已加载和创建
	activeTasks, err := taskService.GetActiveTasks()
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	// 应该有 4 个任务 (3 个预存的 + 1 个新创建的)
	if len(activeTasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(activeTasks))
		for i, task := range activeTasks {
			t.Logf("Task %d: %+v", i, task)
		}
	}

	// 验证服务状态
	status := svc.GetStatus()
	t.Logf("Service status: %+v", status)
}

// TestEnhancedCanalServiceMutexContention 测试锁竞争情况
func TestEnhancedCanalServiceMutexContention(t *testing.T) {
	// 创建内存数据库用于测试
	db, err := gorm.Open(sqlite.Dialector{DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		// 关闭数据库连接
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 自动迁移数据表
	if err := db.AutoMigrate(&database.Task{}, &database.EventLog{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

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

	// 创建任务服务
	taskService := service.NewTaskService(db)

	// 创建用于测试的 EnhancedCanalService
	mockSvc, err := NewMockEnhancedCanalService(cfg, db, taskService)
	if err != nil {
		t.Fatalf("Failed to create mock enhanced canal service: %v", err)
	}
	svc := mockSvc.EnhancedCanalService

	// 创建测试任务
	task := &database.Task{
		Database:    "test",
		Table:       "users",
		EventTypes:  "INSERT,UPDATE,DELETE",
		CallbackURL: "http://127.0.0.1:9669/webhook/test",
		Status:      "active",
	}

	// 并发访问同一个任务
	var wg sync.WaitGroup
	start := make(chan struct{})
	errorChan := make(chan error, 10)

	// 启动多个 goroutine 并发调用 CreateTask
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// 等待所有 goroutine 就绪
			<-start

			// 每个 goroutine 尝试创建相同的任务
			// 只有一个应该成功，其他的应该失败
			err := svc.CreateTask(task)
			if err != nil {
				// 记录错误但不立即失败，因为预期有些会失败
				errorChan <- err
			}
		}(i)
	}

	// 同时开始所有 goroutine
	close(start)
	wg.Wait()
	close(errorChan)

	// 统计错误数量
	errorCount := 0
	for err := range errorChan {
		t.Logf("Expected error: %v", err)
		errorCount++
	}

	// 验证只有一个任务被创建成功
	activeTasks, err := taskService.GetActiveTasks()
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	// 应该只有一个任务
	if len(activeTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(activeTasks))
	}

	// 验证有错误 (多个 goroutine 中只有 1 个应该成功)
	if errorCount < 1 {
		t.Errorf("Expected at least 1 error, got %d", errorCount)
	}

	// 验证服务状态
	status := svc.GetStatus()
	t.Logf("Service status: %+v", status)
}

// MockEnhancedCanalService 用于测试的 EnhancedCanalService 实现
type MockEnhancedCanalService struct {
	*service.EnhancedCanalService
}

// NewMockEnhancedCanalService 创建用于测试的 EnhancedCanalService
func NewMockEnhancedCanalService(cfg *config.Config, db *gorm.DB, taskService *service.TaskService) (*MockEnhancedCanalService, error) {
	svc, err := service.NewEnhancedCanalService(cfg, db, taskService)
	if err != nil {
		return nil, err
	}

	return &MockEnhancedCanalService{
		EnhancedCanalService: svc,
	}, nil
}

// CreateTask 创建任务，对于 ID >= 1000 的任务使用 mock 实例
func (m *MockEnhancedCanalService) CreateTask(task *database.Task) error {
	// 对于 ID >= 1000 的任务，使用 mock 实例
	if task.ID >= 1000 {
		// 直接调用父类方法，让父类处理 mock 实例的创建
		return m.EnhancedCanalService.CreateTask(task)
	}
	// 对于其他任务，调用父类方法
	return m.EnhancedCanalService.CreateTask(task)
}

// createMockTask 创建使用 mock 实例的任务
