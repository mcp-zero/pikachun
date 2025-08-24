package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	databaseCom "pikachun/internal/database"
)

// TaskService 任务服务
type TaskService struct {
	db *gorm.DB
}

// NewTaskService 创建任务服务实例
func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{db: db}
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(task *databaseCom.Task) error {
	// 验证事件类型
	if !s.validateEventTypes(task.EventTypes) {
		return errors.New("无效的事件类型，支持: INSERT, UPDATE, DELETE")
	}

	// 验证回调URL
	if task.CallbackURL == "" {
		return errors.New("回调URL不能为空")
	}

	return s.db.Create(task).Error
}

// GetTasks 获取任务列表
func (s *TaskService) GetTasks(page, pageSize int) ([]databaseCom.Task, int64, error) {
	var tasks []databaseCom.Task
	var total int64

	// 计算总数
	if err := s.db.Model(&databaseCom.Task{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.db.Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// CreateEventLog 创建事件日志
func (s *TaskService) CreateEventLog(taskID uint, database, table, eventType, data, status, errorMsg string) error {
	eventLog := &databaseCom.EventLog{
		TaskID:    taskID,
		Database:  database,
		Table:     table,
		EventType: eventType,
		Data:      data,
		Status:    status,
		Error:     errorMsg,
	}

	return s.db.Create(eventLog).Error
}

// GetTask 根据ID获取任务
func (s *TaskService) GetTask(id uint) (*databaseCom.Task, error) {
	var task databaseCom.Task
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTask 更新任务
func (s *TaskService) UpdateTask(id uint, updates *databaseCom.Task) error {
	// 验证事件类型
	if updates.EventTypes != "" && !s.validateEventTypes(updates.EventTypes) {
		return errors.New("无效的事件类型，支持: INSERT, UPDATE, DELETE")
	}

	return s.db.Model(&databaseCom.Task{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteTask 删除任务
func (s *TaskService) DeleteTask(id uint) error {
	return s.db.Delete(&databaseCom.Task{}, id).Error
}

// GetActiveTasks 获取活跃的任务
func (s *TaskService) GetActiveTasks() ([]databaseCom.Task, error) {
	var tasks []databaseCom.Task
	if err := s.db.Where("status = ?", "active").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// LogEvent 记录事件日志
func (s *TaskService) LogEvent(log *databaseCom.EventLog) error {
	return s.db.Create(log).Error
}

// GetEventLogs 获取事件日志
func (s *TaskService) GetEventLogs(taskID uint, page, pageSize int) ([]databaseCom.EventLog, int64, error) {
	var logs []databaseCom.EventLog
	var total int64

	query := s.db.Model(&databaseCom.EventLog{})
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Task").Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetEventLog 获取单个事件日志
func (s *TaskService) GetEventLog(id uint) (*databaseCom.EventLog, error) {
	var log databaseCom.EventLog
	if err := s.db.Preload("Task").First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// validateEventTypes 验证事件类型
func (s *TaskService) validateEventTypes(eventTypes string) bool {
	validTypes := map[string]bool{
		"INSERT": true,
		"UPDATE": true,
		"DELETE": true,
	}

	types := strings.Split(eventTypes, ",")
	for _, t := range types {
		t = strings.TrimSpace(strings.ToUpper(t))
		if !validTypes[t] {
			return false
		}
	}
	return true
}
