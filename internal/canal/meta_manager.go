package canal

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// DBMetaManager 基于数据库的元数据管理器
type DBMetaManager struct {
	db     *gorm.DB
	logger *log.Logger
	mu     sync.RWMutex
	cache  map[string]Position   // instanceID -> Position
	tables map[string]*TableMeta // schema.table -> TableMeta
}

// BinlogPosition binlog 位置记录
type BinlogPosition struct {
	ID         uint      `gorm:"primarykey"`
	InstanceID string    `gorm:"uniqueIndex;size:100;not null"`
	Filename   string    `gorm:"size:255"`
	Position   uint32    `gorm:"not null"`
	GTIDSet    string    `gorm:"type:text"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// TableMetadata 表元数据记录
type TableMetadata struct {
	ID        uint      `gorm:"primarykey"`
	Schema    string    `gorm:"size:100;not null"`
	Table     string    `gorm:"size:100;not null"`
	Columns   string    `gorm:"type:text"` // JSON 格式存储列信息
	Types     string    `gorm:"type:text"` // JSON 格式存储类型信息
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TableName 指定表名
func (BinlogPosition) TableName() string {
	return "binlog_positions"
}

// TableName 指定表名
func (TableMetadata) TableName() string {
	return "table_metadata"
}

// NewDBMetaManager 创建数据库元数据管理器
func NewDBMetaManager(db *gorm.DB, logger *log.Logger) (*DBMetaManager, error) {
	manager := &DBMetaManager{
		db:     db,
		logger: logger,
		cache:  make(map[string]Position),
		tables: make(map[string]*TableMeta),
	}

	if err := db.AutoMigrate(&BinlogPosition{}, &TableMetadata{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate tables: %v", err)
	}

	// 加载缓存
	if err := manager.loadCache(); err != nil {
		return nil, fmt.Errorf("failed to load cache: %v", err)
	}

	return manager, nil
}

// loadCache 加载缓存
func (m *DBMetaManager) loadCache() error {
	// 加载 binlog 位置缓存
	var positions []BinlogPosition
	if err := m.db.Find(&positions).Error; err != nil {
		return fmt.Errorf("failed to load binlog positions: %v", err)
	}

	m.mu.Lock()
	for _, pos := range positions {
		m.cache[pos.InstanceID] = Position{
			Name:    pos.Filename,
			Pos:     pos.Position,
			GTIDSet: pos.GTIDSet,
		}
	}
	m.mu.Unlock()

	// 加载表元数据缓存
	var tables []TableMetadata
	if err := m.db.Find(&tables).Error; err != nil {
		return fmt.Errorf("failed to load table metadata: %v", err)
	}

	m.mu.Lock()
	for _, table := range tables {
		key := fmt.Sprintf("%s.%s", table.Schema, table.Table)

		var columns []string
		var types []string

		if err := json.Unmarshal([]byte(table.Columns), &columns); err == nil {
			json.Unmarshal([]byte(table.Types), &types)
		}

		m.tables[key] = &TableMeta{
			Schema:  table.Schema,
			Table:   table.Table,
			Columns: columns,
			Types:   types,
		}
	}
	m.mu.Unlock()

	return nil
}

// SavePosition 保存 binlog 位置
func (m *DBMetaManager) SavePosition(instanceID string, pos Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 记录日志
	log.Printf("💾 Saving binlog position for instance %s: %s:%d", instanceID, pos.Name, pos.Pos)

	// 更新缓存
	m.cache[instanceID] = pos
	log.Printf("✅ Updated cache for instance %s", instanceID)

	// 保存到数据库
	binlogPos := BinlogPosition{
		InstanceID: instanceID,
		Filename:   pos.Name,
		Position:   pos.Pos,
		GTIDSet:    pos.GTIDSet,
	}
	log.Printf("🔧 Preparing to save position to database")

	// 使用 UPSERT 操作
	result := m.db.Where("instance_id = ?", instanceID).First(&BinlogPosition{})
	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		log.Printf("🆕 Creating new binlog position record for instance %s", instanceID)
		if err := m.db.Create(&binlogPos).Error; err != nil {
			log.Printf("❌ Failed to create binlog position: %v", err)
			return fmt.Errorf("failed to create binlog position: %v", err)
		}
		log.Printf("✅ Created new binlog position record for instance %s", instanceID)
	} else {
		// 更新现有记录
		log.Printf("🔄 Updating existing binlog position record for instance %s", instanceID)
		if err := m.db.Where("instance_id = ?", instanceID).Updates(&binlogPos).Error; err != nil {
			log.Printf("❌ Failed to update binlog position: %v", err)
			return fmt.Errorf("failed to update binlog position: %v", err)
		}
		log.Printf("✅ Updated binlog position record for instance %s", instanceID)
	}

	log.Printf("🎉 Successfully saved binlog position for instance %s", instanceID)
	return nil
}

// LoadPosition 加载 binlog 位置
func (m *DBMetaManager) LoadPosition(instanceID string) (Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 记录日志
	m.logger.Printf("🔍 Loading binlog position for instance %s", instanceID)

	// 先从缓存查找
	if pos, exists := m.cache[instanceID]; exists {
		m.logger.Printf("✅ Found position in cache for instance %s: %s:%d", instanceID, pos.Name, pos.Pos)
		return pos, nil
	}

	m.logger.Printf("🔄 Position not found in cache for instance %s, loading from database", instanceID)
	// 从数据库获取
	var binlogPos BinlogPosition
	if err := m.db.Where("instance_id = ?", instanceID).First(&binlogPos).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认位置
			m.logger.Printf("⚠️  No position found in database for instance %s, returning default position", instanceID)
			return Position{Name: "", Pos: 4}, nil
		}
		m.logger.Printf("❌ Failed to load binlog position from database for instance %s: %v", instanceID, err)
		return Position{}, fmt.Errorf("failed to load binlog position: %v", err)
	}

	pos := Position{
		Name:    binlogPos.Filename,
		Pos:     binlogPos.Position,
		GTIDSet: binlogPos.GTIDSet,
	}

	m.logger.Printf("✅ Loaded position from database for instance %s: %s:%d", instanceID, pos.Name, pos.Pos)

	// 更新缓存
	m.mu.Lock()
	m.cache[instanceID] = pos
	m.mu.Unlock()

	m.logger.Printf("💾 Updated cache for instance %s", instanceID)

	return pos, nil
}

// LoadTableMeta 加载表元数据
func (m *DBMetaManager) LoadTableMeta(schema, table string) (*TableMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 记录日志
	m.logger.Printf("🔍 Loading table metadata for %s.%s", schema, table)

	// 先从缓存查找
	key := fmt.Sprintf("%s.%s", schema, table)
	if meta, exists := m.tables[key]; exists {
		m.logger.Printf("✅ Found table metadata in cache for %s.%s", schema, table)
		return meta, nil
	}

	m.logger.Printf("🔄 Table metadata not found in cache for %s.%s, loading from database", schema, table)

	// 从数据库获取
	var tableMeta TableMetadata
	if err := m.db.Where("`schema` = ? AND `table` = ?", schema, table).First(&tableMeta).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			m.logger.Printf("⚠️  No table metadata found in database for %s.%s", schema, table)
			return nil, nil
		}
		m.logger.Printf("❌ Failed to load table metadata from database for %s.%s: %v", schema, table, err)
		return nil, fmt.Errorf("failed to load table metadata: %v", err)
	}

	// 解析列信息
	var columns []string
	var types []string
	if err := json.Unmarshal([]byte(tableMeta.Columns), &columns); err != nil {
		m.logger.Printf("❌ Failed to unmarshal columns for %s.%s: %v", schema, table, err)
		return nil, fmt.Errorf("failed to unmarshal columns: %v", err)
	}
	if err := json.Unmarshal([]byte(tableMeta.Types), &types); err != nil {
		m.logger.Printf("❌ Failed to unmarshal types for %s.%s: %v", schema, table, err)
		return nil, fmt.Errorf("failed to unmarshal types: %v", err)
	}

	meta := &TableMeta{
		Schema:  tableMeta.Schema,
		Table:   tableMeta.Table,
		Columns: columns,
		Types:   types,
	}

	m.logger.Printf("✅ Loaded table metadata from database for %s.%s with %d columns", schema, table, len(columns))

	// 更新缓存
	m.mu.Lock()
	m.tables[key] = meta
	m.mu.Unlock()

	m.logger.Printf("💾 Updated table metadata cache for %s.%s", schema, table)

	return meta, nil
}

// SaveTableMeta 保存表元数据
func (m *DBMetaManager) SaveTableMeta(schema, table string, meta *TableMeta) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Printf("💾 Saving table metadata for %s.%s with %d columns", schema, table, len(meta.Columns))

	key := fmt.Sprintf("%s.%s", schema, table)
	m.tables[key] = meta

	// 序列化列信息
	columnsJSON, err := json.Marshal(meta.Columns)
	if err != nil {
		m.logger.Printf("❌ Failed to marshal columns for %s.%s: %v", schema, table, err)
		return fmt.Errorf("failed to marshal columns: %v", err)
	}

	typesJSON, err := json.Marshal(meta.Types)
	if err != nil {
		m.logger.Printf("❌ Failed to marshal types for %s.%s: %v", schema, table, err)
		return fmt.Errorf("failed to marshal types: %v", err)
	}

	tableMeta := TableMetadata{
		Schema:  schema,
		Table:   table,
		Columns: string(columnsJSON),
		Types:   string(typesJSON),
	}

	// 使用 UPSERT 操作
	m.logger.Printf("🔄 Upserting table metadata for %s.%s to database", schema, table)
	result := m.db.Where("`schema` = ? AND `table` = ?", schema, table).First(&TableMetadata{})
	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		m.logger.Printf("🆕 Creating new table metadata record for %s.%s", schema, table)
		if err := m.db.Create(&tableMeta).Error; err != nil {
			m.logger.Printf("❌ Failed to create table metadata for %s.%s: %v", schema, table, err)
			return fmt.Errorf("failed to create table metadata: %v", err)
		}
	} else {
		// 更新现有记录
		m.logger.Printf("🔄 Updating existing table metadata record for %s.%s", schema, table)
		if err := m.db.Where("`schema` = ? AND `table` = ?", schema, table).Updates(&tableMeta).Error; err != nil {
			m.logger.Printf("❌ Failed to update table metadata for %s.%s: %v", schema, table, err)
			return fmt.Errorf("failed to update table metadata: %v", err)
		}
	}

	m.logger.Printf("✅ Successfully saved table metadata for %s.%s", schema, table)

	return nil
}

// GetAllPositions 获取所有 binlog 位置
func (m *DBMetaManager) GetAllPositions() map[string]Position {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]Position)
	for k, v := range m.cache {
		result[k] = v
	}
	return result
}

// GetAllTableMetas 获取所有表元数据
func (m *DBMetaManager) GetAllTableMetas() map[string]*TableMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*TableMeta)
	for k, v := range m.tables {
		result[k] = v
	}
	return result
}

// DeletePosition 删除 binlog 位置
func (m *DBMetaManager) DeletePosition(instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从缓存删除
	delete(m.cache, instanceID)

	// 从数据库删除
	if err := m.db.Where("instance_id = ?", instanceID).Delete(&BinlogPosition{}).Error; err != nil {
		return fmt.Errorf("failed to delete binlog position: %v", err)
	}

	return nil
}

// DeleteTableMeta 删除表元数据
func (m *DBMetaManager) DeleteTableMeta(schema, table string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s.%s", schema, table)

	// 从缓存删除
	delete(m.tables, key)

	// 从数据库删除
	if err := m.db.Where("`schema` = ? AND `table` = ?", schema, table).Delete(&TableMetadata{}).Error; err != nil {
		return fmt.Errorf("failed to delete table metadata: %v", err)
	}

	return nil
}

// Cleanup 清理过期数据
func (m *DBMetaManager) Cleanup(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// 清理过期的 binlog 位置记录
	if err := m.db.Where("updated_at < ?", cutoff).Delete(&BinlogPosition{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old binlog positions: %v", err)
	}

	// 清理过期的表元数据记录
	if err := m.db.Where("updated_at < ?", cutoff).Delete(&TableMetadata{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old table metadata: %v", err)
	}

	// 重新加载缓存
	return m.loadCache()
}

// GetStats 获取统计信息
func (m *DBMetaManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"positions_count":   len(m.cache),
		"table_metas_count": len(m.tables),
		"last_updated":      time.Now(),
	}
}
