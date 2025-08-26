// ========================================
// 自動テスト用データベース接続モジュール
// 本番環境と同じMySQL 8.0を使用してテスト実行
// ========================================

package database

import (
	"log"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"money_management/internal/models"
	"money_management/testconfig"
)

// SetupTestDB テスト用データベース接続を初期化
// 本番環境と同じMySQLを使用するが、テスト用の設定とデータベース名を使用
func SetupTestDB() (*gorm.DB, error) {
	// テスト用データベース接続文字列（接続プール最適化）
	// ポート3307のテスト用MySQLコンテナに接続
	dsn := "root:testpassword@tcp(localhost:3307)/household_budget_test?charset=utf8mb4&parseTime=True&loc=Asia%2FTokyo"

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
				sqlDB.SetMaxOpenConns(20)                   // 最大接続数
				sqlDB.SetMaxIdleConns(10)                   // アイドル接続数
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

	// テスト用テーブル作成（本番と同じモデルを使用）
	err = db.AutoMigrate(
		&models.User{},
		&models.MonthlyBill{},
		&models.BillItem{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
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
			testConfig := testconfig.GetGlobalConfig()
			waitTime := testConfig.GetRetryBackoff(attempt)
			if testConfig.ErrorLogging {
				log.Printf("⚠️  デッドロック検出 - リトライ %d/%d (待機: %v): %s", attempt, maxRetries, waitTime, sql)
			}
			time.Sleep(waitTime)
		} else {
			testConfig := testconfig.GetGlobalConfig()
			if testConfig.ErrorLogging {
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
	// デッドロック対応: テスト用の最大リトライ回数
	maxRetries := 3

	// 並列実行時の外部キー制約問題を回避するため一時的に制約を無効化
	if err := executeWithDeadlockRetry(db, "SET FOREIGN_KEY_CHECKS = 0", maxRetries); err != nil {
		return err
	}

	// 外部キー制約を無効化したので順序を気にせずに削除可能
	if err := executeWithDeadlockRetry(db, "DELETE FROM bill_items", maxRetries); err != nil {
		return err
	}
	if err := executeWithDeadlockRetry(db, "DELETE FROM monthly_bills", maxRetries); err != nil {
		return err
	}
	if err := executeWithDeadlockRetry(db, "DELETE FROM users", maxRetries); err != nil {
		return err
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
