// ========================================
// テスト設定ユーティリティ
// 環境変数ベースの設定管理
// ========================================

package testconfig

import (
	"os"
	"strconv"
	"time"
)

// GetStringEnv 環境変数から文字列を取得（デフォルト値付き）
func GetStringEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// GetBoolEnv 環境変数からboolを取得（デフォルト値付き）
func GetBoolEnv(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	result, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return result
}

// GetIntEnv 環境変数からintを取得（デフォルト値付き）
func GetIntEnv(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return result
}

// TestConfig テスト設定構造体
type TestConfig struct {
	UseInMemoryDB       bool
	VerboseLogging      bool
	ErrorLogging        bool
	FastTestMode        bool
	RetryCount          int
	RetryDelay          time.Duration
	ParallelTestEnabled bool
	DatabaseMaxConns    int
	ConnMaxLifetime     time.Duration
}

// GetGlobalConfig グローバルテスト設定を取得
func GetGlobalConfig() *TestConfig {
	return &TestConfig{
		UseInMemoryDB:       GetBoolEnv("USE_INMEMORY_DB", false),
		VerboseLogging:      GetBoolEnv("VERBOSE_LOGGING", false),
		ErrorLogging:        GetBoolEnv("ERROR_LOGGING", true),
		FastTestMode:        GetBoolEnv("FAST_TEST_MODE", false),
		RetryCount:          GetIntEnv("TEST_RETRY_COUNT", 3),
		RetryDelay:          time.Duration(GetIntEnv("TEST_RETRY_DELAY_MS", 100)) * time.Millisecond,
		ParallelTestEnabled: GetBoolEnv("PARALLEL_TEST_ENABLED", true),
		DatabaseMaxConns:    GetIntEnv("DATABASE_MAX_CONNS", 50),
		ConnMaxLifetime:     time.Duration(GetIntEnv("CONN_MAX_LIFETIME_MINUTES", 5)) * time.Minute,
	}
}

// GetRetryBackoff リトライ時のバックオフ時間を取得
func (tc *TestConfig) GetRetryBackoff(attempt int) time.Duration {
	base := tc.RetryDelay
	if base == 0 {
		base = 100 * time.Millisecond
	}
	// 指数バックオフ
	return base * time.Duration(1<<uint(attempt-1))
}

// MetricsCollector メトリクス収集インターフェース（ダミー実装）
type MetricsCollector struct{}

// TestSession テストセッション（ダミー実装）
type TestSession struct {
	name     string
	category string
	tag      string
	metadata map[string]string
}

// GetMetricsCollector メトリクス収集器を取得
func GetMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// StartTest テストセッション開始
func (mc *MetricsCollector) StartTest(name, category, tag string) *TestSession {
	return &TestSession{
		name:     name,
		category: category,
		tag:      tag,
		metadata: make(map[string]string),
	}
}

// AddAssertion アサーション結果を追加
func (ts *TestSession) AddAssertion(name string, success bool) {
	// ダミー実装
}

// SetMetadata メタデータを設定
func (ts *TestSession) SetMetadata(key, value string) {
	ts.metadata[key] = value
}

// TestStatus テスト結果ステータス
type TestStatus string

const (
	StatusPassed TestStatus = "passed"
	StatusFailed TestStatus = "failed"
)

// End テストセッション終了
func (ts *TestSession) End(status TestStatus, message string) {
	// ダミー実装
}
