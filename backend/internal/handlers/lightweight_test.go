// ========================================
// 軽量テストデータ生成デモンストレーション
// インメモリDB + 高速ファクトリによる超高速テスト実行
// ========================================

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"money_management/internal/database"
	"money_management/internal/models"
	testconfig "money_management/internal/testing"
)

// TestLightweightMode_LoginHandler 軽量モードログインテスト
func TestLightweightMode_LoginHandler(t *testing.T) {
	if !database.IsInMemoryDBEnabled() {
		t.Skip("インメモリDBが無効のためスキップ")
	}

	// 軽量テスト用DB接続取得
	db, cleanup, err := database.SetupLightweightTestDB("LightweightLogin")
	assert.NoError(t, err, "軽量テストDB作成に失敗")
	defer cleanup()

	// 軽量テストデータ作成
	factory := testconfig.NewTestDataFactory(db)
	testData, err := factory.CreateLightweightTestScenario()
	assert.NoError(t, err, "軽量テストデータ作成に失敗")

	// パスワードを設定（軽量版では短縮）
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	testData.User1.PasswordHash = string(hashedPassword)
	db.Save(&testData.User1)

	// ログインテスト実行
	loginRequest := models.LoginRequest{
		AccountID: testData.User1.AccountID,
		Password:  "pass",
	}

	jsonBytes, _ := json.Marshal(loginRequest)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/login", LoginHandlerWithDB(db))

	router.ServeHTTP(w, req)

	// 高速検証（必要最小限のアサーション）
	assert.Equal(t, http.StatusOK, w.Code, "ログイン失敗")

	var response models.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "レスポンス解析失敗")
	assert.NotEmpty(t, response.Token, "トークン未設定")
}

// TestLightweightMode_RegisterHandler 軽量モードユーザー登録テスト
func TestLightweightMode_RegisterHandler(t *testing.T) {
	if !database.IsInMemoryDBEnabled() {
		t.Skip("インメモリDBが無効のためスキップ")
	}

	db, cleanup, err := database.SetupLightweightTestDB("LightweightRegister")
	assert.NoError(t, err)
	defer cleanup()

	// 軽量登録データ
	registerRequest := models.RegisterRequest{
		Name:      "テストユーザー",
		AccountID: "test_user_light",
		Password:  "pass123",
	}

	jsonBytes, _ := json.Marshal(registerRequest)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/register", RegisterHandlerWithDB(db))

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// TestLightweightMode_BillCreation 軽量モード家計簿作成テスト
func TestLightweightMode_BillCreation(t *testing.T) {
	if !database.IsInMemoryDBEnabled() {
		t.Skip("インメモリDBが無効のためスキップ")
	}

	db, cleanup, err := database.SetupLightweightTestDB("LightweightBill")
	assert.NoError(t, err)
	defer cleanup()

	// 軽量テストデータ使用
	factory := testconfig.NewTestDataFactory(db)
	testData, err := factory.CreateLightweightTestScenario()
	assert.NoError(t, err)

	// 家計簿作成リクエスト
	createRequest := map[string]interface{}{
		"year":     2024,
		"month":    1,
		"payer_id": testData.User2.ID,
	}

	jsonBytes, _ := json.Marshal(createRequest)
	req, _ := http.NewRequest("POST", "/bills", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testData.User1.ID)
		c.Next()
	})
	router.POST("/bills", CreateBillHandlerWithDB(db))

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "家計簿作成失敗")
}

// TestPerformanceComparison_InMemoryVsMySQL パフォーマンス比較テスト
func TestPerformanceComparison_InMemoryVsMySQL(t *testing.T) {
	// インメモリDB vs MySQL のベンチマーク
	memoryTime, mysqlTime, speedup := database.BenchmarkInMemoryVsMySQL("PerformanceTest")

	t.Logf("🚀 パフォーマンス比較結果:")
	t.Logf("   インメモリDB: %v", memoryTime)
	t.Logf("   MySQL:        %v", mysqlTime)
	t.Logf("   速度向上:     %.1fx", speedup)

	// インメモリDBの方が高速であることを確認
	assert.True(t, memoryTime < mysqlTime, "インメモリDBがMySQLより遅い")
	assert.Greater(t, speedup, 1.0, "期待された速度向上が得られない")
}

// TestLightweightDataGeneration_Performance 軽量データ生成パフォーマンステスト
func TestLightweightDataGeneration_Performance(t *testing.T) {
	if !database.IsInMemoryDBEnabled() {
		t.Skip("インメモリDBが無効のためスキップ")
	}

	// 軽量データ生成時間測定（独立したDBを使用）
	start := time.Now()
	for i := 0; i < 5; i++ {
		db, cleanup, err := database.SetupLightweightTestDB(fmt.Sprintf("LightweightPerf_%d", i))
		assert.NoError(t, err, "軽量テストDB作成失敗")

		factory := testconfig.NewTestDataFactory(db)
		_, err = factory.CreateLightweightTestScenario()
		assert.NoError(t, err, "軽量データ生成失敗")

		cleanup()
	}
	lightweightTime := time.Since(start)

	// 通常データ生成時間測定（独立したDBを使用）
	start = time.Now()
	for i := 0; i < 5; i++ {
		db, cleanup, err := database.SetupLightweightTestDB(fmt.Sprintf("FullPerf_%d", i))
		assert.NoError(t, err, "完全テストDB作成失敗")

		factory := testconfig.NewTestDataFactory(db)
		_, err = factory.CreateFullTestScenario()
		assert.NoError(t, err, "完全データ生成失敗")

		cleanup()
	}
	fullTime := time.Since(start)

	t.Logf("📊 データ生成パフォーマンス (5回):")
	t.Logf("   軽量データ: %v", lightweightTime)
	t.Logf("   完全データ: %v", fullTime)

	if lightweightTime > 0 {
		speedup := float64(fullTime) / float64(lightweightTime)
		t.Logf("   速度向上:   %.1fx", speedup)
	}

	// データ生成が成功していることを確認
	t.Logf("✅ データ生成パフォーマンステスト完了")
}

// TestConcurrentLightweightTests 並行軽量テスト実行
func TestConcurrentLightweightTests(t *testing.T) {
	if !database.IsInMemoryDBEnabled() {
		t.Skip("インメモリDBが無効のためスキップ")
	}

	// 複数の軽量テストを並行実行
	t.Run("ConcurrentLogin", func(t *testing.T) {
		t.Parallel()
		TestLightweightMode_LoginHandler(t)
	})

	t.Run("ConcurrentRegister", func(t *testing.T) {
		t.Parallel()
		TestLightweightMode_RegisterHandler(t)
	})

	t.Run("ConcurrentBillCreate", func(t *testing.T) {
		t.Parallel()
		TestLightweightMode_BillCreation(t)
	})
}

// TestLightweightConfiguration 軽量テスト設定の動作確認
func TestLightweightConfiguration(t *testing.T) {
	config := testconfig.GetGlobalConfig()

	t.Logf("🔧 軽量テスト設定状況:")
	t.Logf("   インメモリDB: %v", config.UseInMemoryDB)
	t.Logf("   高速モード:   %v", config.FastTestMode)
	t.Logf("   並列テスト:   %v", config.ParallelTestEnabled)

	// 軽量テスト用の設定が適切に読み込まれていることを確認
	if database.IsInMemoryDBEnabled() {
		assert.True(t, config.UseInMemoryDB, "インメモリDB設定が不正")
	}
}

// TestLightweightDBCapabilities インメモリDBの機能テスト
func TestLightweightDBCapabilities(t *testing.T) {
	if !database.IsInMemoryDBEnabled() {
		t.Skip("インメモリDBが無効のためスキップ")
	}

	db, cleanup, err := database.SetupLightweightTestDB("DBCapabilities")
	assert.NoError(t, err)
	defer cleanup()

	// トランザクション動作確認
	err = db.Transaction(func(tx *gorm.DB) error {
		user := models.User{
			Name:      "トランザクションテストユーザー",
			AccountID: "tx_test_user",
		}
		return tx.Create(&user).Error
	})
	assert.NoError(t, err, "トランザクション失敗")

	// 外部キー制約確認
	factory := testconfig.NewTestDataFactory(db)
	testData, err := factory.CreateLightweightTestScenario()
	assert.NoError(t, err)

	// 家計簿が正しくユーザーに関連付けられているか確認
	var bill models.MonthlyBill
	err = db.Preload("Requester").Preload("Payer").First(&bill, testData.Bill.ID).Error
	assert.NoError(t, err, "関連データロード失敗")
	assert.Equal(t, testData.User1.ID, bill.Requester.ID, "請求者関連失敗")
	assert.Equal(t, testData.User2.ID, bill.Payer.ID, "支払者関連失敗")

	t.Logf("✅ インメモリDB機能確認完了")
}
