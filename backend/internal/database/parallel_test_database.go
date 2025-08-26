// ========================================
// 並列テスト専用データベース接続モジュール
// 各テストが独立したデータベースインスタンスを使用
// ========================================

package database

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"money_management/internal/models"
	"money_management/testconfig"
)

// ParallelTestDBManager 並列テスト用DB管理
type ParallelTestDBManager struct {
	mu        sync.Mutex
	databases map[string]*gorm.DB
	basePort  int
}

var (
	dbManager     *ParallelTestDBManager
	dbManagerOnce sync.Once
)

// GetParallelTestDBManager 並列テスト用DB管理インスタンスを取得
func GetParallelTestDBManager() *ParallelTestDBManager {
	dbManagerOnce.Do(func() {
		dbManager = &ParallelTestDBManager{
			databases: make(map[string]*gorm.DB),
			basePort:  3307,
		}
	})
	return dbManager
}

// SetupParallelTestDB 並列テスト用の独立データベース接続を作成
// 各テストが独立したDB接続プールとスキーマを使用
func SetupParallelTestDB(testName string) (*gorm.DB, error) {
	manager := GetParallelTestDBManager()
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// テスト固有のDB名を生成（一意性保証）
	timestamp := time.Now().UnixNano()
	randomNum := rand.Intn(10000)
	dbName := fmt.Sprintf("test_%s_%d_%d",
		sanitizeName(testName), timestamp, randomNum)

	// 既存の接続があれば再利用
	if db, exists := manager.databases[dbName]; exists {
		return db, nil
	}

	// 新しいDB接続を作成
	db, err := createParallelTestDB(dbName)
	if err != nil {
		return nil, fmt.Errorf("並列テスト用DB作成失敗 (%s): %v", dbName, err)
	}

	// 管理マップに登録
	manager.databases[dbName] = db

	log.Printf("✅ 並列テスト用DB作成完了: %s", dbName)
	return db, nil
}

// createParallelTestDB 並列テスト用データベースの実際の作成
func createParallelTestDB(dbName string) (*gorm.DB, error) {
	// テスト用データベース接続文字列（独立したDB名使用）
	dsn := fmt.Sprintf("root:testpassword@tcp(localhost:3307)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FTokyo", dbName)

	// まず、データベースを作成するための接続
	createDSN := "root:testpassword@tcp(localhost:3307)/?charset=utf8mb4&parseTime=True&loc=Asia%2FTokyo"

	// データベース作成用の一時接続
	tempDB, err := gorm.Open(mysql.Open(createDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // ログを抑制
	})
	if err != nil {
		return nil, fmt.Errorf("一時DB接続失敗: %v", err)
	}

	// データベースを作成
	result := tempDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	if result.Error != nil {
		return nil, fmt.Errorf("データベース作成失敗: %v", result.Error)
	}

	// 一時接続を閉じる
	sqlDB, _ := tempDB.DB()
	sqlDB.Close()

	var db *gorm.DB

	// 作成したデータベースに接続（リトライ機構付き）
	for i := 1; i <= 5; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			// 並列テスト用の最適化設定
			DisableForeignKeyConstraintWhenMigrating: false,
			PrepareStmt:                              true,
			Logger:                                   logger.Default.LogMode(logger.Silent), // テスト時のログを抑制
		})
		if err == nil {
			break
		}

		if i < 5 {
			time.Sleep(time.Duration(i*100) * time.Millisecond)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("並列テストDB接続失敗: %v", err)
	}

	// 接続プールの並列テスト用最適化
	config := testconfig.GetGlobalConfig()
	if sqlDB, err := db.DB(); err == nil {
		// 並列実行用に接続数を削減（競合回避）
		maxConns := config.DatabaseMaxConns / 4 // 並列テスト用に1/4に削減
		if maxConns < 2 {
			maxConns = 2
		}

		sqlDB.SetMaxOpenConns(maxConns)
		sqlDB.SetMaxIdleConns(maxConns / 2)
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	// テーブル作成
	err = db.AutoMigrate(
		&models.User{},
		&models.MonthlyBill{},
		&models.BillItem{},
	)
	if err != nil {
		return nil, fmt.Errorf("並列テスト用テーブル作成失敗: %v", err)
	}

	return db, nil
}

// CleanupParallelTestDB 並列テスト用データベースのクリーンアップ
func CleanupParallelTestDB(db *gorm.DB, testName string) error {
	if db == nil {
		return nil
	}

	// データベース名を取得
	var dbName string
	result := db.Raw("SELECT DATABASE()").Scan(&dbName)
	if result.Error != nil {
		log.Printf("⚠️  データベース名取得失敗: %v", result.Error)
		return result.Error
	}

	// 接続を閉じる
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	// データベースを削除（並列テスト後のクリーンアップ）
	if dbName != "" && dbName != "mysql" {
		tempDSN := "root:testpassword@tcp(localhost:3307)/?charset=utf8mb4&parseTime=True&loc=Asia%2FTokyo"
		tempDB, err := gorm.Open(mysql.Open(tempDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			tempDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
			tempSQLDB, _ := tempDB.DB()
			tempSQLDB.Close()
			log.Printf("✅ 並列テスト用DB削除完了: %s", dbName)
		}
	}

	// 管理マップから削除
	manager := GetParallelTestDBManager()
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for key, managedDB := range manager.databases {
		if managedDB == db {
			delete(manager.databases, key)
			break
		}
	}

	return nil
}

// sanitizeName テスト名をデータベース名用にサニタイズ
func sanitizeName(name string) string {
	// アルファベット、数字、アンダースコアのみ許可
	result := ""
	for _, char := range name {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' {
			result += string(char)
		} else {
			result += "_"
		}
	}

	// 長さ制限（MySQLのDB名制限）
	if len(result) > 30 {
		result = result[:30]
	}

	return result
}

// IsParallelTestEnabled 並列テスト実行が有効かチェック
func IsParallelTestEnabled() bool {
	// 環境変数で並列テストの有効/無効を制御
	// export ENABLE_PARALLEL_TESTS=true で有効化
	config := testconfig.GetGlobalConfig()
	return config.ParallelTestEnabled
}

// SetupOptimizedTestDB 最適化されたテスト用DB接続（並列/非並列自動判定）
func SetupOptimizedTestDB(testName string) (*gorm.DB, func(), error) {
	if IsParallelTestEnabled() {
		// 並列テスト用の独立DB
		db, err := SetupParallelTestDB(testName)
		if err != nil {
			return nil, nil, err
		}

		cleanup := func() {
			CleanupParallelTestDB(db, testName)
		}

		return db, cleanup, nil
	} else {
		// 従来の共有テストDB
		db, err := SetupTestDB()
		if err != nil {
			return nil, nil, err
		}

		cleanup := func() {
			CleanupTestDB(db)
		}

		return db, cleanup, nil
	}
}
