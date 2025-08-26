// ========================================
// テスト実行メトリクス・監視システム
// パフォーマンス計測とリソース使用量監視
// ========================================

package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"gorm.io/gorm"
)

// TestMetrics テスト実行メトリクス
type TestMetrics struct {
	TestName     string            `json:"test_name"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	Status       TestStatus        `json:"status"`
	DatabaseType string            `json:"database_type"` // "mysql", "sqlite", "inmemory"
	ParallelMode bool              `json:"parallel_mode"`
	MemoryUsage  MemoryMetrics     `json:"memory_usage"`
	DatabaseOps  DatabaseMetrics   `json:"database_ops"`
	Assertions   AssertionMetrics  `json:"assertions"`
	Tags         []string          `json:"tags"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// TestStatus テスト実行状態
type TestStatus string

const (
	StatusPassed  TestStatus = "passed"
	StatusFailed  TestStatus = "failed"
	StatusSkipped TestStatus = "skipped"
	StatusRunning TestStatus = "running"
)

// MemoryMetrics メモリ使用量メトリクス
type MemoryMetrics struct {
	HeapAlloc     uint64  `json:"heap_alloc"`      // 現在のヒープ使用量
	HeapSys       uint64  `json:"heap_sys"`        // システムから取得したヒープメモリ
	StackInuse    uint64  `json:"stack_inuse"`     // スタック使用量
	NumGC         uint32  `json:"num_gc"`          // GC実行回数
	GCCPUFraction float64 `json:"gc_cpu_fraction"` // GCによるCPU使用率
}

// DatabaseMetrics データベース操作メトリクス
type DatabaseMetrics struct {
	Connections    int           `json:"connections"`     // アクティブ接続数
	Queries        int           `json:"queries"`         // 実行クエリ数
	Transactions   int           `json:"transactions"`    // トランザクション数
	CreatedRecords int           `json:"created_records"` // 作成レコード数
	UpdatedRecords int           `json:"updated_records"` // 更新レコード数
	DeletedRecords int           `json:"deleted_records"` // 削除レコード数
	QueryTime      time.Duration `json:"query_time"`      // 総クエリ実行時間
}

// AssertionMetrics アサーション実行メトリクス
type AssertionMetrics struct {
	Total  int `json:"total"`  // 総アサーション数
	Passed int `json:"passed"` // 成功アサーション数
	Failed int `json:"failed"` // 失敗アサーション数
}

// MetricsCollector テストメトリクス収集器
type MetricsCollector struct {
	mu        sync.RWMutex
	metrics   []TestMetrics
	outputDir string
	startTime time.Time
	memStats  runtime.MemStats
	dbTracker *DatabaseTracker
}

var (
	globalCollector *MetricsCollector
	collectorOnce   sync.Once
)

// GetMetricsCollector グローバルメトリクス収集器を取得
func GetMetricsCollector() *MetricsCollector {
	collectorOnce.Do(func() {
		outputDir := os.Getenv("TEST_METRICS_DIR")
		if outputDir == "" {
			outputDir = "./test-metrics"
		}

		globalCollector = &MetricsCollector{
			metrics:   make([]TestMetrics, 0),
			outputDir: outputDir,
			dbTracker: NewDatabaseTracker(),
		}

		// 出力ディレクトリ作成
		os.MkdirAll(outputDir, 0755)
	})
	return globalCollector
}

// StartTest テスト開始時のメトリクス記録
func (mc *MetricsCollector) StartTest(testName string, tags ...string) *TestSession {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// メモリ統計を取得
	runtime.ReadMemStats(&mc.memStats)

	session := &TestSession{
		collector: mc,
		metrics: TestMetrics{
			TestName:     testName,
			StartTime:    time.Now(),
			Status:       StatusRunning,
			Tags:         tags,
			DatabaseType: mc.detectDatabaseType(),
			ParallelMode: mc.isParallelMode(),
			MemoryUsage:  mc.collectMemoryMetrics(),
			Metadata:     make(map[string]string),
		},
	}

	config := GetGlobalConfig()
	if config.VerboseLogging {
		log.Printf("📊 [%s] テスト開始 - メトリクス収集開始", testName)
	}

	return session
}

// TestSession テスト実行セッション
type TestSession struct {
	collector *MetricsCollector
	metrics   TestMetrics
}

// AddAssertion アサーション結果を記録
func (ts *TestSession) AddAssertion(passed bool) {
	ts.metrics.Assertions.Total++
	if passed {
		ts.metrics.Assertions.Passed++
	} else {
		ts.metrics.Assertions.Failed++
	}
}

// AddDatabaseOp データベース操作を記録
func (ts *TestSession) AddDatabaseOp(opType string, recordCount int, duration time.Duration) {
	switch opType {
	case "create":
		ts.metrics.DatabaseOps.CreatedRecords += recordCount
	case "update":
		ts.metrics.DatabaseOps.UpdatedRecords += recordCount
	case "delete":
		ts.metrics.DatabaseOps.DeletedRecords += recordCount
	}
	ts.metrics.DatabaseOps.Queries++
	ts.metrics.DatabaseOps.QueryTime += duration
}

// SetMetadata メタデータを設定
func (ts *TestSession) SetMetadata(key, value string) {
	if ts.metrics.Metadata == nil {
		ts.metrics.Metadata = make(map[string]string)
	}
	ts.metrics.Metadata[key] = value
}

// End テスト終了時のメトリクス記録
func (ts *TestSession) End(status TestStatus, errorMsg string) {
	ts.metrics.EndTime = time.Now()
	ts.metrics.Duration = ts.metrics.EndTime.Sub(ts.metrics.StartTime)
	ts.metrics.Status = status
	ts.metrics.ErrorMessage = errorMsg

	// 最終メモリ使用量を記録
	ts.metrics.MemoryUsage = ts.collector.collectMemoryMetrics()

	ts.collector.mu.Lock()
	defer ts.collector.mu.Unlock()

	ts.collector.metrics = append(ts.collector.metrics, ts.metrics)

	config := GetGlobalConfig()
	if config.VerboseLogging {
		log.Printf("📊 [%s] テスト完了 - %s (実行時間: %v)",
			ts.metrics.TestName, status, ts.metrics.Duration)
	}
}

// collectMemoryMetrics メモリメトリクス収集
func (mc *MetricsCollector) collectMemoryMetrics() MemoryMetrics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return MemoryMetrics{
		HeapAlloc:     ms.HeapAlloc,
		HeapSys:       ms.HeapSys,
		StackInuse:    ms.StackInuse,
		NumGC:         ms.NumGC,
		GCCPUFraction: ms.GCCPUFraction,
	}
}

// detectDatabaseType 使用中のデータベースタイプを検出
func (mc *MetricsCollector) detectDatabaseType() string {
	config := GetGlobalConfig()
	if config.UseInMemoryDB {
		return "inmemory"
	} else if config.ParallelTestEnabled {
		return "mysql-parallel"
	}
	return "mysql"
}

// isParallelMode 並列実行モードかチェック
func (mc *MetricsCollector) isParallelMode() bool {
	config := GetGlobalConfig()
	return config.ParallelTestEnabled || config.UseInMemoryDB
}

// ExportMetrics メトリクスをファイルに出力
func (mc *MetricsCollector) ExportMetrics(format string) error {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	timestamp := time.Now().Format("20060102_150405")

	switch format {
	case "json":
		return mc.exportJSON(timestamp)
	case "csv":
		return mc.exportCSV(timestamp)
	case "summary":
		return mc.exportSummary(timestamp)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportJSON JSON形式でエクスポート
func (mc *MetricsCollector) exportJSON(timestamp string) error {
	filename := filepath.Join(mc.outputDir, fmt.Sprintf("test-metrics_%s.json", timestamp))

	data, err := json.MarshalIndent(mc.metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON marshal failed: %v", err)
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// exportCSV CSV形式でエクスポート
func (mc *MetricsCollector) exportCSV(timestamp string) error {
	filename := filepath.Join(mc.outputDir, fmt.Sprintf("test-metrics_%s.csv", timestamp))

	content := "test_name,duration_ms,status,database_type,parallel_mode,memory_heap_mb,queries,assertions_total,assertions_failed\n"

	for _, metric := range mc.metrics {
		content += fmt.Sprintf("%s,%d,%s,%s,%t,%.2f,%d,%d,%d\n",
			metric.TestName,
			metric.Duration.Milliseconds(),
			metric.Status,
			metric.DatabaseType,
			metric.ParallelMode,
			float64(metric.MemoryUsage.HeapAlloc)/1024/1024,
			metric.DatabaseOps.Queries,
			metric.Assertions.Total,
			metric.Assertions.Failed,
		)
	}

	return ioutil.WriteFile(filename, []byte(content), 0644)
}

// exportSummary サマリー形式でエクスポート
func (mc *MetricsCollector) exportSummary(timestamp string) error {
	filename := filepath.Join(mc.outputDir, fmt.Sprintf("test-summary_%s.txt", timestamp))

	summary := mc.GenerateSummary()
	return ioutil.WriteFile(filename, []byte(summary), 0644)
}

// GenerateSummary 実行サマリー生成
func (mc *MetricsCollector) GenerateSummary() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.metrics) == 0 {
		return "📊 テスト実行メトリクス: データなし\n"
	}

	var (
		totalTests      = len(mc.metrics)
		passedTests     = 0
		failedTests     = 0
		skippedTests    = 0
		totalDuration   time.Duration
		totalQueries    = 0
		totalAssertion  = 0
		failedAssertion = 0
		maxMemory       uint64
		dbTypes         = make(map[string]int)
	)

	for _, m := range mc.metrics {
		switch m.Status {
		case StatusPassed:
			passedTests++
		case StatusFailed:
			failedTests++
		case StatusSkipped:
			skippedTests++
		}

		totalDuration += m.Duration
		totalQueries += m.DatabaseOps.Queries
		totalAssertion += m.Assertions.Total
		failedAssertion += m.Assertions.Failed

		if m.MemoryUsage.HeapAlloc > maxMemory {
			maxMemory = m.MemoryUsage.HeapAlloc
		}

		dbTypes[m.DatabaseType]++
	}

	successRate := float64(passedTests) / float64(totalTests) * 100
	avgDuration := totalDuration / time.Duration(totalTests)

	summary := fmt.Sprintf(`📊 テスト実行メトリクス サマリー
======================================

🔢 実行統計:
   総テスト数:     %d
   成功:           %d (%.1f%%)
   失敗:           %d
   スキップ:       %d

⏱️ パフォーマンス:
   総実行時間:     %v
   平均実行時間:   %v
   総クエリ数:     %d
   最大メモリ使用: %.2f MB

✅ アサーション:
   総アサーション: %d
   失敗アサーション: %d
   アサーション成功率: %.1f%%

🗄️ データベース利用:
`,
		totalTests, passedTests, successRate, failedTests, skippedTests,
		totalDuration, avgDuration, totalQueries,
		float64(maxMemory)/1024/1024,
		totalAssertion, failedAssertion,
		float64(totalAssertion-failedAssertion)/float64(totalAssertion)*100,
	)

	for dbType, count := range dbTypes {
		summary += fmt.Sprintf("   %s: %d テスト\n", dbType, count)
	}

	summary += fmt.Sprintf("\n📅 生成日時: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	return summary
}

// DatabaseTracker データベース操作追跡
type DatabaseTracker struct {
	mu          sync.RWMutex
	connections map[*gorm.DB]*DBMetrics
}

// DBMetrics データベースメトリクス
type DBMetrics struct {
	QueryCount       int
	TransactionCount int
	LastActivity     time.Time
}

// NewDatabaseTracker データベース追跡器作成
func NewDatabaseTracker() *DatabaseTracker {
	return &DatabaseTracker{
		connections: make(map[*gorm.DB]*DBMetrics),
	}
}

// TrackDB データベース接続を追跡
func (dt *DatabaseTracker) TrackDB(db *gorm.DB) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.connections[db] = &DBMetrics{
		LastActivity: time.Now(),
	}
}

// RecordQuery クエリ実行を記録
func (dt *DatabaseTracker) RecordQuery(db *gorm.DB) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if metrics, exists := dt.connections[db]; exists {
		metrics.QueryCount++
		metrics.LastActivity = time.Now()
	}
}

// GetActiveConnections アクティブ接続数を取得
func (dt *DatabaseTracker) GetActiveConnections() int {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	return len(dt.connections)
}

// CleanupExpiredConnections 期限切れ接続をクリーンアップ
func (dt *DatabaseTracker) CleanupExpiredConnections(ctx context.Context, maxAge time.Duration) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dt.mu.Lock()
			now := time.Now()
			for db, metrics := range dt.connections {
				if now.Sub(metrics.LastActivity) > maxAge {
					delete(dt.connections, db)
				}
			}
			dt.mu.Unlock()
		}
	}
}
