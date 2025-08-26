// ========================================
// テストスキップ最適化システム
// 条件付きテスト実行と環境依存テスト分離
// ========================================

package testing

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// TestSkipOptimizer テストスキップ最適化器
type TestSkipOptimizer struct {
	mu             sync.RWMutex
	skipConditions map[string]*SkipCondition
	skipHistory    []*SkipEvent
	environment    *TestEnvironment
	skipStats      *SkipStatistics
}

// SkipCondition スキップ条件
type SkipCondition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Condition   func() bool       `json:"-"` // 条件判定関数
	Category    SkipCategory      `json:"category"`
	Priority    int               `json:"priority"` // 1-10 (高優先度ほどスキップしにくい)
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
	LastChecked time.Time         `json:"last_checked"`
	CheckCount  int               `json:"check_count"`
	SkipCount   int               `json:"skip_count"`
	Metadata    map[string]string `json:"metadata"`
}

// SkipCategory スキップカテゴリ
type SkipCategory string

const (
	CategoryEnvironment SkipCategory = "environment"  // 環境依存
	CategoryResource    SkipCategory = "resource"     // リソース依存
	CategoryIntegration SkipCategory = "integration"  // 統合テスト
	CategoryPerformance SkipCategory = "performance"  // パフォーマンス
	CategoryFeatureFlag SkipCategory = "feature_flag" // フィーチャーフラグ
	CategoryPlatform    SkipCategory = "platform"     // プラットフォーム依存
)

// SkipEvent スキップイベント
type SkipEvent struct {
	TestName      string        `json:"test_name"`
	SkipReason    string        `json:"skip_reason"`
	Category      SkipCategory  `json:"category"`
	Timestamp     time.Time     `json:"timestamp"`
	Environment   string        `json:"environment"`
	Duration      time.Duration `json:"duration"`
	ShouldHaveRun bool          `json:"should_have_run"` // 本来実行すべきだったか
}

// TestEnvironment テスト実行環境
type TestEnvironment struct {
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	GoVersion   string            `json:"go_version"`
	CPUCount    int               `json:"cpu_count"`
	HasDocker   bool              `json:"has_docker"`
	HasMySQL    bool              `json:"has_mysql"`
	IsCI        bool              `json:"is_ci"`
	Branch      string            `json:"branch"`
	Environment string            `json:"environment"`
	EnvVars     map[string]string `json:"env_vars"`
}

// SkipStatistics スキップ統計
type SkipStatistics struct {
	TotalTests    int                            `json:"total_tests"`
	SkippedTests  int                            `json:"skipped_tests"`
	ExecutedTests int                            `json:"executed_tests"`
	SkipRatio     float64                        `json:"skip_ratio"`
	TimesSaved    time.Duration                  `json:"times_saved"`
	CategoryStats map[SkipCategory]*CategoryStat `json:"category_stats"`
	LastUpdated   time.Time                      `json:"last_updated"`
}

// CategoryStat カテゴリ別統計
type CategoryStat struct {
	TotalSkips  int           `json:"total_skips"`
	TimeSaved   time.Duration `json:"time_saved"`
	AvgSkipTime time.Duration `json:"avg_skip_time"`
}

var (
	globalSkipOptimizer *TestSkipOptimizer
	skipOptimizerOnce   sync.Once
)

// GetGlobalSkipOptimizer グローバルスキップ最適化器取得
func GetGlobalSkipOptimizer() *TestSkipOptimizer {
	skipOptimizerOnce.Do(func() {
		globalSkipOptimizer = &TestSkipOptimizer{
			skipConditions: make(map[string]*SkipCondition),
			skipHistory:    make([]*SkipEvent, 0),
			environment:    detectTestEnvironment(),
			skipStats: &SkipStatistics{
				CategoryStats: make(map[SkipCategory]*CategoryStat),
			},
		}
		globalSkipOptimizer.initializeDefaultConditions()
	})
	return globalSkipOptimizer
}

// detectTestEnvironment テスト実行環境検出
func detectTestEnvironment() *TestEnvironment {
	env := &TestEnvironment{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		CPUCount:  runtime.NumCPU(),
		EnvVars:   make(map[string]string),
	}

	// CI環境検出
	env.IsCI = os.Getenv("CI") == "true" ||
		os.Getenv("GITHUB_ACTIONS") == "true" ||
		os.Getenv("GITLAB_CI") == "true"

	// Docker検出
	env.HasDocker = checkDockerAvailability()

	// MySQL検出
	env.HasMySQL = checkMySQLAvailability()

	// ブランチ検出
	env.Branch = os.Getenv("GITHUB_REF_NAME")
	if env.Branch == "" {
		env.Branch = "unknown"
	}

	// 環境種別判定
	if env.IsCI {
		env.Environment = "ci"
	} else {
		env.Environment = "development"
	}

	// 重要な環境変数記録
	importantVars := []string{
		"USE_INMEMORY_DB", "ENABLE_PARALLEL_TESTS", "FAST_TEST_MODE",
		"TEST_SKIP_SLOW", "TEST_SKIP_INTEGRATION", "TEST_SKIP_EXTERNAL",
	}
	for _, v := range importantVars {
		if val := os.Getenv(v); val != "" {
			env.EnvVars[v] = val
		}
	}

	return env
}

// checkDockerAvailability Docker利用可能性チェック
func checkDockerAvailability() bool {
	// 簡易チェック：docker-compose.ymlファイルの存在確認
	if _, err := os.Stat("docker-compose.yml"); err == nil {
		return true
	}
	if _, err := os.Stat("../docker-compose.yml"); err == nil {
		return true
	}
	return false
}

// checkMySQLAvailability MySQL利用可能性チェック
func checkMySQLAvailability() bool {
	// 環境変数や設定からMySQL利用可能性を判定
	return os.Getenv("TEST_DB_HOST") != "" ||
		os.Getenv("USE_INMEMORY_DB") != "true"
}

// initializeDefaultConditions デフォルトスキップ条件初期化
func (tso *TestSkipOptimizer) initializeDefaultConditions() {
	// 1. 重いテストのスキップ
	tso.RegisterSkipCondition("skip_slow_tests", &SkipCondition{
		Name:        "重いテストスキップ",
		Description: "TEST_SKIP_SLOW=true時に実行時間の長いテストをスキップ",
		Category:    CategoryPerformance,
		Priority:    3,
		Condition: func() bool {
			return os.Getenv("TEST_SKIP_SLOW") == "true"
		},
		Tags:     []string{"performance", "slow"},
		Metadata: map[string]string{"threshold": "30s"},
	})

	// 2. 統合テストのスキップ
	tso.RegisterSkipCondition("skip_integration_tests", &SkipCondition{
		Name:        "統合テストスキップ",
		Description: "TEST_SKIP_INTEGRATION=true時に統合テストをスキップ",
		Category:    CategoryIntegration,
		Priority:    5,
		Condition: func() bool {
			return os.Getenv("TEST_SKIP_INTEGRATION") == "true"
		},
		Tags: []string{"integration", "database"},
	})

	// 3. 外部依存テストのスキップ
	tso.RegisterSkipCondition("skip_external_tests", &SkipCondition{
		Name:        "外部依存テストスキップ",
		Description: "外部サービスに依存するテストをスキップ",
		Category:    CategoryEnvironment,
		Priority:    4,
		Condition: func() bool {
			return os.Getenv("TEST_SKIP_EXTERNAL") == "true" || !tso.environment.HasDocker
		},
		Tags: []string{"external", "docker"},
	})

	// 4. 並列テスト非対応のスキップ
	tso.RegisterSkipCondition("skip_non_parallel_tests", &SkipCondition{
		Name:        "非並列テストスキップ",
		Description: "並列実行に対応していないテストをスキップ",
		Category:    CategoryResource,
		Priority:    6,
		Condition: func() bool {
			return os.Getenv("ENABLE_PARALLEL_TESTS") == "true"
		},
		Tags: []string{"parallel", "compatibility"},
	})

	// 5. プラットフォーム依存テストのスキップ
	tso.RegisterSkipCondition("skip_platform_specific", &SkipCondition{
		Name:        "プラットフォーム固有テストスキップ",
		Description: "特定のOS/アーキテクチャでのみ動作するテストをスキップ",
		Category:    CategoryPlatform,
		Priority:    7,
		Condition: func() bool {
			// Windows固有テストをLinux/macOSでスキップなど
			return false // デフォルトは実行
		},
		Tags: []string{"platform", "os"},
	})

	// 6. リソース不足時のスキップ
	tso.RegisterSkipCondition("skip_resource_intensive", &SkipCondition{
		Name:        "リソース集約テストスキップ",
		Description: "メモリ/CPU不足時にリソース集約テストをスキップ",
		Category:    CategoryResource,
		Priority:    2,
		Condition: func() bool {
			// 利用可能メモリが少ない場合など
			return tso.environment.CPUCount < 2
		},
		Tags:     []string{"resource", "memory", "cpu"},
		Metadata: map[string]string{"min_cpu": "2", "min_memory": "4GB"},
	})
}

// RegisterSkipCondition スキップ条件登録
func (tso *TestSkipOptimizer) RegisterSkipCondition(name string, condition *SkipCondition) {
	tso.mu.Lock()
	defer tso.mu.Unlock()

	condition.Name = name
	condition.CreatedAt = time.Now()
	tso.skipConditions[name] = condition

	log.Printf("📋 スキップ条件登録: %s (%s)", name, condition.Description)
}

// ShouldSkipTest テストスキップ判定
func (tso *TestSkipOptimizer) ShouldSkipTest(testName string, tags []string) (bool, string, *SkipCondition) {
	tso.mu.Lock()
	defer tso.mu.Unlock()

	for conditionName, condition := range tso.skipConditions {
		condition.CheckCount++
		condition.LastChecked = time.Now()

		// 条件チェック
		if condition.Condition() {
			// タグマッチング確認
			if tso.matchesTags(condition.Tags, tags) || len(condition.Tags) == 0 {
				condition.SkipCount++

				skipReason := fmt.Sprintf("%s (%s)", condition.Description, conditionName)
				tso.recordSkipEvent(testName, skipReason, condition.Category, true)

				log.Printf("⏭️  テストスキップ: %s - %s", testName, skipReason)
				return true, skipReason, condition
			}
		}
	}

	// スキップしない場合も記録
	tso.recordSkipEvent(testName, "", CategoryEnvironment, false)
	return false, "", nil
}

// matchesTags タグマッチング
func (tso *TestSkipOptimizer) matchesTags(conditionTags, testTags []string) bool {
	if len(conditionTags) == 0 {
		return true // 条件タグが空の場合は全て対象
	}

	for _, conditionTag := range conditionTags {
		for _, testTag := range testTags {
			if strings.EqualFold(conditionTag, testTag) {
				return true
			}
		}
	}
	return false
}

// recordSkipEvent スキップイベント記録
func (tso *TestSkipOptimizer) recordSkipEvent(testName, reason string, category SkipCategory, skipped bool) {
	event := &SkipEvent{
		TestName:      testName,
		SkipReason:    reason,
		Category:      category,
		Timestamp:     time.Now(),
		Environment:   tso.environment.Environment,
		ShouldHaveRun: !skipped,
	}

	tso.skipHistory = append(tso.skipHistory, event)

	// 履歴サイズ制限
	if len(tso.skipHistory) > 1000 {
		tso.skipHistory = tso.skipHistory[100:] // 古い100件削除
	}

	// 統計更新
	tso.updateStatistics(event, skipped)
}

// updateStatistics 統計更新
func (tso *TestSkipOptimizer) updateStatistics(event *SkipEvent, skipped bool) {
	tso.skipStats.TotalTests++

	if skipped {
		tso.skipStats.SkippedTests++

		// カテゴリ別統計更新
		if _, exists := tso.skipStats.CategoryStats[event.Category]; !exists {
			tso.skipStats.CategoryStats[event.Category] = &CategoryStat{}
		}

		categoryStats := tso.skipStats.CategoryStats[event.Category]
		categoryStats.TotalSkips++
		categoryStats.TimeSaved += 5 * time.Second // 推定節約時間
	} else {
		tso.skipStats.ExecutedTests++
	}

	// スキップ率計算
	if tso.skipStats.TotalTests > 0 {
		tso.skipStats.SkipRatio = float64(tso.skipStats.SkippedTests) / float64(tso.skipStats.TotalTests)
	}

	tso.skipStats.LastUpdated = time.Now()
}

// GetSkipStatistics スキップ統計取得
func (tso *TestSkipOptimizer) GetSkipStatistics() *SkipStatistics {
	tso.mu.RLock()
	defer tso.mu.RUnlock()

	// コピーを返す
	stats := *tso.skipStats
	stats.CategoryStats = make(map[SkipCategory]*CategoryStat)
	for k, v := range tso.skipStats.CategoryStats {
		statsCopy := *v
		stats.CategoryStats[k] = &statsCopy
	}

	return &stats
}

// GetSkipHistory スキップ履歴取得
func (tso *TestSkipOptimizer) GetSkipHistory(limit int) []*SkipEvent {
	tso.mu.RLock()
	defer tso.mu.RUnlock()

	historyLen := len(tso.skipHistory)
	if limit <= 0 || limit > historyLen {
		limit = historyLen
	}

	// 最新の履歴を返す
	start := historyLen - limit
	history := make([]*SkipEvent, limit)
	copy(history, tso.skipHistory[start:])

	return history
}

// GenerateSkipReport スキップレポート生成
func (tso *TestSkipOptimizer) GenerateSkipReport() string {
	tso.mu.RLock()
	defer tso.mu.RUnlock()

	stats := tso.skipStats
	env := tso.environment

	report := fmt.Sprintf(`📊 テストスキップ最適化レポート
======================================

🖥️ 実行環境:
   OS/Arch:        %s/%s
   Go Version:     %s
   CPU Count:      %d
   Environment:    %s
   CI Mode:        %t
   Docker:         %t
   MySQL:          %t

📈 実行統計:
   総テスト数:     %d
   実行テスト数:   %d
   スキップ数:     %d
   スキップ率:     %.1f%%
   推定節約時間:   %v

📋 カテゴリ別統計:
`,
		env.OS, env.Arch, env.GoVersion, env.CPUCount,
		env.Environment, env.IsCI, env.HasDocker, env.HasMySQL,
		stats.TotalTests, stats.ExecutedTests, stats.SkippedTests,
		stats.SkipRatio*100, stats.TimesSaved)

	for category, categoryStats := range stats.CategoryStats {
		report += fmt.Sprintf("   %s: %d回スキップ (節約: %v)\n",
			category, categoryStats.TotalSkips, categoryStats.TimeSaved)
	}

	report += fmt.Sprintf("\n🔧 登録スキップ条件:\n")
	for name, condition := range tso.skipConditions {
		efficiency := 0.0
		if condition.CheckCount > 0 {
			efficiency = float64(condition.SkipCount) / float64(condition.CheckCount) * 100
		}

		report += fmt.Sprintf("   %s: 効率%.1f%% (%d/%d)\n",
			name, efficiency, condition.SkipCount, condition.CheckCount)
	}

	report += fmt.Sprintf("\n📅 生成日時: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	return report
}

// OptimizeSkipConditions スキップ条件最適化
func (tso *TestSkipOptimizer) OptimizeSkipConditions() {
	tso.mu.Lock()
	defer tso.mu.Unlock()

	log.Printf("🔧 スキップ条件最適化開始")

	optimized := 0
	for name, condition := range tso.skipConditions {
		if condition.CheckCount < 10 {
			continue // 十分なデータなし
		}

		efficiency := float64(condition.SkipCount) / float64(condition.CheckCount)

		// 効率が低い条件の優先度を下げる
		if efficiency < 0.1 && condition.Priority > 1 {
			condition.Priority--
			optimized++
			log.Printf("📉 条件 '%s' の優先度を下げました (効率: %.1f%%)", name, efficiency*100)
		}

		// 効率が高い条件の優先度を上げる
		if efficiency > 0.8 && condition.Priority < 10 {
			condition.Priority++
			optimized++
			log.Printf("📈 条件 '%s' の優先度を上げました (効率: %.1f%%)", name, efficiency*100)
		}
	}

	log.Printf("✅ スキップ条件最適化完了: %d件の条件を調整", optimized)
}

// GetEnvironment 環境情報取得
func (tso *TestSkipOptimizer) GetEnvironment() *TestEnvironment {
	tso.mu.RLock()
	defer tso.mu.RUnlock()

	// コピーを返す
	env := *tso.environment
	env.EnvVars = make(map[string]string)
	for k, v := range tso.environment.EnvVars {
		env.EnvVars[k] = v
	}

	return &env
}
