// ========================================
// データベース接続プール最適化システム
// CPU・メモリリソース監視による動的接続数調整
// ========================================

package database

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"gorm.io/gorm"
	"money_management/internal/config"
)

// PoolOptimizer 接続プール最適化器
type PoolOptimizer struct {
	mu               sync.RWMutex
	db               *gorm.DB
	config           *PoolConfig
	metrics          *PoolMetrics
	resourceMonitor  *ResourceMonitor
	lastOptimization time.Time
}

// PoolConfig 接続プール設定
type PoolConfig struct {
	// 基本設定
	MinConnections  int           `json:"min_connections"`    // 最小接続数
	MaxConnections  int           `json:"max_connections"`    // 最大接続数
	MaxIdleConns    int           `json:"max_idle_conns"`     // 最大アイドル接続数
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`  // 接続最大生存時間
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"` // アイドル最大時間

	// 動的調整設定
	AutoOptimize     bool          `json:"auto_optimize"`     // 自動最適化有効/無効
	OptimizeInterval time.Duration `json:"optimize_interval"` // 最適化実行間隔
	LoadThreshold    float64       `json:"load_threshold"`    // 負荷閾値（CPU使用率）
	MemoryThreshold  float64       `json:"memory_threshold"`  // メモリ閾値（使用率）

	// 環境別設定
	Environment string `json:"environment"` // "development", "testing", "production"
}

// PoolMetrics 接続プール使用状況メトリクス
type PoolMetrics struct {
	OpenConnections       int           `json:"open_connections"`       // 現在のオープン接続数
	InUseConnections      int           `json:"in_use_connections"`     // 使用中接続数
	IdleConnections       int           `json:"idle_connections"`       // アイドル接続数
	WaitCount             int64         `json:"wait_count"`             // 接続待ち回数
	WaitDuration          time.Duration `json:"wait_duration"`          // 総待ち時間
	MaxOpenConnections    int           `json:"max_open_connections"`   // 最大オープン接続数設定値
	ConnectionUtilization float64       `json:"connection_utilization"` // 接続使用率
}

// ResourceMonitor リソース監視
type ResourceMonitor struct {
	CPUUsage    float64 `json:"cpu_usage"`    // CPU使用率
	MemoryUsage float64 `json:"memory_usage"` // メモリ使用率
	GCCount     uint32  `json:"gc_count"`     // GC実行回数
	Goroutines  int     `json:"goroutines"`   // ゴルーチン数
}

// NewPoolOptimizer 接続プール最適化器作成
func NewPoolOptimizer(db *gorm.DB) *PoolOptimizer {
	return &PoolOptimizer{
		db:              db,
		config:          getDefaultPoolConfig(),
		metrics:         &PoolMetrics{},
		resourceMonitor: &ResourceMonitor{},
	}
}

// getDefaultPoolConfig デフォルト接続プール設定
func getDefaultPoolConfig() *PoolConfig {
	// 環境変数から設定を直接取得（循環インポートを避けるため）
	useInMemoryDB := config.GetBoolEnv("USE_INMEMORY_DB", false)

	// 環境に応じた基本設定
	var baseConfig PoolConfig

	if useInMemoryDB {
		// SQLite in-memory用設定
		baseConfig = PoolConfig{
			MinConnections:  1,
			MaxConnections:  5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 0, // 無期限
			ConnMaxIdleTime: 1 * time.Minute,
			Environment:     "development",
		}
	} else {
		// MySQL用設定
		baseConfig = PoolConfig{
			MinConnections:  10,
			MaxConnections:  50, // 20→50に拡張
			MaxIdleConns:    20,
			ConnMaxLifetime: 5 * time.Minute, // 300秒→5分に延長
			ConnMaxIdleTime: 2 * time.Minute,
			Environment:     "testing",
		}
	}

	// 自動最適化設定
	baseConfig.AutoOptimize = true
	baseConfig.OptimizeInterval = 30 * time.Second
	baseConfig.LoadThreshold = 0.7   // CPU使用率70%
	baseConfig.MemoryThreshold = 0.8 // メモリ使用率80%

	return &baseConfig
}

// OptimizeConnections 接続プール最適化実行
func (po *PoolOptimizer) OptimizeConnections(ctx context.Context) error {
	po.mu.Lock()
	defer po.mu.Unlock()

	// 現在のメトリクス収集
	if err := po.collectMetrics(); err != nil {
		return fmt.Errorf("メトリクス収集失敗: %v", err)
	}

	// リソース監視
	po.monitorResources()

	// 最適化判定
	optimization := po.calculateOptimalSettings()

	// 設定適用
	if err := po.applyOptimization(optimization); err != nil {
		return fmt.Errorf("最適化適用失敗: %v", err)
	}

	po.lastOptimization = time.Now()

	verboseLogging := config.GetBoolEnv("TEST_VERBOSE_LOGGING", false)
	if verboseLogging {
		po.logOptimizationResult(optimization)
	}

	return nil
}

// OptimizationResult 最適化結果
type OptimizationResult struct {
	OldMaxConnections int           `json:"old_max_connections"`
	NewMaxConnections int           `json:"new_max_connections"`
	OldMaxIdle        int           `json:"old_max_idle"`
	NewMaxIdle        int           `json:"new_max_idle"`
	OldLifetime       time.Duration `json:"old_lifetime"`
	NewLifetime       time.Duration `json:"new_lifetime"`
	Reason            string        `json:"reason"`
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       float64       `json:"memory_usage"`
}

// collectMetrics 接続プールメトリクス収集
func (po *PoolOptimizer) collectMetrics() error {
	// nil チェック
	if po == nil || po.db == nil {
		return fmt.Errorf("PoolOptimizer またはデータベース接続がnilです")
	}

	sqlDB, err := po.db.DB()
	if err != nil {
		return err
	}

	if sqlDB == nil {
		return fmt.Errorf("SQLデータベース接続がnilです")
	}

	stats := sqlDB.Stats()

	po.metrics = &PoolMetrics{
		OpenConnections:       stats.OpenConnections,
		InUseConnections:      stats.InUse,
		IdleConnections:       stats.Idle,
		WaitCount:             stats.WaitCount,
		WaitDuration:          stats.WaitDuration,
		MaxOpenConnections:    stats.MaxOpenConnections,
		ConnectionUtilization: float64(stats.InUse) / float64(stats.MaxOpenConnections),
	}

	return nil
}

// monitorResources システムリソース監視
func (po *PoolOptimizer) monitorResources() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	po.resourceMonitor = &ResourceMonitor{
		CPUUsage:    po.estimateCPUUsage(),
		MemoryUsage: float64(ms.Alloc) / float64(ms.Sys),
		GCCount:     ms.NumGC,
		Goroutines:  runtime.NumGoroutine(),
	}
}

// estimateCPUUsage CPU使用率推定（簡易版）
func (po *PoolOptimizer) estimateCPUUsage() float64 {
	// ゴルーチン数とGC頻度からCPU使用率を推定
	goroutines := runtime.NumGoroutine()
	baseCPU := float64(goroutines) / 1000.0 // ベースCPU使用率

	if baseCPU > 1.0 {
		baseCPU = 1.0
	}

	return baseCPU
}

// calculateOptimalSettings 最適な接続設定を計算
func (po *PoolOptimizer) calculateOptimalSettings() OptimizationResult {
	current := po.config
	result := OptimizationResult{
		OldMaxConnections: current.MaxConnections,
		OldMaxIdle:        current.MaxIdleConns,
		OldLifetime:       current.ConnMaxLifetime,
		CPUUsage:          po.resourceMonitor.CPUUsage,
		MemoryUsage:       po.resourceMonitor.MemoryUsage,
	}

	// 最適化ロジック
	if po.resourceMonitor.CPUUsage > po.config.LoadThreshold {
		// CPU負荷が高い場合: 接続数を削減
		result.NewMaxConnections = max(current.MinConnections, current.MaxConnections-5)
		result.NewMaxIdle = result.NewMaxConnections / 2
		result.NewLifetime = current.ConnMaxLifetime - 30*time.Second
		result.Reason = "CPU負荷軽減のため接続数削減"

	} else if po.resourceMonitor.MemoryUsage > po.config.MemoryThreshold {
		// メモリ不足の場合: 接続生存時間を短縮
		result.NewMaxConnections = current.MaxConnections
		result.NewMaxIdle = max(1, current.MaxIdleConns-2)
		result.NewLifetime = current.ConnMaxLifetime / 2
		result.Reason = "メモリ使用量削減のため生存時間短縮"

	} else if po.metrics.ConnectionUtilization > 0.8 {
		// 接続使用率が高い場合: 接続数を増加
		result.NewMaxConnections = min(100, current.MaxConnections+10)
		result.NewMaxIdle = result.NewMaxConnections / 3
		result.NewLifetime = current.ConnMaxLifetime + 30*time.Second
		result.Reason = "高使用率のため接続数増加"

	} else if po.metrics.WaitCount > 100 {
		// 待ち時間が多い場合: 接続数とアイドル数を増加
		result.NewMaxConnections = min(80, current.MaxConnections+5)
		result.NewMaxIdle = result.NewMaxConnections / 2
		result.NewLifetime = current.ConnMaxLifetime
		result.Reason = "接続待ち削減のため接続数増加"

	} else {
		// 現状維持
		result.NewMaxConnections = current.MaxConnections
		result.NewMaxIdle = current.MaxIdleConns
		result.NewLifetime = current.ConnMaxLifetime
		result.Reason = "最適状態のため現状維持"
	}

	return result
}

// applyOptimization 最適化設定を適用
func (po *PoolOptimizer) applyOptimization(optimization OptimizationResult) error {
	sqlDB, err := po.db.DB()
	if err != nil {
		return err
	}

	// 新しい設定を適用
	sqlDB.SetMaxOpenConns(optimization.NewMaxConnections)
	sqlDB.SetMaxIdleConns(optimization.NewMaxIdle)
	sqlDB.SetConnMaxLifetime(optimization.NewLifetime)

	// 設定を更新
	po.config.MaxConnections = optimization.NewMaxConnections
	po.config.MaxIdleConns = optimization.NewMaxIdle
	po.config.ConnMaxLifetime = optimization.NewLifetime

	return nil
}

// logOptimizationResult 最適化結果をログ出力
func (po *PoolOptimizer) logOptimizationResult(result OptimizationResult) {
	log.Printf("🔧 接続プール最適化実行:")
	log.Printf("   理由: %s", result.Reason)
	log.Printf("   接続数: %d → %d", result.OldMaxConnections, result.NewMaxConnections)
	log.Printf("   アイドル: %d → %d", result.OldMaxIdle, result.NewMaxIdle)
	log.Printf("   生存時間: %v → %v", result.OldLifetime, result.NewLifetime)
	log.Printf("   CPU使用率: %.1f%%", result.CPUUsage*100)
	log.Printf("   メモリ使用率: %.1f%%", result.MemoryUsage*100)
}

// StartAutoOptimization 自動最適化開始
func (po *PoolOptimizer) StartAutoOptimization(ctx context.Context) {
	if !po.config.AutoOptimize {
		return
	}

	ticker := time.NewTicker(po.config.OptimizeInterval)
	defer ticker.Stop()

	log.Printf("🤖 接続プール自動最適化開始 (間隔: %v)", po.config.OptimizeInterval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 接続プール自動最適化停止")
			return
		case <-ticker.C:
			if err := po.OptimizeConnections(ctx); err != nil {
				log.Printf("⚠️ 接続プール最適化エラー: %v", err)
			}
		}
	}
}

// GetCurrentMetrics 現在のメトリクス取得
func (po *PoolOptimizer) GetCurrentMetrics() (*PoolMetrics, *ResourceMonitor, error) {
	po.mu.RLock()
	defer po.mu.RUnlock()

	if err := po.collectMetrics(); err != nil {
		return nil, nil, err
	}
	po.monitorResources()

	return po.metrics, po.resourceMonitor, nil
}

// GetConfig 現在の設定取得
func (po *PoolOptimizer) GetConfig() *PoolConfig {
	po.mu.RLock()
	defer po.mu.RUnlock()

	config := *po.config // コピーを返す
	return &config
}

// UpdateConfig 設定更新
func (po *PoolOptimizer) UpdateConfig(newConfig *PoolConfig) error {
	po.mu.Lock()
	defer po.mu.Unlock()

	po.config = newConfig

	// 即座に新設定を適用
	optimization := OptimizationResult{
		OldMaxConnections: po.config.MaxConnections,
		NewMaxConnections: newConfig.MaxConnections,
		OldMaxIdle:        po.config.MaxIdleConns,
		NewMaxIdle:        newConfig.MaxIdleConns,
		OldLifetime:       po.config.ConnMaxLifetime,
		NewLifetime:       newConfig.ConnMaxLifetime,
		Reason:            "設定更新による適用",
	}

	return po.applyOptimization(optimization)
}

// ヘルパー関数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
