// ========================================
// 並列実行度調整システムテスト
// CPU・メモリリソースベース最適化の検証
// ========================================

package database

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestParallelOptimizer_BasicFunctionality 基本機能テスト
func TestParallelOptimizer_BasicFunctionality(t *testing.T) {
	optimizer := GetGlobalParallelOptimizer()
	assert.NotNil(t, optimizer, "並列最適化器が取得できない")

	// 初期メトリクス確認
	metrics, resources := optimizer.GetCurrentMetrics()
	assert.NotNil(t, metrics, "メトリクスが取得できない")
	assert.NotNil(t, resources, "リソース情報が取得できない")

	t.Logf("📊 初期並列実行メトリクス:")
	t.Logf("   現在の並列度: %d", optimizer.GetCurrentParallelism())
	t.Logf("   アクティブテスト数: %d", metrics.ActiveTests)
	t.Logf("   CPU使用率: %.1f%%", resources.CPUUsage*100)
	t.Logf("   メモリ使用率: %.1f%%", resources.MemoryUsage*100)
	t.Logf("   ゴルーチン数: %d", resources.GoroutineCount)
}

// TestParallelOptimizer_TestLifecycle テストライフサイクル管理テスト
func TestParallelOptimizer_TestLifecycle(t *testing.T) {
	optimizer := GetGlobalParallelOptimizer()

	// テスト開始登録
	testName := "TestLifecycle_Sample"
	optimizer.StartTest(testName, 5, 1.0)

	// アクティブテスト数確認
	metrics, _ := optimizer.GetCurrentMetrics()
	initialActiveTests := metrics.ActiveTests

	t.Logf("🔄 テスト開始後アクティブ数: %d", initialActiveTests)

	// 少し待機（実際のテスト実行をシミュレート）
	time.Sleep(100 * time.Millisecond)

	// テスト終了登録
	optimizer.FinishTest(testName, true)

	// 終了後のメトリクス確認
	finalMetrics, _ := optimizer.GetCurrentMetrics()
	t.Logf("✅ テスト完了後:")
	t.Logf("   完了テスト数: %d", finalMetrics.CompletedTests)
	t.Logf("   失敗テスト数: %d", finalMetrics.FailedTests)

	assert.Greater(t, finalMetrics.CompletedTests, int32(0), "完了テスト数が記録されていない")
}

// TestParallelOptimizer_ResourceBasedOptimization リソースベース最適化テスト
func TestParallelOptimizer_ResourceBasedOptimization(t *testing.T) {
	optimizer := GetGlobalParallelOptimizer()

	// 初期並列度記録
	initialParallelism := optimizer.GetCurrentParallelism()
	t.Logf("📈 初期並列度: %d", initialParallelism)

	// 最適化実行
	event := optimizer.OptimizeParallelism()
	assert.NotNil(t, event, "最適化イベントが生成されない")

	t.Logf("⚡ 最適化結果:")
	t.Logf("   並列度変更: %d → %d", event.OldParallelism, event.NewParallelism)
	t.Logf("   理由: %s", event.Reason)
	t.Logf("   CPU使用率: %.1f%%", event.CPUUsage*100)
	t.Logf("   メモリ使用率: %.1f%%", event.MemoryUsage*100)

	// 最適化履歴確認
	history := optimizer.GetOptimizationHistory()
	assert.Greater(t, len(history), 0, "最適化履歴が記録されていない")
}

// TestParallelOptimizer_LoadSimulation 負荷シミュレーションテスト
func TestParallelOptimizer_LoadSimulation(t *testing.T) {
	optimizer := GetGlobalParallelOptimizer()

	// 高負荷シミュレーション
	t.Run("高負荷シミュレーション", func(t *testing.T) {
		// 複数テストを同時開始
		var wg sync.WaitGroup
		testCount := 20

		for i := 0; i < testCount; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				testName := fmt.Sprintf("LoadTest_%d", id)
				optimizer.StartTest(testName, 3, 1.2) // 重いテスト

				// 短時間の処理をシミュレート
				time.Sleep(50 * time.Millisecond)

				optimizer.FinishTest(testName, true)
			}(i)
		}

		// 負荷中の最適化実行
		time.Sleep(10 * time.Millisecond) // 少し遅延
		event := optimizer.OptimizeParallelism()

		t.Logf("🔥 高負荷時最適化:")
		t.Logf("   並列度: %d → %d", event.OldParallelism, event.NewParallelism)
		t.Logf("   アクティブテスト: %d", event.ActiveTests)

		wg.Wait()

		// 負荷終了後の最適化
		finalEvent := optimizer.OptimizeParallelism()
		t.Logf("😌 負荷終了後最適化:")
		t.Logf("   並列度: %d → %d", finalEvent.OldParallelism, finalEvent.NewParallelism)
	})
}

// TestParallelOptimizer_AutoOptimization 自動最適化テスト
func TestParallelOptimizer_AutoOptimization(t *testing.T) {
	optimizer := GetGlobalParallelOptimizer()

	// 短いインターバル設定
	originalConfig := optimizer.config
	optimizer.config.MonitoringInterval = 200 * time.Millisecond
	optimizer.config.AutoOptimize = true
	defer func() { optimizer.config = originalConfig }()

	// 3秒間自動最適化実行
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	t.Logf("🤖 自動最適化開始（3秒間）")

	// バックグラウンドで自動最適化開始
	go optimizer.StartAutoOptimization(ctx)

	// 実行中に負荷をかける
	time.Sleep(500 * time.Millisecond)

	// 複数テスト実行をシミュレート
	for i := 0; i < 5; i++ {
		testName := fmt.Sprintf("AutoOptimizationTest_%d", i)
		optimizer.StartTest(testName, 1, 0.8)

		time.Sleep(100 * time.Millisecond)

		optimizer.FinishTest(testName, true)
	}

	// 自動最適化完了まで待機
	<-ctx.Done()

	// 最適化履歴確認
	history := optimizer.GetOptimizationHistory()
	t.Logf("📈 自動最適化履歴: %d件", len(history))

	if len(history) > 0 {
		lastEvent := history[len(history)-1]
		t.Logf("   最後の最適化: %s", lastEvent.Reason)
		t.Logf("   最終並列度: %d", lastEvent.NewParallelism)
	}
}

// TestParallelOptimizer_AdaptiveMode 適応モードテスト
func TestParallelOptimizer_AdaptiveMode(t *testing.T) {
	optimizer := GetGlobalParallelOptimizer()

	// 適応モード有効化
	optimizer.config.AdaptiveMode = true
	optimizer.config.AdjustmentStep = 2

	testScenarios := []struct {
		name        string
		testCount   int
		testWeight  float64
		priority    int
		expectation string
	}{
		{"軽量テスト", 3, 0.5, 1, "並列度維持または増加"},
		{"標準テスト", 8, 1.0, 5, "負荷に応じた調整"},
		{"重量テスト", 15, 2.0, 8, "並列度削減の可能性"},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			initialParallelism := optimizer.GetCurrentParallelism()

			// シナリオに応じたテスト実行
			var wg sync.WaitGroup
			for i := 0; i < scenario.testCount; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					testName := fmt.Sprintf("%s_%d", scenario.name, id)
					optimizer.StartTest(testName, scenario.priority, scenario.testWeight)

					// 重みに応じた実行時間
					sleepTime := time.Duration(scenario.testWeight*30) * time.Millisecond
					time.Sleep(sleepTime)

					optimizer.FinishTest(testName, true)
				}(i)
			}

			// 実行中に最適化
			time.Sleep(20 * time.Millisecond)
			event := optimizer.OptimizeParallelism()

			wg.Wait()

			t.Logf("🎯 %s結果:", scenario.name)
			t.Logf("   初期並列度: %d", initialParallelism)
			t.Logf("   最適化後: %d", event.NewParallelism)
			t.Logf("   期待: %s", scenario.expectation)
			t.Logf("   理由: %s", event.Reason)
		})
	}
}

// TestParallelOptimizer_EnvironmentAdaptation 環境適応テスト
func TestParallelOptimizer_EnvironmentAdaptation(t *testing.T) {
	// CPU数に基づく設定確認
	numCPU := runtime.NumCPU()

	t.Run("CPU数ベース設定", func(t *testing.T) {
		optimizer := GetGlobalParallelOptimizer()
		config := optimizer.config

		t.Logf("🖥️ 環境適応設定:")
		t.Logf("   CPU数: %d", numCPU)
		t.Logf("   最小並列度: %d", config.MinParallelism)
		t.Logf("   最大並列度: %d", config.MaxParallelism)
		t.Logf("   デフォルト: %d", config.DefaultParallelism)
		t.Logf("   調整ステップ: %d", config.AdjustmentStep)

		assert.Equal(t, numCPU, config.DefaultParallelism, "デフォルト並列度がCPU数と一致しない")
		assert.Equal(t, numCPU*4, config.MaxParallelism, "最大並列度がCPU数の4倍でない")
		assert.LessOrEqual(t, config.MinParallelism, config.DefaultParallelism, "最小値がデフォルトより大きい")
	})

	t.Run("メモリベース調整", func(t *testing.T) {
		optimizer := GetGlobalParallelOptimizer()

		// 初期メモリ使用量確認
		_, resources := optimizer.GetCurrentMetrics()
		initialMemory := resources.MemoryUsage

		// メモリ使用量を意図的に増加（大きなスライス作成）
		heavyData := make([][]byte, 1000)
		for i := range heavyData {
			heavyData[i] = make([]byte, 1024*1024) // 1MB
		}

		// メモリ負荷後の最適化
		event := optimizer.OptimizeParallelism()

		t.Logf("💾 メモリ負荷テスト:")
		t.Logf("   初期メモリ使用率: %.1f%%", initialMemory*100)
		t.Logf("   負荷後メモリ使用率: %.1f%%", event.MemoryUsage*100)
		t.Logf("   並列度変更: %d → %d", event.OldParallelism, event.NewParallelism)
		t.Logf("   理由: %s", event.Reason)

		// cleanup
		heavyData = nil
		runtime.GC()
	})
}

// TestParallelOptimizer_MetricsAccuracy メトリクス精度テスト
func TestParallelOptimizer_MetricsAccuracy(t *testing.T) {
	optimizer := GetGlobalParallelOptimizer()

	// テスト実行前の状態記録
	initialMetrics, _ := optimizer.GetCurrentMetrics()
	initialCompleted := initialMetrics.CompletedTests

	// 既知の数のテストを実行
	testCount := 10
	for i := 0; i < testCount; i++ {
		testName := fmt.Sprintf("MetricsTest_%d", i)
		optimizer.StartTest(testName, 1, 1.0)

		time.Sleep(10 * time.Millisecond) // 短時間実行

		optimizer.FinishTest(testName, i%7 != 0) // 約14%失敗
	}

	// 最終メトリクス確認
	finalMetrics, _ := optimizer.GetCurrentMetrics()

	expectedCompleted := initialCompleted + int32(testCount)
	actualCompleted := finalMetrics.CompletedTests

	t.Logf("📊 メトリクス精度検証:")
	t.Logf("   期待完了数: %d", expectedCompleted)
	t.Logf("   実際完了数: %d", actualCompleted)
	t.Logf("   失敗数: %d", finalMetrics.FailedTests)
	t.Logf("   平均実行時間: %v", finalMetrics.AverageTestDuration)
	t.Logf("   スループット: %.2f tests/sec", finalMetrics.ThroughputPerSecond)

	assert.Equal(t, expectedCompleted, actualCompleted, "完了テスト数が正確でない")
	assert.Greater(t, finalMetrics.FailedTests, int32(0), "失敗テスト数が記録されていない")
}

// BenchmarkParallelOptimizer 並列最適化器ベンチマーク
func BenchmarkParallelOptimizer(b *testing.B) {
	optimizer := GetGlobalParallelOptimizer()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			testName := fmt.Sprintf("BenchTest_%d", i)
			optimizer.StartTest(testName, 1, 1.0)
			optimizer.FinishTest(testName, true)
			i++
		}
	})
}

func init() {
	// テスト用の短いインターバル設定
	optimizer := GetGlobalParallelOptimizer()
	optimizer.config.MonitoringInterval = 100 * time.Millisecond
}
