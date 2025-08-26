// ========================================
// テスト実行メトリクス・監視デモンストレーション
// パフォーマンス計測とリソース使用量監視
// ========================================

package handlers

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"money_management/internal/database"
	"money_management/internal/models"
	testconfig "money_management/internal/testing"
)

// TestMetricsCollection_BasicLogin メトリクス収集基本ログインテスト
func TestMetricsCollection_BasicLogin(t *testing.T) {
	collector := testconfig.GetMetricsCollector()
	session := collector.StartTest("MetricsBasicLogin", "auth", "login", "performance")

	// データベース作成時のメトリクス記録
	db, cleanup, err := database.SetupLightweightTestDB("MetricsLogin")
	session.AddAssertion(err == nil)
	if err != nil {
		session.End(testconfig.StatusFailed, err.Error())
		t.Fatalf("DB作成失敗: %v", err)
	}
	defer cleanup()

	// メタデータ設定
	session.SetMetadata("test_type", "integration")
	session.SetMetadata("database_mode", "lightweight")

	// テストユーザー作成（DB操作メトリクス記録）
	start := time.Now()
	user := models.User{
		Name:         "メトリクステストユーザー",
		AccountID:    "metrics_test_user",
		PasswordHash: "hash123",
	}
	result := db.Create(&user)
	session.AddDatabaseOp("create", 1, time.Since(start))
	session.AddAssertion(result.Error == nil)

	// ユーザー検索（クエリメトリクス記録）
	start = time.Now()
	var foundUser models.User
	result = db.Where("account_id = ?", "metrics_test_user").First(&foundUser)
	session.AddDatabaseOp("query", 1, time.Since(start))
	session.AddAssertion(result.Error == nil)
	session.AddAssertion(foundUser.Name == "メトリクステストユーザー")

	// テスト完了
	session.End(testconfig.StatusPassed, "")

	t.Logf("✅ メトリクス収集テスト完了")
}

// TestMetricsCollection_BulkOperations メトリクス収集一括操作テスト
func TestMetricsCollection_BulkOperations(t *testing.T) {
	collector := testconfig.GetMetricsCollector()
	session := collector.StartTest("MetricsBulkOps", "database", "bulk", "performance")

	db, cleanup, err := database.SetupLightweightTestDB("MetricsBulk")
	session.AddAssertion(err == nil)
	if err != nil {
		session.End(testconfig.StatusFailed, err.Error())
		t.Fatalf("DB作成失敗: %v", err)
	}
	defer cleanup()

	session.SetMetadata("operation_type", "bulk_insert")
	session.SetMetadata("record_count", "100")

	// 100ユーザーの一括作成（パフォーマンス測定）
	start := time.Now()
	users := make([]models.User, 100)
	for i := 0; i < 100; i++ {
		users[i] = models.User{
			Name:         fmt.Sprintf("バルクユーザー%d", i),
			AccountID:    fmt.Sprintf("bulk_user_%d", i),
			PasswordHash: "hash123",
		}
	}

	result := db.CreateInBatches(users, 10)
	duration := time.Since(start)
	session.AddDatabaseOp("create", 100, duration)
	session.AddAssertion(result.Error == nil)

	// クエリパフォーマンス測定
	start = time.Now()
	var count int64
	db.Model(&models.User{}).Count(&count)
	session.AddDatabaseOp("query", 1, time.Since(start))
	session.AddAssertion(count == 100)

	session.End(testconfig.StatusPassed, "")

	t.Logf("📊 一括操作メトリクス: %d件を%vで作成", 100, duration)
}

// TestMetricsCollection_ErrorHandling メトリクス収集エラーハンドリングテスト
func TestMetricsCollection_ErrorHandling(t *testing.T) {
	collector := testconfig.GetMetricsCollector()
	session := collector.StartTest("MetricsErrorHandling", "error", "handling")

	db, cleanup, err := database.SetupLightweightTestDB("MetricsError")
	session.AddAssertion(err == nil)
	if err != nil {
		session.End(testconfig.StatusFailed, err.Error())
		t.Fatalf("DB作成失敗: %v", err)
	}
	defer cleanup()

	// 意図的にエラーを発生させる（重複キー制約違反）
	start := time.Now()
	user1 := models.User{
		Name:      "ユーザー1",
		AccountID: "duplicate_test",
	}
	result1 := db.Create(&user1)
	session.AddDatabaseOp("create", 1, time.Since(start))
	session.AddAssertion(result1.Error == nil)

	// 重複作成でエラー発生
	start = time.Now()
	user2 := models.User{
		Name:      "ユーザー2",
		AccountID: "duplicate_test", // 同じID
	}
	result2 := db.Create(&user2)
	session.AddDatabaseOp("create", 0, time.Since(start))
	session.AddAssertion(result2.Error != nil) // エラーが期待される

	session.SetMetadata("error_type", "duplicate_key")
	session.End(testconfig.StatusPassed, "")

	t.Logf("⚠️ エラーハンドリングメトリクス収集完了")
}

// TestMetricsCollection_ParallelExecution メトリクス収集並列実行テスト
func TestMetricsCollection_ParallelExecution(t *testing.T) {
	if !database.IsInMemoryDBEnabled() {
		t.Skip("インメモリDBが無効のためスキップ")
	}

	// 並列でメトリクス収集テストを実行
	t.Run("Parallel_Test_1", func(t *testing.T) {
		t.Parallel()
		testMetricsParallelWorker(t, 1)
	})

	t.Run("Parallel_Test_2", func(t *testing.T) {
		t.Parallel()
		testMetricsParallelWorker(t, 2)
	})

	t.Run("Parallel_Test_3", func(t *testing.T) {
		t.Parallel()
		testMetricsParallelWorker(t, 3)
	})
}

// testMetricsParallelWorker 並列テストワーカー
func testMetricsParallelWorker(t *testing.T, workerID int) {
	collector := testconfig.GetMetricsCollector()
	session := collector.StartTest(fmt.Sprintf("MetricsParallelWorker_%d", workerID), "parallel", "worker")

	db, cleanup, err := database.SetupLightweightTestDB(fmt.Sprintf("MetricsParallel_%d", workerID))
	session.AddAssertion(err == nil)
	if err != nil {
		session.End(testconfig.StatusFailed, err.Error())
		return
	}
	defer cleanup()

	session.SetMetadata("worker_id", fmt.Sprintf("%d", workerID))
	session.SetMetadata("execution_mode", "parallel")

	// ワーカー固有の操作実行
	start := time.Now()
	for i := 0; i < 5; i++ {
		user := models.User{
			Name:      fmt.Sprintf("並列ワーカー%d_ユーザー%d", workerID, i),
			AccountID: fmt.Sprintf("parallel_worker_%d_user_%d", workerID, i),
		}
		result := db.Create(&user)
		session.AddAssertion(result.Error == nil)
	}
	session.AddDatabaseOp("create", 5, time.Since(start))

	session.End(testconfig.StatusPassed, "")
	t.Logf("🔄 並列ワーカー %d 完了", workerID)
}

// TestMetricsExport メトリクスエクスポート機能テスト
func TestMetricsExport(t *testing.T) {
	// メトリクス出力ディレクトリを設定
	os.Setenv("TEST_METRICS_DIR", "./test-outputs")
	defer os.Unsetenv("TEST_METRICS_DIR")

	collector := testconfig.GetMetricsCollector()

	// サンプルテストを実行してメトリクスを生成
	session := collector.StartTest("MetricsExportTest", "export", "test")
	session.AddAssertion(true)
	session.SetMetadata("sample", "data")
	session.End(testconfig.StatusPassed, "")

	// 各形式でエクスポートテスト
	formats := []string{"json", "csv", "summary"}
	for _, format := range formats {
		err := collector.ExportMetrics(format)
		assert.NoError(t, err, "エクスポート失敗: %s", format)
		t.Logf("📄 %s形式でエクスポート完了", format)
	}

	// サマリー生成テスト
	summary := collector.GenerateSummary()
	assert.Contains(t, summary, "テスト実行メトリクス", "サマリーが不正")
	assert.Contains(t, summary, "総テスト数:", "テスト統計が含まれていない")

	t.Logf("📊 メトリクスサマリー:\n%s", summary)
}

// TestMetricsPerformanceComparison メトリクス性能比較テスト
func TestMetricsPerformanceComparison(t *testing.T) {
	collector := testconfig.GetMetricsCollector()

	// インメモリDB vs 通常DB のパフォーマンス比較
	testModes := []struct {
		name        string
		useInMemory bool
		fastMode    bool
	}{
		{"StandardMode", false, false},
		{"InMemoryMode", true, false},
		{"FastMode", true, true},
	}

	results := make(map[string]time.Duration)

	for _, mode := range testModes {
		if mode.useInMemory && !database.IsInMemoryDBEnabled() {
			continue // インメモリDBが無効の場合はスキップ
		}

		session := collector.StartTest(fmt.Sprintf("PerfComparison_%s", mode.name), "performance", "comparison")

		start := time.Now()

		// テスト実行（モード別）
		if mode.useInMemory {
			db, cleanup, err := database.SetupLightweightTestDB(fmt.Sprintf("PerfTest_%s", mode.name))
			session.AddAssertion(err == nil)
			if err == nil {
				// 軽量操作実行
				user := models.User{Name: "比較テストユーザー", AccountID: fmt.Sprintf("perf_%s", mode.name)}
				result := db.Create(&user)
				session.AddAssertion(result.Error == nil)
				cleanup()
			}
		} else {
			db, err := database.SetupTestDB()
			session.AddAssertion(err == nil)
			if err == nil {
				user := models.User{Name: "比較テストユーザー", AccountID: fmt.Sprintf("perf_%s", mode.name)}
				result := db.Create(&user)
				session.AddAssertion(result.Error == nil)
				database.CleanupTestDB(db)
			}
		}

		duration := time.Since(start)
		results[mode.name] = duration

		session.SetMetadata("mode", mode.name)
		session.SetMetadata("duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))
		session.End(testconfig.StatusPassed, "")
	}

	// 結果比較
	t.Logf("🚀 パフォーマンス比較結果:")
	for mode, duration := range results {
		t.Logf("   %s: %v", mode, duration)
	}

	if stdDuration, exists := results["StandardMode"]; exists {
		if memDuration, exists := results["InMemoryMode"]; exists {
			speedup := float64(stdDuration) / float64(memDuration)
			t.Logf("   インメモリDB速度向上: %.1fx", speedup)
		}
	}
}

// BenchmarkMetricsOverhead メトリクス収集オーバーヘッド測定
func BenchmarkMetricsOverhead(b *testing.B) {
	collector := testconfig.GetMetricsCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := collector.StartTest(fmt.Sprintf("BenchTest_%d", i), "benchmark")
		session.AddAssertion(true)
		session.SetMetadata("iteration", fmt.Sprintf("%d", i))
		session.End(testconfig.StatusPassed, "")
	}
}

func init() {
	// テスト開始時にメトリクス収集を初期化
	config := testconfig.GetGlobalConfig()
	if config.VerboseLogging {
		collector := testconfig.GetMetricsCollector()
		_ = collector // 初期化のみ実行
	}
}
