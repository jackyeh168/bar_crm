package persistence

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ===========================
// 測試輔助函數
// ===========================

// setupTestDB 創建測試用的 SQLite in-memory 資料庫
// 使用場景：整合測試，測試 Repository 與真實資料庫的互動
//
// 設計原則：
// 1. 隔離性：每個測試使用獨立的 in-memory DB
// 2. 速度：SQLite in-memory 快速，適合測試
// 3. 真實性：使用真實 SQL 引擎，而非 Mock
//
// 返回：
// - *gorm.DB: GORM 資料庫連接
// - cleanup func(): 清理函數，測試結束時調用
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	// 1. 建立 SQLite in-memory 資料庫
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 測試時靜音
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 2. 自動遷移（創建測試表）
	err = db.AutoMigrate(&PointsAccountModel{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// 3. 返回 DB 和清理函數
	cleanup := func() {
		// SQLite in-memory 資料庫會在連接關閉時自動清理
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

// createTestAccount 創建測試用的 PointsAccountModel
// 輔助函數，減少測試中的重複代碼
func createTestAccount(id, memberID string, earned, used int) *PointsAccountModel {
	return &PointsAccountModel{
		ID:           id,
		MemberID:     memberID,
		EarnedPoints: earned,
		UsedPoints:   used,
	}
}
