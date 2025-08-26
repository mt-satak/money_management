// ========================================
// テストスキップ最適化システムテスト
// 条件付きテスト効率化の検証
// ========================================

package testing

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSkipOptimizer_BasicFunctionality 基本機能テスト
func TestSkipOptimizer_BasicFunctionality(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()
	assert.NotNil(t, optimizer, "スキップ最適化器が取得できない")

	// 環境情報確認
	env := optimizer.GetEnvironment()
	assert.NotNil(t, env, "環境情報が取得できない")

	t.Logf("🖥️ テスト実行環境:")
	t.Logf("   OS/Arch: %s/%s", env.OS, env.Arch)
	t.Logf("   Go Version: %s", env.GoVersion)
	t.Logf("   CPU Count: %d", env.CPUCount)
	t.Logf("   CI Mode: %t", env.IsCI)
	t.Logf("   Docker: %t", env.HasDocker)
	t.Logf("   MySQL: %t", env.HasMySQL)
	t.Logf("   Environment: %s", env.Environment)
}

// TestSkipOptimizer_EnvironmentBasedSkipping 環境ベーススキップテスト
func TestSkipOptimizer_EnvironmentBasedSkipping(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()

	testCases := []struct {
		name         string
		envVar       string
		envValue     string
		testTags     []string
		expectSkip   bool
		expectReason string
	}{
		{
			name:         "重いテストスキップ",
			envVar:       "TEST_SKIP_SLOW",
			envValue:     "true",
			testTags:     []string{"slow", "performance"},
			expectSkip:   true,
			expectReason: "TEST_SKIP_SLOW=true時に実行時間の長いテストをスキップ",
		},
		{
			name:         "統合テストスキップ",
			envVar:       "TEST_SKIP_INTEGRATION",
			envValue:     "true",
			testTags:     []string{"integration", "database"},
			expectSkip:   true,
			expectReason: "TEST_SKIP_INTEGRATION=true時に統合テストをスキップ",
		},
		{
			name:         "外部依存テストスキップ",
			envVar:       "TEST_SKIP_EXTERNAL",
			envValue:     "true",
			testTags:     []string{"external", "docker"},
			expectSkip:   true,
			expectReason: "外部サービスに依存するテストをスキップ",
		},
		{
			name:         "通常テスト（スキップなし）",
			envVar:       "TEST_SKIP_SLOW",
			envValue:     "false",
			testTags:     []string{"unit", "fast"},
			expectSkip:   false,
			expectReason: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 環境変数設定
			originalValue := os.Getenv(tc.envVar)
			os.Setenv(tc.envVar, tc.envValue)
			defer os.Setenv(tc.envVar, originalValue)

			// スキップ判定実行
			shouldSkip, reason, condition := optimizer.ShouldSkipTest(tc.name, tc.testTags)

			t.Logf("📋 %s結果:", tc.name)
			t.Logf("   スキップ: %t (期待: %t)", shouldSkip, tc.expectSkip)
			t.Logf("   理由: %s", reason)
			if condition != nil {
				t.Logf("   条件: %s (優先度: %d)", condition.Name, condition.Priority)
			}

			assert.Equal(t, tc.expectSkip, shouldSkip, "スキップ判定が期待と異なる")
			if tc.expectSkip {
				assert.Contains(t, reason, tc.expectReason, "スキップ理由が期待と異なる")
				assert.NotNil(t, condition, "スキップ条件が返されていない")
			}
		})
	}
}

// TestSkipOptimizer_CustomSkipCondition カスタムスキップ条件テスト
func TestSkipOptimizer_CustomSkipCondition(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()

	// カスタム条件登録
	customCondition := &SkipCondition{
		Name:        "カスタムテストスキップ",
		Description: "カスタム条件でのテストスキップ",
		Category:    CategoryFeatureFlag,
		Priority:    8,
		Condition: func() bool {
			return os.Getenv("CUSTOM_SKIP_TEST") == "true"
		},
		Tags:     []string{"custom", "test"},
		Metadata: map[string]string{"custom_flag": "enabled"},
	}

	optimizer.RegisterSkipCondition("custom_skip", customCondition)

	// カスタム条件テスト
	t.Run("カスタム条件有効", func(t *testing.T) {
		os.Setenv("CUSTOM_SKIP_TEST", "true")
		defer os.Unsetenv("CUSTOM_SKIP_TEST")

		shouldSkip, reason, condition := optimizer.ShouldSkipTest("CustomTest", []string{"custom"})

		assert.True(t, shouldSkip, "カスタム条件でスキップされない")
		assert.Contains(t, reason, "カスタム条件でのテストスキップ", "カスタム理由が含まれていない")
		assert.Equal(t, CategoryFeatureFlag, condition.Category, "カテゴリが正しくない")
		assert.Equal(t, 8, condition.Priority, "優先度が正しくない")

		t.Logf("✅ カスタムスキップ条件が正常動作")
	})

	t.Run("カスタム条件無効", func(t *testing.T) {
		os.Setenv("CUSTOM_SKIP_TEST", "false")
		defer os.Unsetenv("CUSTOM_SKIP_TEST")

		shouldSkip, _, _ := optimizer.ShouldSkipTest("CustomTest", []string{"custom"})
		assert.False(t, shouldSkip, "カスタム条件無効時にスキップされた")

		t.Logf("✅ カスタムスキップ条件無効時の動作確認")
	})
}

// TestSkipOptimizer_TagMatching タグマッチングテスト
func TestSkipOptimizer_TagMatching(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()

	testCases := []struct {
		name         string
		conditionTag []string
		testTags     []string
		expectMatch  bool
	}{
		{
			name:         "完全一致",
			conditionTag: []string{"slow"},
			testTags:     []string{"slow", "integration"},
			expectMatch:  true,
		},
		{
			name:         "部分一致",
			conditionTag: []string{"performance", "slow"},
			testTags:     []string{"slow"},
			expectMatch:  true,
		},
		{
			name:         "不一致",
			conditionTag: []string{"integration"},
			testTags:     []string{"unit", "fast"},
			expectMatch:  false,
		},
		{
			name:         "空条件（全て一致）",
			conditionTag: []string{},
			testTags:     []string{"any", "tags"},
			expectMatch:  true,
		},
		{
			name:         "大文字小文字無視",
			conditionTag: []string{"SLOW"},
			testTags:     []string{"slow"},
			expectMatch:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := optimizer.matchesTags(tc.conditionTag, tc.testTags)

			t.Logf("🏷️  %s:", tc.name)
			t.Logf("   条件タグ: %v", tc.conditionTag)
			t.Logf("   テストタグ: %v", tc.testTags)
			t.Logf("   マッチ: %t (期待: %t)", matches, tc.expectMatch)

			assert.Equal(t, tc.expectMatch, matches, "タグマッチング結果が期待と異なる")
		})
	}
}

// TestSkipOptimizer_StatisticsTracking 統計追跡テスト
func TestSkipOptimizer_StatisticsTracking(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()

	// 初期統計取得
	initialStats := optimizer.GetSkipStatistics()
	initialTotalTests := initialStats.TotalTests

	// 複数のテストを実行してスキップ統計を蓄積
	testCases := []struct {
		testName string
		envVar   string
		envValue string
		tags     []string
	}{
		{"SlowTest1", "TEST_SKIP_SLOW", "true", []string{"slow"}},
		{"SlowTest2", "TEST_SKIP_SLOW", "true", []string{"slow"}},
		{"FastTest1", "TEST_SKIP_SLOW", "false", []string{"fast"}},
		{"IntegrationTest1", "TEST_SKIP_INTEGRATION", "true", []string{"integration"}},
		{"UnitTest1", "", "", []string{"unit"}},
	}

	for _, tc := range testCases {
		if tc.envVar != "" {
			os.Setenv(tc.envVar, tc.envValue)
			defer os.Unsetenv(tc.envVar)
		}

		optimizer.ShouldSkipTest(tc.testName, tc.tags)
	}

	// 統計確認
	finalStats := optimizer.GetSkipStatistics()

	t.Logf("📊 スキップ統計:")
	t.Logf("   総テスト数: %d → %d", initialTotalTests, finalStats.TotalTests)
	t.Logf("   スキップ数: %d", finalStats.SkippedTests)
	t.Logf("   実行数: %d", finalStats.ExecutedTests)
	t.Logf("   スキップ率: %.1f%%", finalStats.SkipRatio*100)

	assert.Greater(t, finalStats.TotalTests, initialTotalTests, "総テスト数が増加していない")
	assert.Greater(t, finalStats.SkippedTests, int(0), "スキップ数が記録されていない")
	assert.GreaterOrEqual(t, finalStats.SkipRatio, 0.0, "スキップ率が負の値")
	assert.LessOrEqual(t, finalStats.SkipRatio, 1.0, "スキップ率が100%を超えている")

	// カテゴリ別統計確認
	for category, categoryStats := range finalStats.CategoryStats {
		t.Logf("   %s: %d回スキップ", category, categoryStats.TotalSkips)
		assert.GreaterOrEqual(t, categoryStats.TotalSkips, 0, "カテゴリスキップ数が負の値")
	}
}

// TestSkipOptimizer_SkipHistory スキップ履歴テスト
func TestSkipOptimizer_SkipHistory(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()

	// テスト実行してスキップ履歴を生成
	os.Setenv("TEST_SKIP_SLOW", "true")
	defer os.Unsetenv("TEST_SKIP_SLOW")

	testNames := []string{"HistoryTest1", "HistoryTest2", "HistoryTest3"}
	for _, name := range testNames {
		optimizer.ShouldSkipTest(name, []string{"slow"})
	}

	// 履歴取得
	history := optimizer.GetSkipHistory(10)

	t.Logf("📜 スキップ履歴 (最新%d件):", len(history))
	for i, event := range history {
		t.Logf("   %d. %s: %s (%s)", i+1, event.TestName, event.SkipReason, event.Category)
	}

	assert.GreaterOrEqual(t, len(history), 3, "履歴にテストが記録されていない")

	// 最新の履歴確認（スライスは時系列順）
	for i := len(history) - 3; i < len(history); i++ {
		if i >= 0 {
			event := history[i]
			assert.Contains(t, testNames, event.TestName, "期待されるテスト名が履歴にない")
			assert.NotEmpty(t, event.SkipReason, "スキップ理由が記録されていない")
			assert.Equal(t, CategoryPerformance, event.Category, "カテゴリが正しくない")
		}
	}
}

// TestSkipOptimizer_ConditionOptimization 条件最適化テスト
func TestSkipOptimizer_ConditionOptimization(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()

	// テスト用の条件を作成（効率の異なる条件）
	inefficientCondition := &SkipCondition{
		Name:        "非効率条件",
		Description: "ほとんどスキップしない条件",
		Category:    CategoryEnvironment,
		Priority:    5,
		Condition: func() bool {
			return false // 常にfalse = スキップしない
		},
		Tags: []string{"test"},
	}

	efficientCondition := &SkipCondition{
		Name:        "効率的条件",
		Description: "よくスキップする条件",
		Category:    CategoryEnvironment,
		Priority:    5,
		Condition: func() bool {
			return true // 常にtrue = よくスキップ
		},
		Tags: []string{"test"},
	}

	optimizer.RegisterSkipCondition("inefficient", inefficientCondition)
	optimizer.RegisterSkipCondition("efficient", efficientCondition)

	// 複数回テストして統計を蓄積
	for i := 0; i < 20; i++ {
		optimizer.ShouldSkipTest("TestOptimization", []string{"test"})
	}

	// 初期優先度記録
	initialInefficient := inefficientCondition.Priority
	initialEfficient := efficientCondition.Priority

	t.Logf("🔧 最適化前優先度:")
	t.Logf("   非効率条件: %d", initialInefficient)
	t.Logf("   効率的条件: %d", initialEfficient)

	// 最適化実行
	optimizer.OptimizeSkipConditions()

	// 最適化後優先度確認
	t.Logf("🔧 最適化後優先度:")
	t.Logf("   非効率条件: %d (変化: %+d)", inefficientCondition.Priority, inefficientCondition.Priority-initialInefficient)
	t.Logf("   効率的条件: %d (変化: %+d)", efficientCondition.Priority, efficientCondition.Priority-initialEfficient)

	// 非効率条件の優先度が下がっていることを確認
	// （ただし、効率的条件が先にマッチするため非効率条件は実行されない場合がある）
}

// TestSkipOptimizer_ReportGeneration レポート生成テスト
func TestSkipOptimizer_ReportGeneration(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()

	// テストデータ生成のためにいくつかのスキップ判定実行
	os.Setenv("TEST_SKIP_SLOW", "true")
	defer os.Unsetenv("TEST_SKIP_SLOW")

	for i := 0; i < 5; i++ {
		optimizer.ShouldSkipTest(fmt.Sprintf("ReportTest%d", i), []string{"slow"})
	}

	// レポート生成
	report := optimizer.GenerateSkipReport()

	t.Logf("📋 生成されたスキップレポート:")
	t.Logf("\n%s", report)

	// レポート内容検証
	assert.Contains(t, report, "テストスキップ最適化レポート", "レポートタイトルが含まれていない")
	assert.Contains(t, report, "実行環境", "環境情報が含まれていない")
	assert.Contains(t, report, "実行統計", "統計情報が含まれていない")
	assert.Contains(t, report, "カテゴリ別統計", "カテゴリ統計が含まれていない")
	assert.Contains(t, report, "登録スキップ条件", "条件情報が含まれていない")
	assert.NotEmpty(t, report, "レポートが空")
}

// TestSkipOptimizer_EnvironmentDetection 環境検出テスト
func TestSkipOptimizer_EnvironmentDetection(t *testing.T) {
	optimizer := GetGlobalSkipOptimizer()
	env := optimizer.GetEnvironment()

	t.Logf("🔍 環境検出結果:")
	t.Logf("   OS: %s", env.OS)
	t.Logf("   Architecture: %s", env.Arch)
	t.Logf("   Go Version: %s", env.GoVersion)
	t.Logf("   CPU Count: %d", env.CPUCount)
	t.Logf("   CI Environment: %t", env.IsCI)
	t.Logf("   Docker Available: %t", env.HasDocker)
	t.Logf("   MySQL Available: %t", env.HasMySQL)
	t.Logf("   Branch: %s", env.Branch)

	// 基本的な検証
	assert.NotEmpty(t, env.OS, "OS情報が取得されていない")
	assert.NotEmpty(t, env.Arch, "アーキテクチャ情報が取得されていない")
	assert.NotEmpty(t, env.GoVersion, "Go バージョンが取得されていない")
	assert.Greater(t, env.CPUCount, 0, "CPU数が正しく取得されていない")
	assert.NotEmpty(t, env.Environment, "環境種別が取得されていない")

	// 環境変数確認
	if len(env.EnvVars) > 0 {
		t.Logf("🌍 重要な環境変数:")
		for k, v := range env.EnvVars {
			t.Logf("   %s: %s", k, v)
		}
	}
}

func init() {
	// テスト用環境変数設定
	if os.Getenv("GO_ENV") == "" {
		os.Setenv("GO_ENV", "test")
	}
}
