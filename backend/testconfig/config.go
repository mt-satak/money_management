package testconfig

import (
	"os"
	"strconv"
	"time"
)

// GlobalTestConfig テスト用グローバル設定
type GlobalTestConfig struct {
	DatabaseMaxConns     int
	DatabaseIdleConns    int
	ConnMaxLifetime      time.Duration
	DeadlockMaxRetries   int
	UseInMemoryDB        bool
	VerboseLogging       bool
	ParallelTestEnabled  bool
	ErrorLogging         bool
}

// GetGlobalConfig グローバルテスト設定を取得（環境変数対応）
func GetGlobalConfig() GlobalTestConfig {
	// 環境変数から設定を読み取る
	useInMemoryDB := os.Getenv("USE_INMEMORY_DB") == "true"
	verboseLogging := os.Getenv("VERBOSE_LOGGING") == "true"
	parallelTestEnabled := os.Getenv("PARALLEL_TEST") == "true"
	errorLogging := os.Getenv("ERROR_LOGGING") == "true"

	// 数値設定のデフォルト値
	maxConns := 20
	idleConns := 10
	maxRetries := 3

	if maxConnsStr := os.Getenv("TEST_DB_MAX_CONNS"); maxConnsStr != "" {
		if parsed, err := strconv.Atoi(maxConnsStr); err == nil {
			maxConns = parsed
		}
	}

	if idleConnsStr := os.Getenv("TEST_DB_IDLE_CONNS"); idleConnsStr != "" {
		if parsed, err := strconv.Atoi(idleConnsStr); err == nil {
			idleConns = parsed
		}
	}

	return GlobalTestConfig{
		DatabaseMaxConns:    maxConns,
		DatabaseIdleConns:   idleConns,
		ConnMaxLifetime:     30 * time.Minute,
		DeadlockMaxRetries:  maxRetries,
		UseInMemoryDB:       useInMemoryDB,
		VerboseLogging:      verboseLogging,
		ParallelTestEnabled: parallelTestEnabled,
		ErrorLogging:        errorLogging,
	}
}

// GetRetryBackoff リトライ時のバックオフ時間を取得
func (c GlobalTestConfig) GetRetryBackoff(attempt int) time.Duration {
	return time.Duration(attempt*100) * time.Millisecond
}

// MetricsCollector テストメトリクス収集器（ダミー実装）
type MetricsCollector struct{}

func GetMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// TestSession テストセッション（ダミー実装）
type TestSession struct{}

func (mc *MetricsCollector) StartSession(name string) *TestSession {
	return &TestSession{}
}

func (mc *MetricsCollector) StartTest(name string, args ...string) *TestSession {
	return &TestSession{}
}

func (ts *TestSession) End(status string, errorMsg string) {
	// ダミー実装
}

func (ts *TestSession) AddAssertion(key string, value interface{}) {
	// ダミー実装
}

func (ts *TestSession) SetMetadata(key string, value interface{}) {
	// ダミー実装
}

// ステータス定数
const (
	StatusPassed = "passed"
	StatusFailed = "failed"
)