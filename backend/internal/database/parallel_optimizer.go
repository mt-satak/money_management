// ========================================
// 並列実行度自動調整システム
// CPU・メモリリソース監視による動的並列度制御
// ========================================

package database

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ParallelOptimizer 並列実行度最適化器
type ParallelOptimizer struct {
	mu                  sync.RWMutex
	config              *ParallelConfig
	metrics             *ParallelMetrics
	resourceMonitor     *SystemResourceMonitor
	testTracker         *ActiveTestTracker
	optimizationHistory []*OptimizationEvent
}

// ParallelConfig 並列実行設定
type ParallelConfig struct {
	// 基本設定
	MinParallelism     int `json:"min_parallelism"`     // 最小並列数
	MaxParallelism     int `json:"max_parallelism"`     // 最大並列数
	CurrentParallelism int `json:"current_parallelism"` // 現在の並列数
	DefaultParallelism int `json:"default_parallelism"` // デフォルト並列数

	// リソース閾値
	CPUThresholdLow     float64 `json:"cpu_threshold_low"`     // CPU低負荷閾値 (並列増)
	CPUThresholdHigh    float64 `json:"cpu_threshold_high"`    // CPU高負荷閾値 (並列減)
	MemoryThresholdLow  float64 `json:"memory_threshold_low"`  // メモリ低使用閾値
	MemoryThresholdHigh float64 `json:"memory_threshold_high"` // メモリ高使用閾値

	// 調整設定
	AdjustmentStep     int           `json:"adjustment_step"`     // 調整ステップ数
	MonitoringInterval time.Duration `json:"monitoring_interval"` // 監視間隔
	AdaptiveMode       bool          `json:"adaptive_mode"`       // 適応モード有効
	StabilityWindow    int           `json:"stability_window"`    // 安定性判定ウィンドウ

	// 環境別設定
	Environment  string `json:"environment"`   // 実行環境
	AutoOptimize bool   `json:"auto_optimize"` // 自動最適化有効
}

// ParallelMetrics 並列実行メトリクス
type ParallelMetrics struct {
	ActiveTests         int32         `json:"active_tests"`          // アクティブテスト数
	CompletedTests      int32         `json:"completed_tests"`       // 完了テスト数
	FailedTests         int32         `json:"failed_tests"`          // 失敗テスト数
	AverageTestDuration time.Duration `json:"avg_test_duration"`     // 平均テスト実行時間
	TotalTestDuration   time.Duration `json:"total_test_duration"`   // 総テスト実行時間
	ThroughputPerSecond float64       `json:"throughput_per_second"` // 秒あたりテスト実行数

	// リソース使用量
	PeakMemoryUsage uint64  `json:"peak_memory_usage"` // ピークメモリ使用量
	AverageCPUUsage float64 `json:"average_cpu_usage"` // 平均CPU使用率
	GCPressure      int     `json:"gc_pressure"`       // GC圧迫度
}

// SystemResourceMonitor システムリソース監視
type SystemResourceMonitor struct {
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	AvailableMemory uint64    `json:"available_memory"`
	GoroutineCount  int       `json:"goroutine_count"`
	GCFrequency     uint32    `json:"gc_frequency"`
	LoadAverage     float64   `json:"load_average"`
	LastUpdate      time.Time `json:"last_update"`
}

// ActiveTestTracker アクティブテスト追跡
type ActiveTestTracker struct {
	mu            sync.RWMutex
	activeTests   map[string]*TestExecution
	testStartTime map[string]time.Time
}

// TestExecution テスト実行情報
type TestExecution struct {
	TestName         string        `json:"test_name"`
	StartTime        time.Time     `json:"start_time"`
	ExpectedDuration time.Duration `json:"expected_duration"`
	ResourceWeight   float64       `json:"resource_weight"` // リソース重み(1.0=標準)
	Priority         int           `json:"priority"`        // 優先度(1-10)
}

// OptimizationEvent 最適化イベント
type OptimizationEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	OldParallelism int       `json:"old_parallelism"`
	NewParallelism int       `json:"new_parallelism"`
	Reason         string    `json:"reason"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	ActiveTests    int       `json:"active_tests"`
	Effectiveness  float64   `json:"effectiveness"` // 最適化効果
}

var (
	globalParallelOptimizer *ParallelOptimizer
	parallelOptimizerOnce   sync.Once
)

// GetGlobalParallelOptimizer グローバル並列最適化器取得
func GetGlobalParallelOptimizer() *ParallelOptimizer {
	parallelOptimizerOnce.Do(func() {
		globalParallelOptimizer = &ParallelOptimizer{
			config:              getDefaultParallelConfig(),
			metrics:             &ParallelMetrics{},
			resourceMonitor:     &SystemResourceMonitor{},
			testTracker:         newActiveTestTracker(),
			optimizationHistory: make([]*OptimizationEvent, 0),
		}
	})
	return globalParallelOptimizer
}

// getDefaultParallelConfig デフォルト並列設定取得
func getDefaultParallelConfig() *ParallelConfig {
	numCPU := runtime.NumCPU()

	return &ParallelConfig{
		MinParallelism:      1,
		MaxParallelism:      numCPU * 4, // CPU数の4倍まで
		CurrentParallelism:  numCPU,     // CPU数と同じ
		DefaultParallelism:  numCPU,
		CPUThresholdLow:     0.3,                 // 30%以下で並列度増加
		CPUThresholdHigh:    0.8,                 // 80%以上で並列度削減
		MemoryThresholdLow:  0.4,                 // 40%以下
		MemoryThresholdHigh: 0.85,                // 85%以上
		AdjustmentStep:      maxInt(1, numCPU/4), // 調整ステップ
		MonitoringInterval:  5 * time.Second,
		AdaptiveMode:        true,
		StabilityWindow:     3, // 3回連続で安定判定
		Environment:         "testing",
		AutoOptimize:        true,
	}
}

// newActiveTestTracker アクティブテスト追跡器作成
func newActiveTestTracker() *ActiveTestTracker {
	return &ActiveTestTracker{
		activeTests:   make(map[string]*TestExecution),
		testStartTime: make(map[string]time.Time),
	}
}

// OptimizeParallelism 並列度最適化実行
func (po *ParallelOptimizer) OptimizeParallelism() *OptimizationEvent {
	po.mu.Lock()
	defer po.mu.Unlock()

	// 現在の状況分析
	po.updateResourceMonitor()
	po.updateMetrics()

	// 最適化判定
	event := po.calculateOptimalParallelism()

	// 最適化適用
	if event.NewParallelism != event.OldParallelism {
		po.applyParallelismChange(event)
		po.recordOptimizationEvent(event)

		// ログ出力（常に有効）
		po.logOptimizationEvent(event)
	}

	return event
}

// updateResourceMonitor システムリソース監視更新
func (po *ParallelOptimizer) updateResourceMonitor() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	po.resourceMonitor = &SystemResourceMonitor{
		CPUUsage:        po.estimateCPULoad(),
		MemoryUsage:     float64(memStats.Alloc) / float64(memStats.Sys),
		AvailableMemory: memStats.Sys - memStats.Alloc,
		GoroutineCount:  runtime.NumGoroutine(),
		GCFrequency:     memStats.NumGC,
		LoadAverage:     po.calculateLoadAverage(),
		LastUpdate:      time.Now(),
	}
}

// estimateCPULoad CPU負荷推定
func (po *ParallelOptimizer) estimateCPULoad() float64 {
	// アクティブテスト数とゴルーチン数からCPU負荷を推定
	activeTests := atomic.LoadInt32(&po.metrics.ActiveTests)
	goroutines := runtime.NumGoroutine()

	// 基準値との比率でCPU使用率を推定
	baseline := float64(runtime.NumCPU() * 10) // CPU数×10をベースライン
	currentLoad := float64(goroutines) / baseline

	// アクティブテストの影響を加味
	testLoad := float64(activeTests) / float64(po.config.CurrentParallelism)

	estimatedCPU := (currentLoad + testLoad) / 2
	if estimatedCPU > 1.0 {
		estimatedCPU = 1.0
	}

	return estimatedCPU
}

// calculateLoadAverage 負荷平均計算
func (po *ParallelOptimizer) calculateLoadAverage() float64 {
	activeTests := atomic.LoadInt32(&po.metrics.ActiveTests)
	maxParallel := float64(po.config.CurrentParallelism)

	if maxParallel == 0 {
		return 0
	}

	return float64(activeTests) / maxParallel
}

// updateMetrics メトリクス更新
func (po *ParallelOptimizer) updateMetrics() {
	po.testTracker.mu.RLock()
	activeCount := len(po.testTracker.activeTests)
	po.testTracker.mu.RUnlock()

	atomic.StoreInt32(&po.metrics.ActiveTests, int32(activeCount))

	// 平均実行時間計算
	po.calculateAverageTestDuration()

	// スループット計算
	po.calculateThroughput()
}

// calculateAverageTestDuration 平均テスト実行時間計算
func (po *ParallelOptimizer) calculateAverageTestDuration() {
	po.testTracker.mu.RLock()
	defer po.testTracker.mu.RUnlock()

	if len(po.testTracker.testStartTime) == 0 {
		return
	}

	var totalDuration time.Duration
	var testCount int

	now := time.Now()
	for testName, startTime := range po.testTracker.testStartTime {
		if _, active := po.testTracker.activeTests[testName]; active {
			duration := now.Sub(startTime)
			totalDuration += duration
			testCount++
		}
	}

	if testCount > 0 {
		po.metrics.AverageTestDuration = totalDuration / time.Duration(testCount)
	}
}

// calculateThroughput スループット計算
func (po *ParallelOptimizer) calculateThroughput() {
	completedTests := atomic.LoadInt32(&po.metrics.CompletedTests)
	totalDuration := po.metrics.TotalTestDuration

	if totalDuration > 0 {
		po.metrics.ThroughputPerSecond = float64(completedTests) / totalDuration.Seconds()
	}
}

// calculateOptimalParallelism 最適並列度計算
func (po *ParallelOptimizer) calculateOptimalParallelism() *OptimizationEvent {
	current := po.config.CurrentParallelism
	optimal := current
	reason := "現状維持"

	cpuUsage := po.resourceMonitor.CPUUsage
	memoryUsage := po.resourceMonitor.MemoryUsage
	activeTests := atomic.LoadInt32(&po.metrics.ActiveTests)

	// 最適化ロジック
	if cpuUsage < po.config.CPUThresholdLow && memoryUsage < po.config.MemoryThresholdLow {
		// リソース余裕あり → 並列度増加
		if current < po.config.MaxParallelism {
			optimal = minInt(current+po.config.AdjustmentStep, po.config.MaxParallelism)
			reason = "リソース余裕により並列度増加"
		}

	} else if cpuUsage > po.config.CPUThresholdHigh || memoryUsage > po.config.MemoryThresholdHigh {
		// リソース不足 → 並列度削減
		if current > po.config.MinParallelism {
			optimal = maxInt(current-po.config.AdjustmentStep, po.config.MinParallelism)
			reason = "リソース不足により並列度削減"
		}

	} else if int(activeTests) < current/2 {
		// アクティブテスト数が少ない → 並列度削減
		if current > po.config.MinParallelism {
			optimal = maxInt(current-1, po.config.MinParallelism)
			reason = "低アクティビティにより並列度削減"
		}

	} else if int(activeTests) >= current && cpuUsage < po.config.CPUThresholdHigh {
		// 全並列スロットが使用中かつCPUに余裕 → 並列度増加
		if current < po.config.MaxParallelism {
			optimal = minInt(current+1, po.config.MaxParallelism)
			reason = "高利用率により並列度増加"
		}
	}

	return &OptimizationEvent{
		Timestamp:      time.Now(),
		OldParallelism: current,
		NewParallelism: optimal,
		Reason:         reason,
		CPUUsage:       cpuUsage,
		MemoryUsage:    memoryUsage,
		ActiveTests:    int(activeTests),
	}
}

// applyParallelismChange 並列度変更適用
func (po *ParallelOptimizer) applyParallelismChange(event *OptimizationEvent) {
	po.config.CurrentParallelism = event.NewParallelism

	// Goのランタイム設定更新
	runtime.GOMAXPROCS(event.NewParallelism)
}

// recordOptimizationEvent 最適化イベント記録
func (po *ParallelOptimizer) recordOptimizationEvent(event *OptimizationEvent) {
	po.optimizationHistory = append(po.optimizationHistory, event)

	// 履歴サイズ制限（最新100件）
	if len(po.optimizationHistory) > 100 {
		po.optimizationHistory = po.optimizationHistory[1:]
	}
}

// logOptimizationEvent 最適化イベントログ出力
func (po *ParallelOptimizer) logOptimizationEvent(event *OptimizationEvent) {
	log.Printf("⚡ 並列度最適化実行:")
	log.Printf("   並列度: %d → %d", event.OldParallelism, event.NewParallelism)
	log.Printf("   理由: %s", event.Reason)
	log.Printf("   CPU使用率: %.1f%%", event.CPUUsage*100)
	log.Printf("   メモリ使用率: %.1f%%", event.MemoryUsage*100)
	log.Printf("   アクティブテスト: %d", event.ActiveTests)
}

// StartTest テスト開始登録
func (po *ParallelOptimizer) StartTest(testName string, priority int, resourceWeight float64) {
	po.testTracker.mu.Lock()
	defer po.testTracker.mu.Unlock()

	now := time.Now()
	po.testTracker.activeTests[testName] = &TestExecution{
		TestName:       testName,
		StartTime:      now,
		ResourceWeight: resourceWeight,
		Priority:       priority,
	}
	po.testTracker.testStartTime[testName] = now

	atomic.AddInt32(&po.metrics.ActiveTests, 1)
}

// FinishTest テスト終了登録
func (po *ParallelOptimizer) FinishTest(testName string, success bool) {
	po.testTracker.mu.Lock()
	defer po.testTracker.mu.Unlock()

	if startTime, exists := po.testTracker.testStartTime[testName]; exists {
		duration := time.Since(startTime)
		po.metrics.TotalTestDuration += duration

		delete(po.testTracker.activeTests, testName)
		delete(po.testTracker.testStartTime, testName)
	}

	atomic.AddInt32(&po.metrics.ActiveTests, -1)
	atomic.AddInt32(&po.metrics.CompletedTests, 1)

	if !success {
		atomic.AddInt32(&po.metrics.FailedTests, 1)
	}
}

// StartAutoOptimization 自動最適化開始
func (po *ParallelOptimizer) StartAutoOptimization(ctx context.Context) {
	if !po.config.AutoOptimize {
		return
	}

	ticker := time.NewTicker(po.config.MonitoringInterval)
	defer ticker.Stop()

	log.Printf("🤖 並列度自動最適化開始 (間隔: %v)", po.config.MonitoringInterval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 並列度自動最適化停止")
			return
		case <-ticker.C:
			event := po.OptimizeParallelism()
			_ = event // 結果を記録済み
		}
	}
}

// GetCurrentMetrics 現在のメトリクス取得
func (po *ParallelOptimizer) GetCurrentMetrics() (*ParallelMetrics, *SystemResourceMonitor) {
	po.mu.RLock()
	defer po.mu.RUnlock()

	// メトリクス更新
	po.updateResourceMonitor()
	po.updateMetrics()

	return po.metrics, po.resourceMonitor
}

// GetOptimizationHistory 最適化履歴取得
func (po *ParallelOptimizer) GetOptimizationHistory() []*OptimizationEvent {
	po.mu.RLock()
	defer po.mu.RUnlock()

	// コピーを返す
	history := make([]*OptimizationEvent, len(po.optimizationHistory))
	copy(history, po.optimizationHistory)
	return history
}

// GetCurrentParallelism 現在の並列度取得
func (po *ParallelOptimizer) GetCurrentParallelism() int {
	po.mu.RLock()
	defer po.mu.RUnlock()

	return po.config.CurrentParallelism
}

// ヘルパー関数
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
