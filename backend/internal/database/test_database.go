// ========================================
// 自動テスト用データベース接続モジュール
// 本番環境と同じMySQL 8.0を使用してテスト実行
// ========================================

package database

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"money_management/internal/config"
	"money_management/internal/models"
)

// SetupTestDB テスト用データベース接続を初期化
// 本番環境と同じMySQLを使用するが、テスト用の設定とデータベース名を使用
func SetupTestDB() (*gorm.DB, error) {
	// テスト用データベース接続文字列（環境変数ベース）
	// CI/CD環境（GitHub Actions、GitLab CI等）やローカル環境で環境変数が設定される
	dbHost := config.GetStringEnv("DB_HOST", "localhost")
	dbPort := config.GetStringEnv("DB_PORT", "3306")
	dbUser := config.GetStringEnv("DB_USER", "root")
	dbPassword := config.GetStringEnv("DB_PASSWORD", "root_test_password")
	dbName := config.GetStringEnv("DB_NAME", "money_management_test")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FTokyo",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	var db *gorm.DB
	var err error

	// テスト用データベース接続を最大10回試行
	// テスト用コンテナの起動待ちのためのリトライロジック
	for i := 1; i <= 10; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			// パフォーマンス最適化設定
			DisableForeignKeyConstraintWhenMigrating: false,
			PrepareStmt:                              true, // プリペアドステートメント有効化
		})
		if err == nil {
			// 接続プールの最適化（テスト用固定設定）
			if sqlDB, err := db.DB(); err == nil {
				sqlDB.SetMaxOpenConns(20)                  // 最大接続数
				sqlDB.SetMaxIdleConns(10)                  // アイドル接続数
				sqlDB.SetConnMaxLifetime(30 * time.Minute) // 接続の最大生存時間
				log.Printf("✅ テスト用データベースに接続しました (最大接続: %d, アイドル: %d)", 20, 10)
			} else {
				log.Println("✅ テスト用データベースに接続しました")
			}
			break
		}

		log.Printf("テスト用データベース接続を待機中... (%d/10)", i)
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return nil, err
	}

	// テスト用テーブル作成（安全な初期化）
	err = safeAutoMigrate(db)
	if err != nil {
		return nil, fmt.Errorf("テーブル作成エラー: %v", err)
	}

	return db, nil
}

// safeAutoMigrate テーブルの安全な作成（安定性優先・非並列対応）
func safeAutoMigrate(db *gorm.DB) error {
	// Phase 1: 完全なデータベースリセット（安定性優先）
	if err := dropTablesIfExistsWithRetry(db); err != nil {
		log.Printf("⚠️ テーブル削除時エラー: %v", err)
		// 削除エラーは無視して続行
	}

	// Phase 2: 接続状態確認
	if err := verifyDatabaseConnection(db); err != nil {
		return fmt.Errorf("データベース接続確認失敗: %v", err)
	}

	// Phase 3: マイグレーション実行（リトライ機能付き）
	models := []interface{}{
		&models.User{},
		&models.MonthlyBill{},
		&models.BillItem{},
	}

	for _, model := range models {
		if err := createTableWithRetry(db, model); err != nil {
			return fmt.Errorf("テーブル作成失敗 %T: %v", model, err)
		}
	}

	log.Printf("✅ 統合テスト用データベース初期化完了")
	return nil
}

// dropTablesIfExistsWithRetry リトライ機能付きテーブル削除（外部キー制約考慮）
func dropTablesIfExistsWithRetry(db *gorm.DB) error {
	// 外部キー制約を一時的に無効化
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		log.Printf("⚠️ 外部キー制約無効化失敗: %v", err)
	}

	// 外部キー制約の逆順でテーブル削除
	tables := []string{"bill_items", "monthly_bills", "users"}

	for attempt := 1; attempt <= 3; attempt++ {
		allDeleted := true

		for _, table := range tables {
			if err := db.Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
				log.Printf("⚠️ テーブル削除失敗 %s (試行 %d/3): %v", table, attempt, err)
				allDeleted = false
			}
		}

		if allDeleted {
			// 外部キー制約を再有効化
			if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
				log.Printf("⚠️ 外部キー制約再有効化失敗: %v", err)
			}

			// テーブル削除確認のための待機
			time.Sleep(100 * time.Millisecond)

			log.Printf("🧹 統合テスト用テーブル削除完了")
			return nil
		}

		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	// 外部キー制約を再有効化（失敗時も）
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	return fmt.Errorf("テーブル削除が3回失敗しました")
}

// createTableWithRetry リトライ機能付きテーブル作成
func createTableWithRetry(db *gorm.DB, model interface{}) error {
	for attempt := 1; attempt <= 3; attempt++ {
		// テーブル存在確認
		if hasTable := db.Migrator().HasTable(model); hasTable {
			log.Printf("📋 テーブル既存確認済み %T", model)
			return nil
		}

		// マイグレーション実行
		err := db.AutoMigrate(model)
		if err == nil {
			// 作成確認のための短い待機
			time.Sleep(50 * time.Millisecond)
			log.Printf("✅ テーブル作成成功 %T", model)
			return nil
		}

		// テーブル存在エラーは成功扱い（競合時の安全策）
		if strings.Contains(err.Error(), "Error 1050") ||
			strings.Contains(err.Error(), "already exists") {
			log.Printf("⚠️ テーブル既存のためスキップ %T", model)
			return nil
		}

		log.Printf("⚠️ テーブル作成失敗 %T (試行 %d/3): %v", model, attempt, err)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return fmt.Errorf("テーブル作成が3回失敗しました")
}

// verifyDatabaseConnection データベース接続状態確認
func verifyDatabaseConnection(db *gorm.DB) error {
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Ping(); err != nil {
			return fmt.Errorf("データベースPing失敗: %v", err)
		}
		log.Printf("📡 データベース接続確認済み")
	}
	return nil
}

// isDeadlockError デッドロックエラーかどうかを判定
func isDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "deadlock") || strings.Contains(errStr, "1213")
}

// executeWithDeadlockRetry デッドロック発生時のリトライ機構付きでSQL実行（環境変数対応）
func executeWithDeadlockRetry(db *gorm.DB, sql string, maxRetries int) error {
	// nil ポインタチェック
	if db == nil {
		log.Printf("⚠️  executeWithDeadlockRetry: データベース接続がnilです。SQL実行をスキップ: %s", sql)
		return nil
	}

	if maxRetries == 0 {
		maxRetries = 3 // テスト用のデフォルト最大リトライ回数
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := db.Exec(sql).Error
		if err == nil {
			return nil
		}

		if !isDeadlockError(err) {
			// デッドロック以外のエラーの場合は即座に返す
			return err
		}

		if attempt < maxRetries {
			// デッドロックの場合は環境設定に基づくバックオフで待機してリトライ
			retryBackoffMs := config.GetIntEnv("TEST_RETRY_BACKOFF_MS", 100)
			waitTime := time.Duration(retryBackoffMs*attempt) * time.Millisecond
			errorLogging := config.GetBoolEnv("TEST_ERROR_LOGGING", true)
			if errorLogging {
				log.Printf("⚠️  デッドロック検出 - リトライ %d/%d (待機: %v): %s", attempt, maxRetries, waitTime, sql)
			}
			time.Sleep(waitTime)
		} else {
			errorLogging := config.GetBoolEnv("TEST_ERROR_LOGGING", true)
			if errorLogging {
				log.Printf("❌ デッドロック回避失敗 - 最大試行回数到達: %s", sql)
			}
			return err
		}
	}
	return nil
}

// CleanupTestDB テスト用データベースのクリーンアップ（外部キー制約対応・デッドロック回避機構付き）
// テスト実行後にテーブル内のデータを全削除して初期状態に戻す
func CleanupTestDB(db *gorm.DB) error {
	// nil ポインタチェック
	if db == nil {
		log.Println("⚠️  CleanupTestDB: データベース接続がnilです。クリーンアップをスキップします。")
		return nil
	}

	// データベース接続の健全性確認
	sqlDB, err := db.DB()
	if err != nil || sqlDB == nil {
		log.Printf("⚠️  CleanupTestDB: データベース接続が無効です: %v", err)
		return nil
	}

	// デッドロック対応: テスト用の最大リトライ回数
	maxRetries := 3

	// 並列実行時の外部キー制約問題を回避するため一時的に制約を無効化
	if err := executeWithDeadlockRetry(db, "SET FOREIGN_KEY_CHECKS = 0", maxRetries); err != nil {
		return err
	}

	// テーブル全体のクリーンアップ（TRUNCATE使用で高速化と重複回避）
	tables := []string{"bill_items", "monthly_bills", "users"}

	for _, table := range tables {
		// テーブル存在確認（正しい方法）
		var tableName string
		checkSQL := fmt.Sprintf("SHOW TABLES LIKE '%s'", table)
		result := db.Raw(checkSQL).Scan(&tableName)

		if result.Error != nil {
			log.Printf("⚠️ テーブル存在確認失敗 %s: %v", table, result.Error)
			continue
		}

		// テーブルが存在する場合のみクリーンアップ実行（行数で判定）
		tableExists := result.RowsAffected > 0
		if tableExists {
			// TRUNCATEでテーブル全体をクリア（AUTO_INCREMENTもリセット）
			sql := fmt.Sprintf("TRUNCATE TABLE %s", table)
			if err := executeWithDeadlockRetry(db, sql, maxRetries); err != nil {
				// TRUNCATE失敗時はDELETEにフォールバック
				log.Printf("⚠️ TRUNCATE失敗、DELETEでフォールバック: %s", table)
				fallbackSQL := fmt.Sprintf("DELETE FROM %s", table)
				if fallbackErr := executeWithDeadlockRetry(db, fallbackSQL, maxRetries); fallbackErr != nil {
					log.Printf("⚠️ テーブル%sのクリーンアップ失敗（無視して続行）: %v", table, fallbackErr)
				}
			}
		} else {
			log.Printf("📋 テーブル%sは存在しないためスキップ", table)
		}
	}

	// AUTO_INCREMENTをリセット（デッドロック回避機構付き）
	if err := executeWithDeadlockRetry(db, "ALTER TABLE bill_items AUTO_INCREMENT = 1", maxRetries); err != nil {
		return err
	}
	if err := executeWithDeadlockRetry(db, "ALTER TABLE monthly_bills AUTO_INCREMENT = 1", maxRetries); err != nil {
		return err
	}
	if err := executeWithDeadlockRetry(db, "ALTER TABLE users AUTO_INCREMENT = 1", maxRetries); err != nil {
		return err
	}

	// 外部キー制約を再有効化（重要: データ整合性の復旧）
	if err := executeWithDeadlockRetry(db, "SET FOREIGN_KEY_CHECKS = 1", maxRetries); err != nil {
		return err
	}

	return nil
}

// GenerateUniqueTestID テスト用のユニークID生成（重複回避）
func GenerateUniqueTestID(prefix string) string {
	timestamp := time.Now().UnixNano()
	randomPart := rand.Int63()
	return fmt.Sprintf("%s_%d_%d", prefix, timestamp, randomPart)
}

// SafeCreateTestUser 重複回避機能付きテストユーザー作成
func SafeCreateTestUser(db *gorm.DB, baseName string) (*models.User, error) {
	maxAttempts := 5

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// ユニークなaccount_idを生成
		accountID := GenerateUniqueTestID(baseName)

		user := &models.User{
			Name:         fmt.Sprintf("%s_%d", baseName, attempt),
			AccountID:    accountID,
			PasswordHash: "test_password_hash",
		}

		err := db.Create(user).Error
		if err == nil {
			return user, nil
		}

		// 重複エラー（Error 1062）の場合はリトライ
		if strings.Contains(err.Error(), "Error 1062") ||
			strings.Contains(err.Error(), "Duplicate entry") {
			log.Printf("⚠️  重複エラー - リトライ %d/%d: %s", attempt, maxAttempts, accountID)
			time.Sleep(time.Duration(attempt*10) * time.Millisecond)
			continue
		}

		// その他のエラーは即座に返す
		return nil, fmt.Errorf("ユーザー作成失敗: %v", err)
	}

	return nil, fmt.Errorf("最大試行回数(%d)に達しました - ユニークなユーザー作成に失敗", maxAttempts)
}
