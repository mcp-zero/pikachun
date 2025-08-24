/*
 * @Author: lucklidi@126.com
 * @Date: 2025-08-21 18:01:23
 * @LastEditTime: 2025-08-24 21:04:40
 * @Description:
 * Copyright (c) 2023 by pikachun
 */
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pikachun/internal/config"
	"pikachun/internal/database"
	"pikachun/internal/server"
	"pikachun/internal/service"
)

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.Println("🔧 Starting Pikachun Enhanced with Canal Architecture...")

	// 加载配置
	log.Println("🔧 Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}
	log.Printf("✅ Configuration loaded successfully")

	// 初始化数据库
	log.Println("🔧 Initializing database...")
	db, err := database.Init(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	log.Printf("✅ Database initialized successfully")

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化任务服务
	log.Println("🔧 Initializing task service...")
	taskService := service.NewTaskService(db)
	log.Printf("✅ Task service initialized successfully")

	// 初始化增强的Canal服务
	log.Println("🔧 Initializing enhanced Canal service...")
	enhancedCanalService, err := service.NewEnhancedCanalService(cfg, db, taskService)
	if err != nil {
		log.Fatalf("❌ Failed to initialize enhanced Canal service: %v", err)
	}
	log.Printf("✅ Enhanced Canal service initialized successfully")

	// 启动增强的Canal服务
	log.Println("🔧 Starting enhanced Canal service...")
	if err := enhancedCanalService.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start enhanced Canal service: %v", err)
	}
	log.Printf("✅ Enhanced Canal service started successfully")

	// 创建增强的服务器
	log.Println("🔧 Creating enhanced server...")
	srv := NewEnhancedServer(cfg, taskService, enhancedCanalService)
	log.Printf("✅ Enhanced server created successfully")

	// 启动Web服务器
	log.Println("🔧 Starting Web server...")
	go func() {
		log.Printf("🚀 Web management interface started: http://%s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.Start(); err != nil {
			log.Printf("❌ Web server error: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("✅ Pikachun Enhanced service started successfully, press Ctrl+C to stop")
	<-sigChan

	log.Println("🛑 Shutting down service gracefully...")

	// 设置关闭超时
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 停止Canal服务
	log.Println("🔧 Stopping Canal service...")
	if err := enhancedCanalService.Stop(); err != nil {
		log.Printf("❌ Error stopping Canal service: %v", err)
	}
	log.Println("✅ Canal service stopped")

	// 取消主上下文
	cancel()

	// 等待所有协程结束或超时
	log.Println("🔧 Waiting for all goroutines to finish...")
	select {
	case <-shutdownCtx.Done():
		log.Println("❌ Shutdown timeout, force exit")
	case <-time.After(2 * time.Second):
		log.Println("✅ Service shutdown gracefully")
	}
}

// EnhancedServer 增强的服务器
type EnhancedServer struct {
	config               *config.Config
	taskService          *service.TaskService
	enhancedCanalService *service.EnhancedCanalService
	server               *server.Server
}

// NewEnhancedServer 创建增强的服务器
func NewEnhancedServer(
	cfg *config.Config,
	taskService *service.TaskService,
	enhancedCanalService *service.EnhancedCanalService,
) *EnhancedServer {
	// 创建适配器，将增强的Canal服务适配到原有接口
	canalAdapter := &CanalServiceAdapter{enhanced: enhancedCanalService}

	return &EnhancedServer{
		config:               cfg,
		taskService:          taskService,
		enhancedCanalService: enhancedCanalService,
		server:               server.New(cfg, taskService, canalAdapter),
	}
}

// Start 启动增强的服务器
func (s *EnhancedServer) Start() error {
	return s.server.Start()
}

// CanalServiceAdapter Canal服务适配器
type CanalServiceAdapter struct {
	enhanced *service.EnhancedCanalService
}

// Start 启动服务
func (a *CanalServiceAdapter) Start(ctx context.Context) error {
	// 增强服务已经在main中启动，这里返回nil
	return nil
}

// Stop 停止服务
func (a *CanalServiceAdapter) Stop() error {
	// 增强服务会在main中停止，这里不需要操作
	return nil
}

// CreateTask 创建任务
func (a *CanalServiceAdapter) CreateTask(task *database.Task) error {
	return a.enhanced.CreateTask(task)
}

// DeleteTask 删除任务
func (a *CanalServiceAdapter) DeleteTask(taskID uint) error {
	return a.enhanced.DeleteTask(taskID)
}

// GetStatus 获取状态
func (a *CanalServiceAdapter) GetStatus() map[string]interface{} {
	return a.enhanced.GetStatus()
}

// StopInstance 停止指定实例
func (a *CanalServiceAdapter) StopInstance(instanceID uint) error {
	return a.enhanced.StopInstance(instanceID)
}

// UpdateInstance 更新指定实例
func (a *CanalServiceAdapter) UpdateInstance(instanceID uint, task *database.Task) error {
	return a.enhanced.UpdateInstance(instanceID, task)
}
