// ========================================
// 接続プール最適化テスト
// パフォーマンス改善と動的最適化の検証
// ========================================

package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"money_management/testconfig"
)

// TestConnectionPoolOptimization_BasicFunctionality 基本的な接続プール最適化テスト
func TestConnectionPoolOptimization_BasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("データベース接続が必要なためスキップ（-shortフラグ使用時）")
	}

	db, err := SetupTestDB()
	assert.NoError(t, err, "テストDB作成失敗")
	defer CleanupTestDB(db)

	// 接続プール最適化器作成
	optimizer := NewPoolOptimizer(db)
	assert.NotNil(t, optimizer, "最適化器作成失敗")

	// 初期メトリクス取得
	metrics, resources, err := optimizer.GetCurrentMetrics()
	assert.NoError(t, err, "メトリクス取得失敗")
	assert.NotNil(t, metrics, "メトリクスが取得できない")
	assert.NotNil(t, resources, "リソース情報が取得できない")

	t.Logf("📊 初期接続プールメトリクス:")
	t.Logf("   オープン接続数: %d", metrics.OpenConnections)
	t.Logf("   使用中接続数: %d", metrics.InUseConnections)
	t.Logf("   アイドル接続数: %d", metrics.IdleConnections)
	t.Logf("   最大接続数: %d", metrics.MaxOpenConnections)
	t.Logf("   使用率: %.1f%%", metrics.ConnectionUtilization*100)

	t.Logf("🖥️ システムリソース:")
	t.Logf("   CPU使用率: %.1f%%", resources.CPUUsage*100)
	t.Logf("   メモリ使用率: %.1f%%", resources.MemoryUsage*100)
	t.Logf("   ゴルーチン数: %d", resources.Goroutines)
}

// TestConnectionPoolOptimization_ConfigurationUpdate 設定更新テスト
func TestConnectionPoolOptimization_ConfigurationUpdate(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err, "テストDB作成失敗")
	defer CleanupTestDB(db)

	optimizer := NewPoolOptimizer(db)

	// 初期設定確認
	initialConfig := optimizer.GetConfig()
	t.Logf("📋 初期設定: 最大接続数=%d", initialConfig.MaxConnections)

	// 新しい設定を適用
	newConfig := &PoolConfig{
		MinConnections:   5,
		MaxConnections:   100, // 大幅増加
		MaxIdleConns:     30,
		ConnMaxLifetime:  10 * time.Minute,
		ConnMaxIdleTime:  5 * time.Minute,
		AutoOptimize:     true,
		OptimizeInterval: 15 * time.Second,
		LoadThreshold:    0.8,
		MemoryThreshold:  0.9,
		Environment:      "testing",
	}

	err = optimizer.UpdateConfig(newConfig)
	assert.NoError(t, err, "設定更新失敗")

	// 設定が適用されたか確認
	updatedConfig := optimizer.GetConfig()
	assert.Equal(t, 100, updatedConfig.MaxConnections, "最大接続数が更新されていない")
	assert.Equal(t, 30, updatedConfig.MaxIdleConns, "最大アイドル接続数が更新されていない")

	t.Logf("✅ 設定更新完了: 最大接続数=%d", updatedConfig.MaxConnections)
}

// TestConnectionPoolOptimization_AutoOptimization 自動最適化テスト
func TestConnectionPoolOptimization_AutoOptimization(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err, "テストDB作成失敗")
	defer CleanupTestDB(db)

	optimizer := NewPoolOptimizer(db)

	// 自動最適化設定を短いインターバルに
	config := optimizer.GetConfig()
	config.OptimizeInterval = 1 * time.Second // 1秒間隔
	config.AutoOptimize = true
	optimizer.UpdateConfig(config)

	// 自動最適化を短時間実行
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Logf("🤖 自動最適化開始（5秒間）")

	// バックグラウンドで自動最適化実行
	go optimizer.StartAutoOptimization(ctx)

	// 2秒待機
	time.Sleep(2 * time.Second)

	// メトリクス取得して最適化が動作しているか確認
	metrics, resources, err := optimizer.GetCurrentMetrics()
	assert.NoError(t, err, "自動最適化後のメトリクス取得失敗")

	t.Logf("🔧 自動最適化後メトリクス:")
	t.Logf("   CPU使用率: %.1f%%", resources.CPUUsage*100)
	t.Logf("   メモリ使用率: %.1f%%", resources.MemoryUsage*100)
	t.Logf("   接続使用率: %.1f%%", metrics.ConnectionUtilization*100)

	// 自動最適化が少なくとも一度実行されたことを確認
	assert.True(t, !optimizer.lastOptimization.IsZero(), "自動最適化が実行されていない")
}

// TestConnectionPoolOptimization_LoadBasedOptimization 負荷ベース最適化テスト
func TestConnectionPoolOptimization_LoadBasedOptimization(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err, "テストDB作成失敗")
	defer CleanupTestDB(db)

	optimizer := NewPoolOptimizer(db)

	// 負荷をシミュレートするため複数の同時接続を作成
	t.Run("高負荷シミュレーション", func(t *testing.T) {
		ctx := context.Background()

		// 初期状態の接続数を記録
		initialMetrics, _, _ := optimizer.GetCurrentMetrics()
		initialMaxConns := initialMetrics.MaxOpenConnections

		// 同時に複数のクエリを実行して負荷をかける
		for i := 0; i < 10; i++ {
			go func(id int) {
				var result int
				db.Raw("SELECT SLEEP(0.1)").Scan(&result) // 100ms待機
			}(i)
		}

		// 少し待ってから最適化実行
		time.Sleep(200 * time.Millisecond)
		err := optimizer.OptimizeConnections(ctx)
		assert.NoError(t, err, "負荷時の最適化失敗")

		// 最適化後のメトリクス取得
		optimizedMetrics, _, _ := optimizer.GetCurrentMetrics()

		t.Logf("📊 負荷最適化結果:")
		t.Logf("   最適化前最大接続数: %d", initialMaxConns)
		t.Logf("   最適化後最大接続数: %d", optimizedMetrics.MaxOpenConnections)
		t.Logf("   現在の使用中接続数: %d", optimizedMetrics.InUseConnections)
	})
}

// TestConnectionPoolOptimization_EnvironmentSpecific 環境別設定テスト
func TestConnectionPoolOptimization_EnvironmentSpecific(t *testing.T) {
	testCases := []struct {
		name        string
		useInMemory bool
		expected    struct {
			maxConns int
			minConns int
		}
	}{
		{
			name:        "MySQL環境",
			useInMemory: false,
			expected: struct {
				maxConns int
				minConns int
			}{maxConns: 50, minConns: 10},
		},
		{
			name:        "SQLite InMemory環境",
			useInMemory: true,
			expected: struct {
				maxConns int
				minConns int
			}{maxConns: 5, minConns: 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 環境に応じたDB作成
			var db *gorm.DB
			var cleanup func()
			var err error

			if tc.useInMemory {
				db, cleanup, err = SetupLightweightTestDB("PoolOptimizationTest")
				if err != nil {
					t.Skip("InMemoryDB not available")
				}
			} else {
				db, err = SetupTestDB()
				cleanup = func() { CleanupTestDB(db) }
			}

			assert.NoError(t, err, "DB作成失敗")
			defer cleanup()

			// 一時的に環境変数を設定
			if tc.useInMemory {
				os.Setenv("USE_INMEMORY_DB", "true")
				defer os.Unsetenv("USE_INMEMORY_DB")
			}

			optimizer := NewPoolOptimizer(db)
			config := optimizer.GetConfig()

			t.Logf("🏗️ %s設定:", tc.name)
			t.Logf("   最大接続数: %d (期待値: %d)", config.MaxConnections, tc.expected.maxConns)
			t.Logf("   最小接続数: %d (期待値: %d)", config.MinConnections, tc.expected.minConns)

			assert.Equal(t, tc.expected.maxConns, config.MaxConnections, "最大接続数が期待値と異なる")
			assert.Equal(t, tc.expected.minConns, config.MinConnections, "最小接続数が期待値と異なる")
		})
	}
}

// TestConnectionPoolOptimization_PerformanceBenchmark パフォーマンスベンチマーク
func TestConnectionPoolOptimization_PerformanceBenchmark(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err, "テストDB作成失敗")
	defer CleanupTestDB(db)

	// 最適化前のパフォーマンス測定
	t.Run("最適化前", func(t *testing.T) {
		measureConnectionPerformance(t, db, "最適化前", 50)
	})

	// 接続プール最適化適用
	optimizer := NewPoolOptimizer(db)
	ctx := context.Background()
	err = optimizer.OptimizeConnections(ctx)
	assert.NoError(t, err, "最適化実行失敗")

	// 最適化後のパフォーマンス測定
	t.Run("最適化後", func(t *testing.T) {
		measureConnectionPerformance(t, db, "最適化後", 50)
	})
}

// measureConnectionPerformance 接続パフォーマンス測定
func measureConnectionPerformance(t *testing.T, db *gorm.DB, phase string, queryCount int) {
	start := time.Now()

	// 並列でクエリを実行
	done := make(chan bool, queryCount)

	for i := 0; i < queryCount; i++ {
		go func(id int) {
			var result int
			db.Raw("SELECT 1").Scan(&result)
			done <- true
		}(i)
	}

	// すべてのクエリ完了を待機
	for i := 0; i < queryCount; i++ {
		<-done
	}

	duration := time.Since(start)
	avgPerQuery := duration / time.Duration(queryCount)

	t.Logf("⚡ %s パフォーマンス (%dクエリ):", phase, queryCount)
	t.Logf("   総実行時間: %v", duration)
	t.Logf("   平均/クエリ: %v", avgPerQuery)
	t.Logf("   QPS: %.1f", float64(queryCount)/duration.Seconds())
}

// TestConnectionPoolOptimization_MetricsExport メトリクスエクスポートテスト
func TestConnectionPoolOptimization_MetricsExport(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err, "テストDB作成失敗")
	defer CleanupTestDB(db)

	optimizer := NewPoolOptimizer(db)

	// メトリクス統合テスト
	collector := testconfig.GetMetricsCollector()
	session := collector.StartTest("ConnectionPoolOptimizationTest", "pool", "optimization")

	// 最適化実行
	ctx := context.Background()
	start := time.Now()
	err = optimizer.OptimizeConnections(ctx)
	duration := time.Since(start)

	session.AddAssertion("optimization_success", err == nil)
	session.SetMetadata("optimization_duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))

	// 現在のプール状態をメタデータに記録
	metrics, resources, _ := optimizer.GetCurrentMetrics()
	session.SetMetadata("max_connections", fmt.Sprintf("%d", metrics.MaxOpenConnections))
	session.SetMetadata("connection_utilization", fmt.Sprintf("%.1f", metrics.ConnectionUtilization*100))
	session.SetMetadata("cpu_usage", fmt.Sprintf("%.1f", resources.CPUUsage*100))

	if err == nil {
		session.End(testconfig.StatusPassed, "")
	} else {
		session.End(testconfig.StatusFailed, err.Error())
	}

	t.Logf("📊 接続プール最適化メトリクス記録完了")
}

// BenchmarkConnectionPool 接続プールベンチマーク
func BenchmarkConnectionPool(b *testing.B) {
	db, err := SetupTestDB()
	if err != nil {
		b.Fatalf("テストDB作成失敗: %v", err)
	}
	defer CleanupTestDB(db)

	optimizer := NewPoolOptimizer(db)
	ctx := context.Background()
	optimizer.OptimizeConnections(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result int
			db.Raw("SELECT 1").Scan(&result)
		}
	})
}
