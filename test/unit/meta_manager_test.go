package main

import (
	"log"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"pikachun/internal/canal"
)

// setupTestDB 设置测试数据库
func setupTestDB() (*gorm.DB, error) {
	// 使用内存 SQLite 数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移表结构
	err = db.AutoMigrate(&canal.BinlogPosition{}, &canal.TableMetadata{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// TestDBMetaManager 测试 DBMetaManager 的基本功能
func TestDBMetaManager(t *testing.T) {
	// 设置测试数据库
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDBMetaManager] ", log.LstdFlags|log.Lshortfile)

	// 创建 DBMetaManager
	metaManager, err := canal.NewDBMetaManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create DBMetaManager: %v", err)
	}

	// 测试保存和加载位置
	instanceID := "test-instance-1"
	position := canal.Position{
		Name:    "mysql-bin.000001",
		Pos:     12345,
		GTIDSet: "test-gtid-set",
	}

	// 保存位置
	err = metaManager.SavePosition(instanceID, position)
	if err != nil {
		t.Errorf("Failed to save position: %v", err)
	}

	// 加载位置
	loadedPosition, err := metaManager.LoadPosition(instanceID)
	if err != nil {
		t.Errorf("Failed to load position: %v", err)
	}

	if loadedPosition.Name != position.Name {
		t.Errorf("Expected position name %s, got %s", position.Name, loadedPosition.Name)
	}

	if loadedPosition.Pos != position.Pos {
		t.Errorf("Expected position %d, got %d", position.Pos, loadedPosition.Pos)
	}

	if loadedPosition.GTIDSet != position.GTIDSet {
		t.Errorf("Expected GTID set %s, got %s", position.GTIDSet, loadedPosition.GTIDSet)
	}

	// 测试保存和加载表元数据
	tableMeta := &canal.TableMeta{
		Schema:  "test",
		Table:   "users",
		Columns: []string{"id", "name", "email"},
		Types:   []string{"int", "varchar", "varchar"},
	}

	// 保存表元数据
	err = metaManager.SaveTableMeta("test", "users", tableMeta)
	if err != nil {
		t.Errorf("Failed to save table metadata: %v", err)
	}

	// 加载表元数据
	loadedTableMeta, err := metaManager.LoadTableMeta("test", "users")
	if err != nil {
		t.Errorf("Failed to load table metadata: %v", err)
	}

	if loadedTableMeta == nil {
		t.Error("Expected loaded table metadata to be non-nil")
	} else {
		if loadedTableMeta.Schema != tableMeta.Schema {
			t.Errorf("Expected schema %s, got %s", tableMeta.Schema, loadedTableMeta.Schema)
		}

		if loadedTableMeta.Table != tableMeta.Table {
			t.Errorf("Expected table %s, got %s", tableMeta.Table, loadedTableMeta.Table)
		}

		if len(loadedTableMeta.Columns) != len(tableMeta.Columns) {
			t.Errorf("Expected %d columns, got %d", len(tableMeta.Columns), len(loadedTableMeta.Columns))
		}

		if len(loadedTableMeta.Types) != len(tableMeta.Types) {
			t.Errorf("Expected %d types, got %d", len(tableMeta.Types), len(loadedTableMeta.Types))
		}
	}

	// 测试加载不存在的表元数据
	nonExistentMeta, err := metaManager.LoadTableMeta("nonexistent", "table")
	if err != nil {
		t.Errorf("Failed to load non-existent table metadata: %v", err)
	}

	if nonExistentMeta != nil {
		t.Error("Expected non-existent table metadata to be nil")
	}
}

// TestDBMetaManagerConcurrentAccess 测试并发访问
func TestDBMetaManagerConcurrentAccess(t *testing.T) {
	// 设置测试数据库
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDBMetaManagerConcurrentAccess] ", log.LstdFlags|log.Lshortfile)

	// 创建 DBMetaManager
	metaManager, err := canal.NewDBMetaManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create DBMetaManager: %v", err)
	}

	// 并发测试保存位置
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			instanceID := "concurrent-test-" + string(rune(i+'0'))
			position := canal.Position{
				Name: "mysql-bin.000001",
				Pos:  uint32(10000 + i),
			}

			err := metaManager.SavePosition(instanceID, position)
			if err != nil {
				t.Errorf("Failed to save position in goroutine %d: %v", i, err)
			}

			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有位置都已保存
	for i := 0; i < 10; i++ {
		instanceID := "concurrent-test-" + string(rune(i+'0'))
		_, err := metaManager.LoadPosition(instanceID)
		if err != nil {
			t.Errorf("Failed to load position for %s: %v", instanceID, err)
		}
	}
}

// TestDBMetaManagerUpdatePosition 测试位置更新
func TestDBMetaManagerUpdatePosition(t *testing.T) {
	// 设置测试数据库
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDBMetaManagerUpdatePosition] ", log.LstdFlags|log.Lshortfile)

	// 创建 DBMetaManager
	metaManager, err := canal.NewDBMetaManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create DBMetaManager: %v", err)
	}

	instanceID := "update-test-instance"
	initialPosition := canal.Position{
		Name: "mysql-bin.000001",
		Pos:  10000,
	}

	// 保存初始位置
	err = metaManager.SavePosition(instanceID, initialPosition)
	if err != nil {
		t.Fatalf("Failed to save initial position: %v", err)
	}

	// 更新位置
	updatedPosition := canal.Position{
		Name: "mysql-bin.000001",
		Pos:  20000,
	}

	err = metaManager.SavePosition(instanceID, updatedPosition)
	if err != nil {
		t.Errorf("Failed to update position: %v", err)
	}

	// 加载更新后的位置
	loadedPosition, err := metaManager.LoadPosition(instanceID)
	if err != nil {
		t.Errorf("Failed to load updated position: %v", err)
	}

	if loadedPosition.Pos != updatedPosition.Pos {
		t.Errorf("Expected updated position %d, got %d", updatedPosition.Pos, loadedPosition.Pos)
	}
}

// TestDBMetaManagerUpdateTableMeta 测试表元数据更新
func TestDBMetaManagerUpdateTableMeta(t *testing.T) {
	// 设置测试数据库
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	// 创建测试日志器
	logger := log.New(os.Stdout, "[TestDBMetaManagerUpdateTableMeta] ", log.LstdFlags|log.Lshortfile)

	// 创建 DBMetaManager
	metaManager, err := canal.NewDBMetaManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create DBMetaManager: %v", err)
	}

	schema := "test"
	table := "products"

	// 初始表元数据
	initialMeta := &canal.TableMeta{
		Schema:  schema,
		Table:   table,
		Columns: []string{"id", "name"},
		Types:   []string{"int", "varchar"},
	}

	// 保存初始元数据
	err = metaManager.SaveTableMeta(schema, table, initialMeta)
	if err != nil {
		t.Fatalf("Failed to save initial table metadata: %v", err)
	}

	// 更新表元数据
	updatedMeta := &canal.TableMeta{
		Schema:  schema,
		Table:   table,
		Columns: []string{"id", "name", "price", "description"},
		Types:   []string{"int", "varchar", "decimal", "text"},
	}

	err = metaManager.SaveTableMeta(schema, table, updatedMeta)
	if err != nil {
		t.Errorf("Failed to update table metadata: %v", err)
	}

	// 加载更新后的元数据
	loadedMeta, err := metaManager.LoadTableMeta(schema, table)
	if err != nil {
		t.Errorf("Failed to load updated table metadata: %v", err)
	}

	if len(loadedMeta.Columns) != len(updatedMeta.Columns) {
		t.Errorf("Expected %d columns, got %d", len(updatedMeta.Columns), len(loadedMeta.Columns))
	}

	if len(loadedMeta.Types) != len(updatedMeta.Types) {
		t.Errorf("Expected %d types, got %d", len(updatedMeta.Types), len(loadedMeta.Types))
	}
}
