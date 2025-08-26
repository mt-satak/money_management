// ========================================
// インメモリテストデータベース管理
// SQLite in-memoryによる超高速テスト実行
// ========================================

package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"money_management/internal/models"
	"money_management/testconfig"
)

// InMemoryDBManager インメモリDB管理
type InMemoryDBManager struct {
	mu        sync.Mutex
	databases map[string]*gorm.DB
	counter   int64
}

var (
	memDBManager     *InMemoryDBManager
	memDBManagerOnce sync.Once
)

// GetInMemoryDBManager インメモリDB管理インスタンスを取得
func GetInMemoryDBManager() *InMemoryDBManager {
	memDBManagerOnce.Do(func() {
		memDBManager = &InMemoryDBManager{
			databases: make(map[string]*gorm.DB),
			counter:   0,
		}
	})
	return memDBManager
}

// SetupInMemoryTestDB 超高速インメモリテストDB作成
// SQLiteのin-memoryモードを使用して最高速度のテスト実行を実現
func SetupInMemoryTestDB(testName string) (*gorm.DB, error) {
	manager := GetInMemoryDBManager()
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.counter++
	dbName := fmt.Sprintf("memory_test_%s_%d", sanitizeName(testName), manager.counter)

	// 既存の接続があれば再利用
	if db, exists := manager.databases[dbName]; exists {
		return db, nil
	}

	// SQLite in-memory データベース作成
	// :memory: を使用することで、ディスクI/Oを完全に回避
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		// インメモリDB用の最適化設定
		DisableForeignKeyConstraintWhenMigrating: false,
		PrepareStmt:                              false, // インメモリでは不要
		Logger:                                   logger.Default.LogMode(logger.Silent),

		// SQLiteに特化した設定
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("インメモリDB作成失敗 (%s): %v", dbName, err)
	}

	// SQLite固有の最適化設定
	sqlDB, err := db.DB()
	if err == nil {
		// インメモリDBでは接続プールは最小限に
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(0) // 無期限

		// SQLiteのパフォーマンス最適化
		db.Exec("PRAGMA journal_mode = MEMORY")
		db.Exec("PRAGMA synchronous = OFF")
		db.Exec("PRAGMA cache_size = 1000000")
		db.Exec("PRAGMA locking_mode = EXCLUSIVE")
		db.Exec("PRAGMA temp_store = MEMORY")
	}

	// SQLite用のテーブル作成（MySQL ENUM互換性の対応）
	err = db.AutoMigrate(
		&models.User{},
	)
	if err != nil {
		return nil, fmt.Errorf("インメモリユーザーテーブル作成失敗: %v", err)
	}

	// MonthlyBillテーブルを手動作成（SQLite ENUM対応）
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS monthly_bills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			year INTEGER NOT NULL,
			month INTEGER NOT NULL,
			requester_id INTEGER NOT NULL,
			payer_id INTEGER NOT NULL,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'requested', 'paid')),
			request_date DATETIME,
			payment_date DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (requester_id) REFERENCES users(id),
			FOREIGN KEY (payer_id) REFERENCES users(id)
		)
	`).Error
	if err != nil {
		return nil, fmt.Errorf("インメモリ家計簿テーブル作成失敗: %v", err)
	}

	// BillItemテーブルを手動作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS bill_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bill_id INTEGER NOT NULL,
			item_name TEXT NOT NULL,
			amount REAL NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (bill_id) REFERENCES monthly_bills(id)
		)
	`).Error
	if err != nil {
		return nil, fmt.Errorf("インメモリ家計簿項目テーブル作成失敗: %v", err)
	}

	// 管理マップに登録
	manager.databases[dbName] = db

	config := testconfig.GetGlobalConfig()
	if config.VerboseLogging {
		log.Printf("⚡ インメモリテストDB作成完了: %s (SQLite in-memory)", dbName)
	}

	return db, nil
}

// CleanupInMemoryTestDB インメモリテストDBのクリーンアップ
func CleanupInMemoryTestDB(db *gorm.DB, testName string) error {
	if db == nil {
		return nil
	}

	// 接続を閉じる（インメモリDBは自動削除される）
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	// 管理マップから削除
	manager := GetInMemoryDBManager()
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for key, managedDB := range manager.databases {
		if managedDB == db {
			delete(manager.databases, key)

			config := testconfig.GetGlobalConfig()
			if config.VerboseLogging {
				log.Printf("⚡ インメモリテストDB削除完了: %s", key)
			}
			break
		}
	}

	return nil
}

// SetupLightweightTestDB 軽量テスト用DB接続（自動選択）
// 環境変数 USE_INMEMORY_DB=true でインメモリDB使用
func SetupLightweightTestDB(testName string) (*gorm.DB, func(), error) {
	config := testconfig.GetGlobalConfig()

	// インメモリDBを使用するかチェック
	if config.UseInMemoryDB {
		// 超高速インメモリDB使用
		db, err := SetupInMemoryTestDB(testName)
		if err != nil {
			return nil, nil, err
		}

		cleanup := func() {
			CleanupInMemoryTestDB(db, testName)
		}

		return db, cleanup, nil
	}

	// 並列テスト有効の場合は独立MySQL使用
	if config.ParallelTestEnabled {
		return SetupOptimizedTestDB(testName)
	}

	// 通常の共有MySQL使用
	db, err := SetupTestDB()
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		CleanupTestDB(db)
	}

	return db, cleanup, nil
}

// BenchmarkInMemoryVsMySQL インメモリDB vs MySQL のパフォーマンス比較
func BenchmarkInMemoryVsMySQL(testName string) (memoryTime, mysqlTime time.Duration, speedup float64) {
	// インメモリDB測定
	start := time.Now()
	memDB, err := SetupInMemoryTestDB(testName + "_memory")
	if err == nil {
		// サンプル操作実行
		user := models.User{Name: "ベンチマークユーザー", AccountID: "benchmark_user"}
		memDB.Create(&user)

		var foundUser models.User
		memDB.Where("account_id = ?", "benchmark_user").First(&foundUser)

		CleanupInMemoryTestDB(memDB, testName+"_memory")
	}
	memoryTime = time.Since(start)

	// MySQL測定
	start = time.Now()
	mysqlDB, err := SetupTestDB()
	if err == nil {
		// サンプル操作実行
		user := models.User{Name: "ベンチマークユーザー", AccountID: "benchmark_user_mysql"}
		mysqlDB.Create(&user)

		var foundUser models.User
		mysqlDB.Where("account_id = ?", "benchmark_user_mysql").First(&foundUser)

		CleanupTestDB(mysqlDB)
	}
	mysqlTime = time.Since(start)

	// 速度向上率計算
	if memoryTime > 0 {
		speedup = float64(mysqlTime) / float64(memoryTime)
	}

	return memoryTime, mysqlTime, speedup
}

// IsInMemoryDBEnabled インメモリDBが有効かチェック
func IsInMemoryDBEnabled() bool {
	config := testconfig.GetGlobalConfig()
	return config.UseInMemoryDB
}
