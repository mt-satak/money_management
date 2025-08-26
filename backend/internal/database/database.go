package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB データベース接続のグローバル変数
// アプリケーション全体で共有されるデータベース接続
var DB *gorm.DB

// GlobalPoolOptimizer グローバル接続プール最適化器
var GlobalPoolOptimizer *PoolOptimizer

// Init データベース接続を初期化する
// MySQLデータベースに接続し、接続に失敗した場合は最大30回リトライする
func Init() error {
	var err error

	// 環境変数からデータベース接続情報を取得
	dbHost := getEnvOrDefault("DB_HOST", "database")
	dbPort := getEnvOrDefault("DB_PORT", "3306")
	dbUser := getEnvOrDefault("DB_USER", "root")
	dbName := getEnvOrDefault("DB_NAME", "household_budget")

	// データベースパスワードを安全に取得
	dbPassword := getSecurePassword()

	// データベース接続文字列を環境変数から構築
	// Docker環境での接続を想定した設定、日本時間（Asia/Tokyo）を明示的に指定
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FTokyo",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// データベース接続を最大30回試行（合計60秒間）
	// コンテナ起動時の順序問題に対応するためのリトライロジック
	for i := 1; i <= 30; i++ {
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Println("✅ データベースに接続しました")

			// 接続プール最適化器を初期化
			GlobalPoolOptimizer = NewPoolOptimizer(DB)

			// 接続プール設定を適用
			if err := applyOptimizedPoolSettings(DB); err != nil {
				log.Printf("⚠️ 接続プール最適化設定失敗: %v", err)
			} else {
				log.Println("🔧 接続プール最適化設定完了")
			}

			return nil
		}

		log.Printf("データベース接続を待機中... (%d/30)", i)
		time.Sleep(2 * time.Second)
	}

	return err
}

// GetDB データベース接続を取得する
// 他のパッケージからデータベース接続にアクセスするためのヘルパー関数
func GetDB() *gorm.DB {
	return DB
}

// applyOptimizedPoolSettings 最適化された接続プール設定を適用
func applyOptimizedPoolSettings(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// デフォルト設定から最適化設定を取得
	config := getDefaultPoolConfig()

	// 接続プール設定を適用
	sqlDB.SetMaxOpenConns(config.MaxConnections)     // 50接続に拡張
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)       // 20アイドル接続
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime) // 5分生存時間
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime) // 2分アイドル時間

	log.Printf("📊 接続プール設定:")
	log.Printf("   最大接続数: %d", config.MaxConnections)
	log.Printf("   最大アイドル: %d", config.MaxIdleConns)
	log.Printf("   接続生存時間: %v", config.ConnMaxLifetime)
	log.Printf("   アイドル時間: %v", config.ConnMaxIdleTime)

	return nil
}

// StartPoolOptimization 接続プール自動最適化開始
func StartPoolOptimization(ctx context.Context) {
	if GlobalPoolOptimizer == nil {
		log.Printf("⚠️ 接続プール最適化器が初期化されていません")
		return
	}

	// バックグラウンドで自動最適化開始
	go GlobalPoolOptimizer.StartAutoOptimization(ctx)
}

// GetPoolMetrics 現在の接続プールメトリクス取得
func GetPoolMetrics() (*PoolMetrics, *ResourceMonitor, error) {
	if GlobalPoolOptimizer == nil {
		return nil, nil, fmt.Errorf("接続プール最適化器が未初期化")
	}

	return GlobalPoolOptimizer.GetCurrentMetrics()
}

// getEnvOrDefault 環境変数を取得し、設定されていない場合はデフォルト値を返す
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getSecurePassword データベースパスワードをDocker Secretsまたは環境変数から安全に取得する
// 1. Docker Secrets (/run/secrets/db_password) を優先的に確認
// 2. DB_PASSWORD環境変数をフォールバック
// 3. 最後に開発用デフォルトパスワードを使用（警告付き）
func getSecurePassword() string {
	// Docker Secretsから読み込みを試行
	secretPath := "/run/secrets/db_password"
	if passwordBytes, err := os.ReadFile(secretPath); err == nil {
		password := strings.TrimSpace(string(passwordBytes))
		log.Println("🔐 Database password loaded from Docker Secrets")
		return password
	}

	// 環境変数からフォールバック
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		log.Println("🔐 Database password loaded from environment variable")
		return password
	}

	// 開発用デフォルト（セキュリティ警告付き）
	log.Println("⚠️ WARNING: Using default database password. Set DB_PASSWORD environment variable or use Docker Secrets for production")
	return "password"
}
