/*
 * @Author: lucklidi@126.com
 * @Date: 2025-08-21 16:58:47
 * @LastEditTime: 2025-08-23 16:09:35
 * @LastEditors: lucklidi@126.com
 * @Description:
 * Copyright (c) 2023 by pikachun
 */
package database

import (
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init 初始化数据库连接
func Init(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移数据表
	if err := migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

// migrate 执行数据库迁移
func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Task{},
		&EventLog{},
	)
}

// EventLog 事件日志模型
type EventLog struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	TaskID    uint      `json:"task_id" gorm:"not null;index"`
	Database  string    `json:"database" gorm:"not null;size:100"`
	Table     string    `json:"table" gorm:"not null;size:100"`
	EventType string    `json:"event_type" gorm:"not null;size:20"`
	Data      string    `json:"data" gorm:"type:text"`
	Status    string    `json:"status" gorm:"default:'pending';size:20"` // pending, success, failed
	Error     string    `json:"error" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	Task      Task      `json:"task" gorm:"foreignKey:TaskID"`
}

// Task 监听任务模型
type Task struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null;size:100"`
	Database    string         `json:"database" gorm:"not null;size:100"`
	Table       string         `json:"table" gorm:"not null;size:100"`
	EventTypes  string         `json:"event_types" gorm:"not null;size:200"` // INSERT,UPDATE,DELETE
	CallbackURL string         `json:"callback_url" gorm:"not null;size:500"`
	Status      string         `json:"status" gorm:"default:'active';size:20"` // active, inactive
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}

// TableName 指定表名
func (EventLog) TableName() string {
	return "event_logs"
}
