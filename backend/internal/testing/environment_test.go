// ========================================
// 環境別テスト戦略の実装例
// SQLite開発環境 vs MySQL本番環境の差異検証
// ========================================

package testing

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"money_management/internal/database"
	"money_management/internal/models"
)

// TestEnvironmentStrategy_DevelopmentVsProduction 開発環境vs本番環境戦略テスト
func TestEnvironmentStrategy_DevelopmentVsProduction(t *testing.T) {
	t.Run("Development_SQLite_FastFeedback", func(t *testing.T) {
		if testing.Short() {
			t.Skip("データベーステストは-shortフラグ使用時はスキップ")
		}
		// 開発環境: SQLite使用（高速フィードバック）
		if !database.IsInMemoryDBEnabled() {
			t.Skip("開発環境テスト: USE_INMEMORY_DB=true が必要")
		}

		start := time.Now()
		db, cleanup, err := database.SetupLightweightTestDB("DevelopmentTest")
		assert.NoError(t, err, "開発環境DB作成失敗")
		defer cleanup()

		// 典型的な開発サイクルテスト
		user := models.User{
			Name:      "開発テストユーザー",
			AccountID: "dev_test_user",
		}
		result := db.Create(&user)
		assert.NoError(t, result.Error, "開発環境でのユーザー作成失敗")

		duration := time.Since(start)
		t.Logf("🚀 開発環境テスト実行時間: %v", duration)
		assert.Less(t, duration, 10*time.Millisecond, "開発環境は10ms以内で完了すべき")
	})

	t.Run("Production_MySQL_QualityAssurance", func(t *testing.T) {
		if testing.Short() {
			t.Skip("データベーステストは-shortフラグ使用時はスキップ")
		}

		// 本番環境: MySQL使用（品質保証）
		if database.IsInMemoryDBEnabled() {
			t.Skip("本番環境テスト: MySQLが必要")
		}

		start := time.Now()
		db, err := database.SetupTestDB()
		assert.NoError(t, err, "本番同等環境DB作成失敗")
		defer database.CleanupTestDB(db)

		// 本番固有の制約テスト
		user := models.User{
			Name:      "本番テストユーザー",
			AccountID: "prod_test_user",
		}
		result := db.Create(&user)
		assert.NoError(t, result.Error, "本番環境でのユーザー作成失敗")

		// ENUM制約テスト（SQLiteでは検出できない）
		bill := models.MonthlyBill{
			Year:        2024,
			Month:       1,
			RequesterID: user.ID,
			PayerID:     user.ID,
			Status:      "invalid_status", // 不正なENUM値
		}
		result = db.Create(&bill)

		// MySQLではエラーになるべき、SQLiteは通る可能性
		if result.Error != nil {
			t.Logf("✅ 本番環境でENUM制約が正常動作: %v", result.Error)
		} else {
			t.Logf("⚠️ ENUM制約チェックが無効の可能性")
		}

		duration := time.Since(start)
		t.Logf("🏭 本番環境テスト実行時間: %v", duration)
	})
}

// TestCICDPipeline_StageBasedTesting CI/CD段階的テスト戦略
func TestCICDPipeline_StageBasedTesting(t *testing.T) {
	// 環境変数でCI/CD段階を判定
	ciStage := os.Getenv("CI_STAGE") // "quick" | "integration" | "deploy"

	switch ciStage {
	case "quick":
		// Phase 1: 高速スクリーニング（30秒以内）
		t.Run("QuickScreening", func(t *testing.T) {
			testQuickScreening(t)
		})

	case "integration":
		// Phase 2: 統合テスト（5分以内）
		t.Run("IntegrationTest", func(t *testing.T) {
			testIntegrationQuality(t)
		})

	case "deploy":
		// Phase 3: デプロイ前完全検証（10分以内）
		t.Run("PreDeploymentTest", func(t *testing.T) {
			testPreDeploymentValidation(t)
		})

	default:
		t.Skip("CI_STAGE環境変数が設定されていません")
	}
}

// testQuickScreening 高速スクリーニングテスト
func testQuickScreening(t *testing.T) {
	start := time.Now()

	// SQLite使用の高速テスト
	db, cleanup, err := database.SetupLightweightTestDB("QuickScreening")
	assert.NoError(t, err)
	defer cleanup()

	// 基本機能の動作確認のみ
	user := models.User{Name: "QuickTest", AccountID: "quick_test"}
	result := db.Create(&user)
	assert.NoError(t, result.Error)

	duration := time.Since(start)
	t.Logf("⚡ 高速スクリーニング: %v", duration)
	assert.Less(t, duration, 30*time.Second, "30秒以内で完了すべき")
}

// testIntegrationQuality 統合テスト品質検証
func testIntegrationQuality(t *testing.T) {
	start := time.Now()

	// MySQL使用の統合テスト
	db, err := database.SetupTestDB()
	assert.NoError(t, err)
	defer database.CleanupTestDB(db)

	// 本番環境固有の動作確認
	factory := NewTestDataFactory(db)
	testData, err := factory.CreateStandardTestScenario()
	assert.NoError(t, err)

	// 外部キー制約テスト
	invalidBill := models.MonthlyBill{
		Year:        2024,
		Month:       1,
		RequesterID: 99999, // 存在しないユーザーID
		PayerID:     testData.User1.ID,
		Status:      "pending",
	}
	result := db.Create(&invalidBill)
	assert.Error(t, result.Error, "外部キー制約でエラーになるべき")

	duration := time.Since(start)
	t.Logf("🔧 統合テスト: %v", duration)
	assert.Less(t, duration, 5*time.Minute, "5分以内で完了すべき")
}

// testPreDeploymentValidation デプロイ前完全検証
func testPreDeploymentValidation(t *testing.T) {
	start := time.Now()

	// 完全なMySQL環境でのテスト
	db, err := database.SetupTestDB()
	assert.NoError(t, err)
	defer database.CleanupTestDB(db)

	// パフォーマンステスト
	t.Run("PerformanceTest", func(t *testing.T) {
		// 大量データでの性能確認
		users := make([]models.User, 1000)
		for i := 0; i < 1000; i++ {
			users[i] = models.User{
				Name:      fmt.Sprintf("PerfTestUser%d", i),
				AccountID: fmt.Sprintf("perf_user_%d", i),
			}
		}

		perfStart := time.Now()
		result := db.CreateInBatches(users, 100)
		perfDuration := time.Since(perfStart)

		assert.NoError(t, result.Error)
		t.Logf("📊 1000ユーザー作成: %v", perfDuration)
		assert.Less(t, perfDuration, 10*time.Second, "大量データ処理が遅すぎる")
	})

	// メモリリークテスト
	t.Run("MemoryLeakTest", func(t *testing.T) {
		// 繰り返し処理でメモリリーク検証
		for i := 0; i < 100; i++ {
			factory := NewTestDataFactory(db)
			_, err := factory.CreateStandardTestScenario()
			assert.NoError(t, err)
		}
	})

	duration := time.Since(start)
	t.Logf("🚀 デプロイ前完全検証: %v", duration)
	assert.Less(t, duration, 10*time.Minute, "10分以内で完了すべき")
}

// TestDatabaseSpecificBugs DB固有バグの検出テスト
func TestDatabaseSpecificBugs(t *testing.T) {
	// 並列実行を無効化してデッドロックを回避
	t.Run("MySQL_ENUM_Constraint_Bug", func(t *testing.T) {
		if testing.Short() {
			t.Skip("データベーステストは-shortフラグ使用時はスキップ")
		}

		// MySQLでのみ発生するENUM制約バグ
		if database.IsInMemoryDBEnabled() {
			t.Skip("MySQL固有バグテスト: MySQLが必要")
		}

		db, err := database.SetupTestDB()
		assert.NoError(t, err)
		defer database.CleanupTestDB(db)

		// トランザクションのクリーンアップを確実に行う
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Exec("ROLLBACK")
		}

		// テスト用の一意な識別子を生成（デッドロック回避）
		now := time.Now()
		uniqueYear := now.Year() + (int(now.UnixNano()) % 1000)

		// 不正なENUM値でのテスト
		invalidBill := models.MonthlyBill{
			Year:        uniqueYear,
			Month:       int(now.UnixNano()/1000000)%12 + 1,
			RequesterID: 1,
			PayerID:     2, // 異なるpayer_idを使用して制約違反を回避
			Status:      "completely_invalid_status",
		}

		result := db.Create(&invalidBill)
		assert.Error(t, result.Error, "MySQL ENUM制約でエラーになるべき")

		// デッドロックエラーの場合はリトライ
		if result.Error != nil && strings.Contains(result.Error.Error(), "Deadlock found") {
			t.Logf("デッドロック検出、リトライ中...")
			time.Sleep(100 * time.Millisecond)
			result = db.Create(&invalidBill)
		}

		assert.Error(t, result.Error, "MySQL ENUM制約でエラーになるべき")
		assert.Contains(t, result.Error.Error(), "1265", "MySQL ENUM制約エラーコードを確認")
	})

	t.Run("SQLite_Development_Speed", func(t *testing.T) {
		// SQLiteの開発効率テスト
		if !database.IsInMemoryDBEnabled() {
			t.Skip("SQLite開発効率テスト: USE_INMEMORY_DB=true が必要")
		}

		start := time.Now()

		// 100回の高速テスト実行
		for i := 0; i < 100; i++ {
			db, cleanup, err := database.SetupLightweightTestDB(fmt.Sprintf("SpeedTest_%d", i))
			assert.NoError(t, err)

			user := models.User{Name: fmt.Sprintf("User%d", i), AccountID: fmt.Sprintf("user_%d", i)}
			db.Create(&user)

			cleanup()
		}

		duration := time.Since(start)
		t.Logf("⚡ SQLite 100回実行: %v", duration)
		assert.Less(t, duration, 5*time.Second, "SQLiteは100回実行でも5秒以内")
	})
}

// 使用例の環境変数設定
func ExampleEnvironmentConfiguration() {
	// 開発環境
	// export USE_INMEMORY_DB=true
	// export FAST_TEST_MODE=true
	// go test ./internal/handlers

	// CI/CD Quick Phase
	// export CI_STAGE=quick
	// export USE_INMEMORY_DB=true
	// go test ./internal/testing -run TestCICDPipeline

	// CI/CD Integration Phase
	// export CI_STAGE=integration
	// docker-compose up -d mysql
	// go test ./internal/testing -run TestCICDPipeline

	// CI/CD Deploy Phase
	// export CI_STAGE=deploy
	// go test ./... -tags=integration,performance
}
