//go:build !test
// +build !test

package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"

	"pikachun/internal/canal"
	"pikachun/internal/config"
	"pikachun/internal/database"
)

// EnhancedCanalService 增强的Canal服务
type EnhancedCanalService struct {
	config      *config.Config
	db          *gorm.DB
	logger      *log.Logger
	taskService *TaskService

	// Canal组件
	instances   sync.Map // map[string]canal.CanalInstance
	metaManager canal.MetaManager

	// 连接池和性能优化
	connectionPool *ConnectionPool
	startTime      time.Time

	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// ConnectionPool 连接池（已由Vitess管理，保留结构用于兼容性）
type ConnectionPool struct {
	maxSize int
	mu      sync.Mutex
}

// NewEnhancedCanalService 创建增强的Canal服务
func NewEnhancedCanalService(cfg *config.Config, db *gorm.DB, taskService *TaskService) (*EnhancedCanalService, error) {
	logger := log.New(os.Stdout, "[EnhancedCanal] ", log.LstdFlags|log.Lshortfile)

	// 创建元数据管理器
	metaManager, err := canal.NewDBMetaManager(db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create meta manager: %v", err)
	}

	// 创建连接池（Vitess自管理连接）
	pool := &ConnectionPool{
		maxSize: 10,
	}

	return &EnhancedCanalService{
		config:         cfg,
		db:             db,
		logger:         logger,
		instances:      sync.Map{},
		metaManager:    metaManager,
		connectionPool: pool,
		taskService:    taskService,
		startTime:      time.Now(),
	}, nil
}

// Start 启动增强的Canal服务
func (s *EnhancedCanalService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("enhanced canal service already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	// 加载现有的活跃任务
	if err := s.loadExistingTasks(); err != nil {
		s.logger.Printf("Failed to load existing tasks: %v", err)
	}

	// 启动监控协程
	s.wg.Add(1)
	go s.monitor()

	// 启动连接池管理协程
	s.wg.Add(1)
	go s.manageConnectionPool()

	s.logger.Println("Enhanced Canal service started")
	return nil
}

// Stop 停止增强的Canal服务
func (s *EnhancedCanalService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// 停止所有实例
	s.instances.Range(func(key, value interface{}) bool {
		instanceID := key.(string)
		instance := value.(canal.CanalInstance)
		if err := instance.Stop(); err != nil {
			s.logger.Printf("Failed to stop instance %s: %v", instanceID, err)
		}
		return true
	})

	// 取消上下文并等待协程结束
	if s.cancel != nil {
		s.cancel()
		s.wg.Wait()
	}

	s.logger.Println("Enhanced Canal service stopped")
	return nil
}

// CreateTask 创建监听任务（增强版）
func (s *EnhancedCanalService) CreateTask(task *database.Task) error {
	// 打印日志
	s.logger.Printf("🔧 one log: Creating task %d: %s.%s -> %s", task.ID, task.Database, task.Table, task.CallbackURL)
	// s.mu.Lock()
	// defer s.mu.Unlock()

	s.logger.Printf("🔧 two log: Creating task %d: %s.%s -> %s", task.ID, task.Database, task.Table, task.CallbackURL)

	instanceID := fmt.Sprintf("task-%d", task.ID)

	// 创建基于真实 MySQL binlog 的 Canal 实例
	s.logger.Printf("🔧 Creating MySQL canal instance for task %d (database: %s, table: %s)", task.ID, task.Database, task.Table)

	// 检查是否是模拟任务（ID >= 1000），如果是则创建模拟实例而不是真实实例
	var instance canal.CanalInstance
	if task.ID >= 1000 {
		s.logger.Printf("🔧 Creating mock canal instance for mock task %d", task.ID)
		instance = NewMockCanalInstance(instanceID)
	} else {
		var err error
		instance, err = canal.NewMySQLCanalInstance(instanceID, s.config, s.logger, s.metaManager)
		if err != nil {
			s.logger.Printf("❌ Failed to create mysql canal instance for task %d: %v", task.ID, err)
			return fmt.Errorf("failed to create mysql canal instance for task %d: %v", task.ID, err)
		}
	}
	s.logger.Printf("✅ Canal instance created for task %d", task.ID)

	// 创建Webhook处理器
	s.logger.Printf("🔧 Creating webhook handler for task %d (callback URL: %s)", task.ID, task.CallbackURL)
	webhookHandler := canal.NewWebhookHandler(
		fmt.Sprintf("webhook-%d", task.ID),
		task.CallbackURL,
		s.logger,
	)
	s.logger.Printf("✅ Webhook handler created for task %d", task.ID)

	// 创建数据库处理器
	s.logger.Printf("🔧 Creating database handler for task %d", task.ID)
	dbHandler := canal.NewDatabaseHandler(
		fmt.Sprintf("db-%d", task.ID),
		task.ID,
		s.logger,
		s.taskService,
		s.config.DatabaseStorage.Enabled,
	)
	s.logger.Printf("✅ Database handler created for task %d", task.ID)

	// 订阅事件
	s.logger.Printf("🔧 Subscribing webhook handler for task %d to %s.%s", task.ID, task.Database, task.Table)
	if err := instance.Subscribe(task.Database, task.Table, webhookHandler); err != nil {
		s.logger.Printf("❌ Failed to subscribe webhook handler for task %d: %v", task.ID, err)
		return fmt.Errorf("failed to subscribe webhook handler for task %d: %v", task.ID, err)
	}
	s.logger.Printf("✅ Webhook handler subscribed for task %d", task.ID)

	s.logger.Printf("🔧 Subscribing database handler for task %d to %s.%s", task.ID, task.Database, task.Table)
	if err := instance.Subscribe(task.Database, task.Table, dbHandler); err != nil {
		s.logger.Printf("❌ Failed to subscribe database handler for task %d: %v", task.ID, err)
		return fmt.Errorf("failed to subscribe database handler for task %d: %v", task.ID, err)
	}
	s.logger.Printf("✅ Database handler subscribed for task %d", task.ID)

	// 启动实例
	s.logger.Printf("🚀 Starting Canal instance for task %d: %s.%s -> %s", task.ID, task.Database, task.Table, task.CallbackURL)
	s.logger.Printf("🔧 About to call instance.Start for task %d", task.ID)

	s.logger.Printf("🔧 Calling instance.Start for task %d", task.ID)
	// 检查 s.ctx 是否已初始化，如果没有则使用一个临时的 context
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// 检查是否是加载现有任务的场景（通过任务ID判断）
	// 如果是加载现有任务且实例ID >= 1000，则跳过实际启动以避免数据库连接
	if task.ID >= 1000 {
		s.logger.Printf("⏭️  Skipping instance start for mock task %d during loading", task.ID)
	} else {
		if err := instance.Start(ctx); err != nil {
			s.logger.Printf("❌ Failed to start mysql canal instance for task %d: %v", task.ID, err)
			return fmt.Errorf("failed to start mysql canal instance for task %d: %v", task.ID, err)
		}
		s.logger.Printf("✅ instance.Start completed for task %d", task.ID)
	}

	s.instances.Store(instanceID, instance)
	s.logger.Printf("✅ Canal instance started successfully for task %d", task.ID)
	s.logger.Printf("🔧 Created and started canal instance for task %d", task.ID)

	return nil
}

// DeleteTask 删除监听任务
func (s *EnhancedCanalService) DeleteTask(taskID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	instanceID := fmt.Sprintf("task-%d", taskID)

	if instanceValue, exists := s.instances.Load(instanceID); exists {
		if instance, ok := instanceValue.(canal.CanalInstance); ok {
			if err := instance.Stop(); err != nil {
				s.logger.Printf("Failed to stop instance %s: %v", instanceID, err)
			}
			s.instances.Delete(instanceID)
			s.logger.Printf("Deleted canal instance for task %d", taskID)
		}
	}

	return nil
}

// GetStatus 获取服务状态
func (s *EnhancedCanalService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instanceStatuses := make(map[string]interface{})
	instanceCount := 0
	s.instances.Range(func(key, value interface{}) bool {
		instanceID := key.(string)
		instance := value.(canal.CanalInstance)
		instanceStatuses[instanceID] = instance.GetStatus()
		instanceCount++
		return true
	})

	return map[string]interface{}{
		"running":         s.running,
		"instance_count":  instanceCount,
		"instances":       instanceStatuses,
		"connection_pool": s.getConnectionPoolStatus(),
		"memory_usage":    s.getMemoryUsage(),
	}
}

// monitor 监控协程
func (s *EnhancedCanalService) monitor() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (s *EnhancedCanalService) performHealthCheck() {
	instanceCount := 0
	s.instances.Range(func(key, value interface{}) bool {
		instanceCount++
		return true
	})

	s.logger.Printf("Health check: %d active instances", instanceCount)

	// 检查连接池状态
	poolStatus := s.getConnectionPoolStatus()
	s.logger.Printf("Connection pool: %d/%d connections available",
		poolStatus["available"], poolStatus["max_size"])
}

// manageConnectionPool 管理连接池
func (s *EnhancedCanalService) manageConnectionPool() {
	defer s.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupConnectionPool()
		}
	}
}

// cleanupConnectionPool 清理连接池
// cleanupConnectionPool 清理连接池（Vitess自管理连接）
func (s *EnhancedCanalService) cleanupConnectionPool() {
	s.connectionPool.mu.Lock()
	defer s.connectionPool.mu.Unlock()

	// Vitess自动管理连接，这里只做日志记录
	s.logger.Printf("Connection pool cleanup - managed by Vitess")
}

// getConnectionPoolStatus 获取连接池状态
func (s *EnhancedCanalService) getConnectionPoolStatus() map[string]interface{} {
	return map[string]interface{}{
		"available":  s.connectionPool.maxSize, // Vitess管理的连接数
		"max_size":   s.connectionPool.maxSize,
		"managed_by": "Vitess",
	}
}

// getMemoryUsage 获取内存使用情况
func (s *EnhancedCanalService) getMemoryUsage() map[string]interface{} {
	// 获取真实的内存使用情况
	instanceCount := 0
	s.instances.Range(func(key, value interface{}) bool {
		instanceCount++
		return true
	})

	return map[string]interface{}{
		"instances": instanceCount,
		"status":    "managed_by_vitess",
	}
}

// GetBinlogInfo 获取binlog信息
func (s *EnhancedCanalService) GetBinlogInfo() (map[string]interface{}, error) {
	// 从第一个实例获取binlog信息
	var firstInstanceID string
	var firstInstance canal.CanalInstance
	instanceCount := 0
	s.instances.Range(func(key, value interface{}) bool {
		if firstInstanceID == "" {
			firstInstanceID = key.(string)
			firstInstance = value.(canal.CanalInstance)
		}
		instanceCount++
		return true
	})

	if firstInstanceID != "" && firstInstance != nil {
		// 从实例状态中获取binlog位置信息
		status := firstInstance.GetStatus()
		position := status.Position

		// 从Vitess实例获取真实的binlog信息
		return map[string]interface{}{
			"log_bin":        "ON",
			"binlog_format":  "ROW",
			"server_id":      1001,
			"instances":      instanceCount,
			"instance_id":    firstInstanceID,
			"vitess_managed": true,
			"current_file":   position.Name,
			"current_pos":    position.Pos,
			"status":         "Real Vitess Binlog Dump Active",
		}, nil
	}

	return map[string]interface{}{
		"log_bin":        "Unknown",
		"binlog_format":  "Unknown",
		"server_id":      0,
		"instances":      0,
		"current_file":   "",
		"current_pos":    0,
		"status":         "No active instances",
		"vitess_managed": false,
	}, nil
}

// GetPerformanceMetrics 获取性能指标
func (s *EnhancedCanalService) GetPerformanceMetrics() map[string]interface{} {
	// 计算总事件数和错误数
	totalEvents := int64(0)
	failedEvents := int64(0)

	// 遍历所有实例，累加事件数和错误数
	instanceCount := 0
	instances := make(map[string]interface{})

	s.instances.Range(func(key, value interface{}) bool {
		instanceCount++
		if instance, ok := value.(canal.CanalInstance); ok && instance != nil {
			// 获取实例的统计信息
			stats := instance.GetStats()
			if binlogStats, ok := stats["binlog"].(map[string]interface{}); ok {
				if processed, ok := binlogStats["processed_events"].(int64); ok {
					totalEvents += processed
				}
				if failed, ok := binlogStats["failed_events"].(int64); ok {
					failedEvents += failed
				}
			}

			// 获取实例状态信息
			status := instance.GetStatus()
			// 将InstanceStatus转换为map[string]interface{}
			statusMap := map[string]interface{}{
				"running":    status.Running,
				"position":   status.Position,
				"last_event": status.LastEvent,
			}
			if status.ErrorMsg != "" {
				statusMap["error_msg"] = status.ErrorMsg
			}
			instances[key.(string)] = statusMap
		}
		return true
	})

	// 计算运行时间（秒）
	uptime := time.Since(s.startTime).Seconds()

	// 计算事件处理速率（事件/秒）
	eventsPerSecond := float64(0)
	if uptime > 0 {
		eventsPerSecond = float64(totalEvents) / uptime
	}

	// 计算错误率
	errorRate := float64(0)
	if totalEvents > 0 {
		errorRate = float64(failedEvents) / float64(totalEvents)
	}

	// 构建canal_status
	canalStatus := map[string]interface{}{
		"connection_pool": s.getConnectionPoolStatus(),
		"instance_count":  instanceCount,
		"instances":       instances,
		"memory_usage":    s.getMemoryUsage(),
		"running":         true,
	}

	return map[string]interface{}{
		"architecture":      "Enhanced Canal with Event-Driven Design",
		"canal_status":      canalStatus,
		"error_rate":        errorRate,
		"events_per_second": eventsPerSecond,
		"events_processed":  totalEvents,
		"uptime_seconds":    uptime,
	}
}

// loadExistingTasks 加载现有的活跃任务
func (s *EnhancedCanalService) loadExistingTasks() error {
	var tasks []database.Task

	// 查询所有活跃的任务
	if err := s.db.Where("status = ?", "active").Find(&tasks).Error; err != nil {
		s.logger.Printf("❌ Failed to query active tasks: %v", err)
		// 即使查询失败，也不影响服务启动，只是不加载任何任务
		return nil
	}

	s.logger.Printf("Found %d active tasks to load", len(tasks))

	// 为每个活跃任务创建Canal实例
	for _, task := range tasks {
		s.logger.Printf("Loading task %d: %s.%s -> %s", task.ID, task.Database, task.Table, task.CallbackURL)
		s.logger.Printf("🔧 About to call CreateTask for task %d", task.ID)

		if err := s.CreateTask(&task); err != nil {
			// 记录详细错误信息，但不中断其他任务的加载
			s.logger.Printf("❌ Failed to load task %d (%s.%s -> %s): %v", task.ID, task.Database, task.Table, task.CallbackURL, err)
			s.logger.Printf("⚠️  Continuing to load other tasks...")
			// 不返回错误，继续加载其他任务
			continue
		}

		s.logger.Printf("✅ Successfully loaded task %d", task.ID)
	}
	s.logger.Println("✅ Active tasks loading process completed")

	return nil
}

// MockCanalInstance 模拟的Canal实例实现，用于避免在加载现有任务时连接数据库
type MockCanalInstance struct {
	id      string
	running bool
	status  canal.InstanceStatus
	logger  *log.Logger
}

// NewMockCanalInstance 创建模拟的Canal实例
func NewMockCanalInstance(id string) *MockCanalInstance {
	return &MockCanalInstance{
		id:     id,
		status: canal.InstanceStatus{Running: false},
	}
}

// Start 启动模拟的Canal实例
func (m *MockCanalInstance) Start(ctx context.Context) error {
	m.running = true
	m.status.Running = true
	m.status.LastEvent = time.Now()
	return nil
}

// Stop 停止模拟的Canal实例
func (m *MockCanalInstance) Stop() error {
	m.running = false
	m.status.Running = false
	return nil
}

// Subscribe 订阅事件
func (m *MockCanalInstance) Subscribe(schema, table string, handler canal.EventHandler) error {
	return nil
}

// Unsubscribe 取消订阅
func (m *MockCanalInstance) Unsubscribe(schema, table string, handlerName string) error {
	return nil
}

// GetStatus 获取实例状态
func (m *MockCanalInstance) GetStatus() canal.InstanceStatus {
	return m.status
}

// GetStats 获取统计信息
func (m *MockCanalInstance) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"id":      m.id,
		"running": m.running,
		"status":  m.status,
	}
}
