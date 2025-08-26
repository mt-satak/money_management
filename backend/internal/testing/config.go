// ========================================
// テスト設定管理モジュール
// 環境変数による動的テスト設定
// ========================================

package testing

import (
	"log"
	"strconv"
	"time"

	"money_management/internal/config"
)

// TestConfig テスト実行設定
type TestConfig struct {
	// 並列実行設定
	MaxParallel         int           // 最大並列実行数
	ParallelEnabled     bool          // 並列実行の有効/無効
	ParallelTestEnabled bool          // 独立DB並列テストの有効/無効
	DatabaseMaxConns    int           // データベース最大接続数
	DatabaseIdleConns   int           // データベースアイドル接続数
	ConnMaxLifetime     time.Duration // データベース接続最大生存時間

	// 軽量テストデータ生成設定
	UseInMemoryDB bool // SQLite in-memoryDB使用の有効/無効
	FastTestMode  bool // 高速テストモード（軽量データ生成）

	// リトライ設定
	DeadlockMaxRetries int // デッドロック最大リトライ回数
	RetryBackoffMs     int // リトライ間隔（ミリ秒）

	// ログ設定
	VerboseLogging bool // 詳細ログの有効/無効
	ErrorLogging   bool // エラーログの有効/無効

	// テスト環境設定
	TestTimeout time.Duration // テストタイムアウト
	SkipSlow    bool          // 重いテストのスキップ
}

// DefaultConfig デフォルト設定
func DefaultConfig() *TestConfig {
	return &TestConfig{
		MaxParallel:         4,
		ParallelEnabled:     true,
		ParallelTestEnabled: false, // デフォルトは無効（安定性重視）
		DatabaseMaxConns:    20,
		DatabaseIdleConns:   10,
		ConnMaxLifetime:     300 * time.Second,
		UseInMemoryDB:       false, // デフォルトは無効（MySQL使用）
		FastTestMode:        false, // デフォルトは通常モード
		DeadlockMaxRetries:  3,
		RetryBackoffMs:      100,
		VerboseLogging:      false,
		ErrorLogging:        true,
		TestTimeout:         30 * time.Second,
		SkipSlow:            false,
	}
}

// LoadFromEnv 環境変数から設定を読み込み
func LoadFromEnv() *TestConfig {
	testConfig := DefaultConfig()

	// 並列実行設定
	if val := config.GetStringEnv("TEST_MAX_PARALLEL", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			testConfig.MaxParallel = parsed
			log.Printf("📋 TEST_MAX_PARALLEL設定: %d", parsed)
		}
	}

	if val := config.GetStringEnv("TEST_PARALLEL_ENABLED", ""); val != "" {
		testConfig.ParallelEnabled = val == "true" || val == "1"
		log.Printf("📋 TEST_PARALLEL_ENABLED設定: %v", testConfig.ParallelEnabled)
	}

	if val := config.GetStringEnv("ENABLE_PARALLEL_TESTS", ""); val != "" {
		testConfig.ParallelTestEnabled = val == "true" || val == "1"
		log.Printf("📋 ENABLE_PARALLEL_TESTS設定: %v", testConfig.ParallelTestEnabled)
	}

	// 軽量テストデータ生成設定
	if val := config.GetStringEnv("USE_INMEMORY_DB", ""); val != "" {
		testConfig.UseInMemoryDB = val == "true" || val == "1"
		log.Printf("📋 USE_INMEMORY_DB設定: %v", testConfig.UseInMemoryDB)
	}

	if val := config.GetStringEnv("FAST_TEST_MODE", ""); val != "" {
		testConfig.FastTestMode = val == "true" || val == "1"
		log.Printf("📋 FAST_TEST_MODE設定: %v", testConfig.FastTestMode)
	}

	// データベース接続設定
	if val := config.GetStringEnv("TEST_DB_MAX_CONNS", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			testConfig.DatabaseMaxConns = parsed
			log.Printf("📋 TEST_DB_MAX_CONNS設定: %d", parsed)
		}
	}

	if val := config.GetStringEnv("TEST_DB_IDLE_CONNS", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			testConfig.DatabaseIdleConns = parsed
			log.Printf("📋 TEST_DB_IDLE_CONNS設定: %d", parsed)
		}
	}

	if val := config.GetStringEnv("TEST_DB_CONN_LIFETIME", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			testConfig.ConnMaxLifetime = time.Duration(parsed) * time.Second
			log.Printf("📋 TEST_DB_CONN_LIFETIME設定: %v", testConfig.ConnMaxLifetime)
		}
	}

	// リトライ設定
	if val := config.GetStringEnv("TEST_DEADLOCK_MAX_RETRIES", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			testConfig.DeadlockMaxRetries = parsed
			log.Printf("📋 TEST_DEADLOCK_MAX_RETRIES設定: %d", parsed)
		}
	}

	if val := config.GetStringEnv("TEST_RETRY_BACKOFF_MS", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			testConfig.RetryBackoffMs = parsed
			log.Printf("📋 TEST_RETRY_BACKOFF_MS設定: %d", parsed)
		}
	}

	// ログ設定
	if val := config.GetStringEnv("TEST_VERBOSE_LOGGING", ""); val != "" {
		testConfig.VerboseLogging = val == "true" || val == "1"
		log.Printf("📋 TEST_VERBOSE_LOGGING設定: %v", testConfig.VerboseLogging)
	}

	if val := config.GetStringEnv("TEST_ERROR_LOGGING", ""); val != "" {
		testConfig.ErrorLogging = val == "true" || val == "1"
		log.Printf("📋 TEST_ERROR_LOGGING設定: %v", testConfig.ErrorLogging)
	}

	// テスト環境設定
	if val := config.GetStringEnv("TEST_TIMEOUT", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			testConfig.TestTimeout = time.Duration(parsed) * time.Second
			log.Printf("📋 TEST_TIMEOUT設定: %v", testConfig.TestTimeout)
		}
	}

	if val := config.GetStringEnv("TEST_SKIP_SLOW", ""); val != "" {
		testConfig.SkipSlow = val == "true" || val == "1"
		log.Printf("📋 TEST_SKIP_SLOW設定: %v", testConfig.SkipSlow)
	}

	return testConfig
}

// GetGlobalConfig グローバル設定を取得
var globalConfig *TestConfig

func GetGlobalConfig() *TestConfig {
	if globalConfig == nil {
		globalConfig = LoadFromEnv()
	}
	return globalConfig
}

// ResetGlobalConfig グローバル設定をリセット（テスト用）
func ResetGlobalConfig() {
	globalConfig = nil
}

// ShouldSkipParallel 並列実行をスキップすべきかチェック
func (c *TestConfig) ShouldSkipParallel() bool {
	return !c.ParallelEnabled
}

// GetRetryBackoff リトライ間隔を取得
func (c *TestConfig) GetRetryBackoff(attempt int) time.Duration {
	// 指数バックオフ: baseTime * attempt
	baseTime := time.Duration(c.RetryBackoffMs) * time.Millisecond
	return baseTime * time.Duration(attempt)
}

// LogConfig 設定内容をログ出力
func (c *TestConfig) LogConfig() {
	log.Printf("🔧 テスト設定:")
	log.Printf("   並列実行: %v (最大: %d)", c.ParallelEnabled, c.MaxParallel)
	log.Printf("   並列テスト: %v, インメモリDB: %v, 高速モード: %v", c.ParallelTestEnabled, c.UseInMemoryDB, c.FastTestMode)
	log.Printf("   DB接続: 最大%d, アイドル%d, 生存時間%v", c.DatabaseMaxConns, c.DatabaseIdleConns, c.ConnMaxLifetime)
	log.Printf("   リトライ: 最大%d回, 間隔%dms", c.DeadlockMaxRetries, c.RetryBackoffMs)
	log.Printf("   ログ: 詳細%v, エラー%v", c.VerboseLogging, c.ErrorLogging)
	log.Printf("   その他: タイムアウト%v, 重いテストスキップ%v", c.TestTimeout, c.SkipSlow)
}
