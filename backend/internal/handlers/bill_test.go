// ========================================
// 家計簿ハンドラーの自動テスト
// 本番環境と同じMySQL 8.0を使用してテスト実行
// ========================================

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"money_management/internal/database"
	"money_management/internal/models"
	testfactory "money_management/internal/testing"
)

// logTestContext テストコンテキスト情報をログ出力
func logTestContext(t *testing.T, action string, details map[string]interface{}) {
	// 呼び出し元の情報を取得
	_, file, line, ok := runtime.Caller(1)
	caller := "unknown"
	if ok {
		fileParts := strings.Split(file, "/")
		if len(fileParts) > 0 {
			caller = fmt.Sprintf("%s:%d", fileParts[len(fileParts)-1], line)
		}
	}

	// 基本情報をログ出力
	logMsg := fmt.Sprintf("🔍 [%s] %s @ %s", t.Name(), action, caller)

	// 詳細情報があれば追加
	if len(details) > 0 {
		logMsg += " - "
		var pairs []string
		for k, v := range details {
			pairs = append(pairs, fmt.Sprintf("%s: %v", k, v))
		}
		logMsg += strings.Join(pairs, ", ")
	}

	log.Printf(logMsg)
}

// logTestError テストエラー情報を詳細にログ出力
func logTestError(t *testing.T, err error, context string) {
	if err == nil {
		return
	}

	errStr := err.Error()
	errorType := "一般エラー"

	// エラータイプを分類
	if strings.Contains(errStr, "1213") || strings.Contains(strings.ToLower(errStr), "deadlock") {
		errorType = "デッドロック"
	} else if strings.Contains(errStr, "1452") {
		errorType = "外部キー制約違反"
	} else if strings.Contains(errStr, "1062") {
		errorType = "重複キー制約違反"
	} else if strings.Contains(strings.ToLower(errStr), "connection") {
		errorType = "DB接続エラー"
	}

	log.Printf("❌ [%s] %s - %s: %s", t.Name(), errorType, context, errStr)
}

// setupTestDB テスト用データベースのセットアップ（詳細ログ付き）
// 本番環境と同じMySQL 8.0を使用してテスト環境を構築
// グローバル変数は変更せず、独立したDB接続を返す
func setupTestDB() (*gorm.DB, error) {
	db, err := database.SetupTestDB()
	if err != nil {
		return nil, fmt.Errorf("データベース接続エラー: %w", err)
	}

	// データベース接続の有効性を確認
	if db == nil {
		return nil, fmt.Errorf("データベース接続がnilです")
	}

	// 接続テスト
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("SQL DB接続取得エラー: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("データベースping失敗: %w", err)
	}

	return db, nil
}

// cleanupTestResources リソースの適切なクリーンアップを実行
func cleanupTestResources(db *gorm.DB) {
	if db != nil {
		// テストデータをクリーンアップ
		database.CleanupTestDB(db)

		// SQLドライバ接続を適切に閉じる
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

// TestData テスト用データを格納する構造体
type TestData struct {
	User1 models.User
	User2 models.User
	User3 models.User
	Bill  models.MonthlyBill
	Items []models.BillItem
}

// generateUniqueID 一意のIDを生成するヘルパー関数
func generateUniqueID() string {
	timestamp := time.Now().UnixNano()
	randomNum := rand.Intn(10000)
	return strconv.FormatInt(timestamp, 10) + "_" + strconv.Itoa(randomNum)
}

// generateTestUser 動的テストユーザーを生成
func generateTestUser(baseName, baseAccountID string) models.User {
	uniqueID := generateUniqueID()
	return models.User{
		Name:         baseName + "_" + uniqueID,
		AccountID:    baseAccountID + "_" + uniqueID,
		PasswordHash: "hashedpassword_" + uniqueID,
	}
}

// setupTestData テスト用のデータセットアップ（ファクトリパターン使用）
// より効率的で保守しやすいテストデータ生成システム
func setupTestData(db *gorm.DB) (*TestData, error) {
	// テストデータファクトリを初期化
	factory := testfactory.NewTestDataFactory(db)

	// 標準的なテストシナリオを生成
	standardData, err := factory.CreateStandardTestScenario()
	if err != nil {
		return nil, fmt.Errorf("テストデータファクトリでの生成に失敗: %w", err)
	}

	// 既存のTestData構造体に変換（後方互換性のため）
	data := &TestData{
		User1: standardData.User1,
		User2: standardData.User2,
		User3: standardData.User3,
		Bill:  standardData.Bill,
		Items: standardData.Items,
	}

	return data, nil
}

// setupRouter テスト用のGinルーター設定
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// setUserID コンテキストにユーザーIDを設定するミドルウェア
func setUserID(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// TestGetBillHandler_Success 請求者が有効な年月の家計簿を取得できることを検証（HTTP200と正しい家計簿情報の返却を期待）
func TestGetBillHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	// DB注入機能を使用してテスト用DB接続を使用
	router.GET("/bills/:year/:month", setUserID(testData.User1.ID), GetBillHandlerWithDB(db))

	// 正常系: 請求者としてアクセス（動的な年月を使用）
	billURL := fmt.Sprintf("/bills/%d/%d", testData.Bill.Year, testData.Bill.Month)
	req := httptest.NewRequest("GET", billURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response models.BillResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, testData.Bill.Year, response.Year)
	assert.Equal(t, testData.Bill.Month, response.Month)
	assert.Equal(t, testData.User1.ID, response.RequesterID)
	assert.Equal(t, testData.User2.ID, response.PayerID)
	// 動的に計算された金額をチェック
	expectedTotal := testData.Items[0].Amount + testData.Items[1].Amount
	assert.Equal(t, expectedTotal, response.TotalAmount)
}

// TestGetBillHandler_PayerAccess 支払者が自分に関連する家計簿を取得できることを検証（HTTP200と正しい家計簿情報の返却を期待）
func TestGetBillHandler_PayerAccess(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.GET("/bills/:year/:month", setUserID(testData.User2.ID), GetBillHandlerWithDB(db))

	// 支払者としてアクセス（動的な年月を使用）
	billURL := fmt.Sprintf("/bills/%d/%d", testData.Bill.Year, testData.Bill.Month)
	req := httptest.NewRequest("GET", billURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response models.BillResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, testData.Bill.Year, response.Year)
	assert.Equal(t, testData.Bill.Month, response.Month)
}

// TestGetBillHandler_AccessDenied 第三者が関連のない家計簿にアクセスした際に拒否されることを検証（HTTP200でbill:nullの返却を期待）
func TestGetBillHandler_AccessDenied(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.GET("/bills/:year/:month", setUserID(testData.User3.ID), GetBillHandlerWithDB(db))

	// 第三者としてアクセス（拒否されるべき）
	billURL := fmt.Sprintf("/bills/%d/%d", testData.Bill.Year, testData.Bill.Month)
	req := httptest.NewRequest("GET", billURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Nil(t, response["bill"])
}

// TestGetBillHandler_NotFound 存在しない年月の家計簿を取得しようとした際にnullが返されることを検証（HTTP200でbill:nullの返却を期待）
func TestGetBillHandler_NotFound(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.GET("/bills/:year/:month", setUserID(testData.User1.ID), GetBillHandlerWithDB(db))

	// 存在しない年月
	req := httptest.NewRequest("GET", "/bills/2023/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Nil(t, response["bill"])
}

// TestCreateBillHandler_Success 有効なリクエストで新しい家計簿が作成できることを検証（HTTP201と作成された家計簿情報の返却を期待）
func TestCreateBillHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	// テストデータの整合性確認
	if testData.User1.ID == 0 {
		t.Fatal("User1のIDが設定されていません")
	}
	if testData.User2.ID == 0 {
		t.Fatal("User2のIDが設定されていません")
	}

	t.Logf("テストデータ確認: User1 ID=%d, User2 ID=%d", testData.User1.ID, testData.User2.ID)

	router := setupRouter()
	router.POST("/bills", setUserID(testData.User1.ID), CreateBillHandlerWithDB(db))

	// テスト間の競合を避けるため動的な年月を生成
	now := time.Now()
	uniqueYear := now.Year() + (int(now.UnixNano()) % 1000) // 現在年 + ナノ秒ベースのオフセット
	uniqueMonth := (int(now.UnixNano()/1000000) % 12) + 1   // 1-12の範囲

	requestBody := map[string]interface{}{
		"year":     uniqueYear,
		"month":    uniqueMonth,
		"payer_id": testData.User2.ID,
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/bills", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "HTTPステータスが期待値と異なります")

	var response models.BillResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, uniqueYear, response.Year)
	assert.Equal(t, uniqueMonth, response.Month)
	assert.Equal(t, testData.User1.ID, response.RequesterID)
	assert.Equal(t, testData.User2.ID, response.PayerID)
	assert.Equal(t, "pending", response.Status)
}

// TestCreateBillHandler_InvalidRequest 必須フィールドが不足したリクエストでエラーが返されることを検証（HTTP400の返却を期待）
func TestCreateBillHandler_InvalidRequest(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.POST("/bills", setUserID(testData.User1.ID), CreateBillHandlerWithDB(db))

	// 不正なリクエスト（必須フィールド不足）
	requestBody := map[string]interface{}{
		"year": 2025,
		// monthとpayer_idが不足
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/bills", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTPステータスが期待値と異なります")
}

// TestCreateBillHandler_SameRequesterAndPayer 請求者と支払者が同一ユーザーの場合にエラーが返されることを検証（HTTP400の返却を期待）
func TestCreateBillHandler_SameRequesterAndPayer(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	logTestContext(t, "開始", map[string]interface{}{
		"requester_id": testData.User1.ID,
		"payer_id":     testData.User1.ID,
	})

	router := setupRouter()
	router.POST("/bills", setUserID(testData.User1.ID), CreateBillHandlerWithDB(db))

	// 請求者と支払者が同一ユーザーになるリクエスト
	requestBody := map[string]interface{}{
		"year":     2025,
		"month":    12,
		"payer_id": testData.User1.ID, // 請求者（ログインユーザー）と同じID
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/bills", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	logTestContext(t, "レスポンス確認", map[string]interface{}{
		"status_code":   w.Code,
		"response_body": w.Body.String(),
	})

	// HTTP400 Bad Requestが返されることを期待
	assert.Equal(t, http.StatusBadRequest, w.Code, "請求者と支払者が同一の場合は400エラーが返されるべきです")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "レスポンスのJSONパースに失敗しました")

	// エラーメッセージの確認
	assert.Contains(t, response["error"], "請求者と支払者は異なるユーザーである必要があります", "適切なエラーメッセージが返されるべきです")

	logTestContext(t, "完了", map[string]interface{}{
		"validation": "請求者=支払者のバリデーションが正常に動作",
	})
}

// TestUpdateItemsHandler_Success 請求者が家計簿項目を正常に更新できることを検証（HTTP200と更新された家計簿情報の返却を期待）
func TestUpdateItemsHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.PUT("/bills/:id/items", setUserID(testData.User1.ID), UpdateItemsHandlerWithDB(db))

	requestBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{"item_name": "新しい項目1", "amount": 3000},
			{"item_name": "新しい項目2", "amount": 4000},
		},
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/bills/%d/items", testData.Bill.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response models.BillResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(response.Items))
	assert.Equal(t, float64(7000), response.TotalAmount) // 3000 + 4000
}

// TestUpdateItemsHandler_AccessDenied 支払者が家計簿項目を更新しようとした際にアクセスが拒否されることを検証（HTTP404とエラーメッセージの返却を期待）
func TestUpdateItemsHandler_AccessDenied(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.PUT("/bills/:id/items", setUserID(testData.User2.ID), UpdateItemsHandlerWithDB(db)) // 支払者でアクセス

	requestBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{"item_name": "新しい項目", "amount": 1000},
		},
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/bills/%d/items", testData.Bill.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "家計簿が見つかりません", response["error"])
}

// TestRequestBillHandler_Success 請求者が家計簿の請求を正常に確定できることを検証（HTTP200とステータス更新メッセージの返却を期待）
func TestRequestBillHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.PUT("/bills/:id/request", setUserID(testData.User1.ID), RequestBillHandlerWithDB(db))

	req := httptest.NewRequest("PUT", fmt.Sprintf("/bills/%d/request", testData.Bill.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "家計簿の請求が確定しました", response["message"])

	// ステータスが更新されたことを確認
	var bill models.MonthlyBill
	err = db.First(&bill, testData.Bill.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "requested", bill.Status)
	assert.NotNil(t, bill.RequestDate)
}

// TestRequestBillHandler_AccessDenied 支払者が家計簿の請求確定を試行した際にアクセスが拒否されることを検証（HTTP404とエラーメッセージの返却を期待）
func TestRequestBillHandler_AccessDenied(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.PUT("/bills/:id/request", setUserID(testData.User2.ID), RequestBillHandlerWithDB(db)) // 支払者でアクセス

	req := httptest.NewRequest("PUT", fmt.Sprintf("/bills/%d/request", testData.Bill.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "家計簿が見つかりません", response["error"])
}

// TestPaymentBillHandler_Success 支払者が請求済み家計簿の支払いを正常に確定できることを検証（HTTP200とステータス更新メッセージの返却を期待）
func TestPaymentBillHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	// 家計簿を請求済み状態にする
	now := time.Now()
	err = db.Model(&models.MonthlyBill{}).Where("id = ?", testData.Bill.ID).Updates(models.MonthlyBill{
		Status:      "requested",
		RequestDate: &now,
	}).Error
	assert.NoError(t, err)

	router := setupRouter()
	router.PUT("/bills/:id/payment", setUserID(testData.User2.ID), PaymentBillHandlerWithDB(db))

	req := httptest.NewRequest("PUT", fmt.Sprintf("/bills/%d/payment", testData.Bill.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "支払いが確定しました", response["message"])

	// ステータスが更新されたことを確認
	var bill models.MonthlyBill
	err = db.First(&bill, testData.Bill.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "paid", bill.Status)
	assert.NotNil(t, bill.PaymentDate)
}

// TestPaymentBillHandler_AccessDenied 請求者が支払い確定を試行した際にアクセスが拒否されることを検証（HTTP404とエラーメッセージの返却を期待）
func TestPaymentBillHandler_AccessDenied(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.PUT("/bills/:id/payment", setUserID(testData.User1.ID), PaymentBillHandlerWithDB(db)) // 請求者でアクセス

	req := httptest.NewRequest("PUT", "/bills/1/payment", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "家計簿が見つかりません", response["error"])
}

// TestPaymentBillHandler_InvalidStatus pending状態の家計簿に対して支払い確定を試行した際にエラーが返されることを検証（HTTP400とエラーメッセージの返却を期待）
func TestPaymentBillHandler_InvalidStatus(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.PUT("/bills/:id/payment", setUserID(testData.User2.ID), PaymentBillHandlerWithDB(db))

	// pending状態のまま支払いを試行（requested状態でない）
	req := httptest.NewRequest("PUT", "/bills/1/payment", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "家計簿が請求中状態ではありません", response["error"])
}

// TestGetBillsListHandler_Success ユーザーが関連する全ての家計簿一覧を取得できることを検証（HTTP200と複数の家計簿情報の返却を期待）
func TestGetBillsListHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	// 追加の家計簿作成（ユーザー1が支払者）
	bill2 := models.MonthlyBill{
		Year:        2024,
		Month:       11,
		RequesterID: testData.User2.ID,
		PayerID:     testData.User1.ID,
		Status:      "paid",
	}
	err = db.Create(&bill2).Error
	assert.NoError(t, err)

	router := setupRouter()
	router.GET("/bills", setUserID(testData.User1.ID), GetBillsListHandlerWithDB(db))

	req := httptest.NewRequest("GET", "/bills", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	billsData, ok := response["bills"].([]interface{})
	if !ok || billsData == nil {
		t.Fatalf("レスポンスにbillsが含まれていないか、型が正しくありません: %+v", response)
	}
	assert.Equal(t, 2, len(billsData)) // ユーザー1が関与する家計簿が2件
}

// TestGetBillsListHandler_EmptyResult どの家計簿にも関連していないユーザーが一覧を取得した際に空の結果が返されることを検証（HTTP200でbills:nullの返却を期待）
func TestGetBillsListHandler_EmptyResult(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.GET("/bills", setUserID(testData.User3.ID), GetBillsListHandlerWithDB(db)) // ユーザー3は家計簿に関与していない

	req := httptest.NewRequest("GET", "/bills", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPステータスが期待値と異なります")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	bills := response["bills"]
	assert.Nil(t, bills) // 関与する家計簿なし
}

// TestCreateBillHandler_DuplicateEntry 同一年月の重複家計簿作成時に409エラーが返されることを検証
func TestCreateBillHandler_DuplicateEntry(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	logTestContext(t, "テスト開始", map[string]interface{}{"type": "duplicate_entry", "parallel": false})

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	// テストデータをセットアップ
	logTestContext(t, "テストデータセットアップ開始", map[string]interface{}{"phase": "setup"})
	testData, err := setupTestData(db)
	if err != nil {
		logTestError(t, err, "テストデータセットアップ")
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}
	logTestContext(t, "テストデータセットアップ完了", map[string]interface{}{
		"users_created": 3,
		"bills_created": 1,
		"items_created": len(testData.Items),
	})

	// 最初の家計簿を作成（成功するはず） - testDataとは異なる年月を使用
	firstBillReq := map[string]interface{}{
		"year":     2026,
		"month":    9,
		"payer_id": testData.User2.ID,
	}

	logTestContext(t, "最初の家計簿作成", map[string]interface{}{
		"year":     firstBillReq["year"],
		"month":    firstBillReq["month"],
		"payer_id": firstBillReq["payer_id"],
	})

	reqBody, _ := json.Marshal(firstBillReq)
	req, _ := http.NewRequest("POST", "/api/bills", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", testData.User1.ID) // 請求者として設定

	CreateBillHandlerWithDB(db)(c)

	assert.Equal(t, http.StatusCreated, w.Code, "最初の家計簿作成は成功するべき")

	// 同じ年月で重複家計簿を作成しようとする（409エラーになるはず）
	duplicateBillReq := map[string]interface{}{
		"year":     2026,
		"month":    9,
		"payer_id": testData.User3.ID, // 異なる支払者でも重複はNG
	}

	logTestContext(t, "重複家計簿作成試行", map[string]interface{}{
		"year":     duplicateBillReq["year"],
		"month":    duplicateBillReq["month"],
		"payer_id": duplicateBillReq["payer_id"],
	})

	reqBody2, _ := json.Marshal(duplicateBillReq)
	req2, _ := http.NewRequest("POST", "/api/bills", bytes.NewBuffer(reqBody2))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = req2
	c2.Set("user_id", testData.User1.ID) // 同じ請求者

	CreateBillHandlerWithDB(db)(c2)

	// 409 Conflictエラーが返されることを確認
	assert.Equal(t, http.StatusConflict, w2.Code, "重複した家計簿作成は409エラーになるべき")

	var response map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "指定された年月の家計簿は既に存在します", "適切なエラーメッセージが返されるべき")

	logTestContext(t, "テスト完了", map[string]interface{}{
		"first_status":     http.StatusCreated,
		"duplicate_status": http.StatusConflict,
		"error_message":    response["error"],
	})
}

// TestCreateBillHandler_DatabaseError データベースエラー時の家計簿作成ハンドリングをテスト
func TestCreateBillHandler_DatabaseError(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	logTestContext(t, "テスト開始", map[string]interface{}{"type": "database_error", "parallel": true})

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	// テストデータをセットアップしてからDB接続を閉じる
	logTestContext(t, "テストデータセットアップ開始", map[string]interface{}{"phase": "setup"})
	testData, err := setupTestData(db)
	if err != nil {
		logTestError(t, err, "テストデータセットアップ")
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}
	logTestContext(t, "テストデータセットアップ完了", map[string]interface{}{
		"users_created": 3,
		"bills_created": 1,
		"items_created": len(testData.Items),
	})

	// データベース接続を闉じてエラーを発生させる
	sqlDB, _ := db.DB()
	sqlDB.Close()

	router := setupRouter()
	router.POST("/bills", setUserID(testData.User1.ID), CreateBillHandlerWithDB(db))

	requestBody := map[string]interface{}{
		"year":     2025,
		"month":    1,
		"payer_id": testData.User2.ID,
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/bills", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "データベースエラー時に適切なステータスが返されませんでした")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "家計簿の作成に失敗しました", response["error"], "期待されるエラーメッセージと異なります")
}

// TestGetBillsListHandler_DatabaseError データベースエラー時の家計簿一覧取得をテスト
func TestGetBillsListHandler_DatabaseError(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	// テストデータをセットアップしてからDB接続を閉じる
	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	// データベース接続を闉じてエラーを発生させる
	sqlDB, _ := db.DB()
	sqlDB.Close()

	router := setupRouter()
	router.GET("/bills", setUserID(testData.User1.ID), GetBillsListHandlerWithDB(db))

	req := httptest.NewRequest("GET", "/bills", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "データベースエラー時に適切なステータスが返されませんでした")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "家計簿一覧の取得に失敗しました", response["error"], "期待されるエラーメッセージと異なります")
}

// TestUpdateItemsHandler_NonExistentBill 存在しない家計簿のIDで項目更新を試行
func TestUpdateItemsHandler_NonExistentBill(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	testData, err := setupTestData(db)
	if err != nil {
		t.Skipf("テストデータのセットアップに失敗、テストをスキップ: %v", err)
		return
	}

	router := setupRouter()
	router.PUT("/bills/:id/items", setUserID(testData.User1.ID), UpdateItemsHandlerWithDB(db))

	requestBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{"item_name": "テスト項目", "amount": 1000},
		},
	}
	jsonData, _ := json.Marshal(requestBody)

	// 存在しない家計簿IDでアクセス
	req := httptest.NewRequest("PUT", "/bills/99999/items", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "存在しない家計簿でアクセスが許可されました")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "家計簿が見つかりません", response["error"], "期待されるエラーメッセージと異なります")
}

// TestCreateBillHandler_FactoryDemo ファクトリパターンの柔軟性を実証するテスト
func TestCreateBillHandler_FactoryDemo(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	// ファクトリを使って柔軟にテストデータを作成
	factory := testfactory.NewTestDataFactory(db)

	// カスタムユーザーを作成
	requester, err := factory.NewUser().
		WithName("カスタム請求者").
		WithAccountID("custom_requester").
		Build()
	assert.NoError(t, err)

	payer, err := factory.NewUser().
		WithName("カスタム支払者").
		WithAccountID("custom_payer").
		Build()
	assert.NoError(t, err)

	router := setupRouter()
	router.POST("/bills", setUserID(requester.ID), CreateBillHandlerWithDB(db))

	// 家計簿作成リクエスト（現在の年月を動的取得）
	now := time.Now()
	requestBody := map[string]interface{}{
		"year":     now.Year(),
		"month":    int(now.Month()),
		"payer_id": payer.ID,
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/bills", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "ファクトリパターンでの家計簿作成が失敗しました")

	// レスポンス内容の検証（作成された家計簿オブジェクトが返される）
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 作成された家計簿の基本情報を検証
	assert.Equal(t, float64(now.Year()), response["year"])
	assert.Equal(t, float64(now.Month()), response["month"])
	assert.Equal(t, float64(requester.ID), response["requester_id"])
	assert.Equal(t, float64(payer.ID), response["payer_id"])
	assert.Equal(t, "pending", response["status"])

	// ネストされたrequesterオブジェクトの検証
	requesterObj := response["requester"].(map[string]interface{})
	assert.Equal(t, "カスタム請求者", requesterObj["name"])
	assert.Equal(t, "custom_requester", requesterObj["account_id"])

	// ネストされたpayerオブジェクトの検証
	payerObj := response["payer"].(map[string]interface{})
	assert.Equal(t, "カスタム支払者", payerObj["name"])
	assert.Equal(t, "custom_payer", payerObj["account_id"])

	// データベースでの確認
	var createdBill models.MonthlyBill
	err = db.Where("requester_id = ? AND payer_id = ?", requester.ID, payer.ID).First(&createdBill).Error
	assert.NoError(t, err, "作成された家計簿がデータベースに見つかりません")
	assert.Equal(t, now.Year(), createdBill.Year)
	assert.Equal(t, int(now.Month()), createdBill.Month)
}

// TestDeleteBillHandler_Success 作成中ステータスの家計簿を請求者が削除できることを検証
func TestDeleteBillHandler_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	factory := testfactory.NewTestDataFactory(db)

	// テストユーザーを作成
	requester, err := factory.NewUser().
		WithName("削除テスト請求者").
		WithAccountID("delete_requester").
		Build()
	assert.NoError(t, err)

	payer, err := factory.NewUser().
		WithName("削除テスト支払者").
		WithAccountID("delete_payer").
		Build()
	assert.NoError(t, err)

	// テスト用家計簿を作成（pending状態）
	bill := &models.MonthlyBill{
		Year:        2025,
		Month:       3,
		RequesterID: requester.ID,
		PayerID:     payer.ID,
		Status:      "pending",
	}
	err = db.Create(bill).Error
	assert.NoError(t, err, "テスト用家計簿の作成に失敗しました")

	// 家計簿項目も追加
	item := &models.BillItem{
		BillID:   bill.ID,
		ItemName: "テスト項目",
		Amount:   1000,
	}
	err = db.Create(item).Error
	assert.NoError(t, err, "テスト用家計簿項目の作成に失敗しました")

	// 削除リクエスト
	router := setupRouter()
	router.DELETE("/bills/:id", setUserID(requester.ID), DeleteBillHandlerWithDB(db))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/bills/%d", bill.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// レスポンス検証
	assert.Equal(t, http.StatusOK, w.Code, "削除リクエストが失敗しました")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "レスポンスJSONの解析に失敗しました")
	assert.Equal(t, "家計簿を削除しました", response["message"], "期待されるメッセージと異なります")

	// データベースから削除されていることを確認
	var deletedBill models.MonthlyBill
	err = db.Where("id = ?", bill.ID).First(&deletedBill).Error
	assert.Error(t, err, "家計簿が削除されていません")

	// 関連項目も削除されていることを確認
	var deletedItem models.BillItem
	err = db.Where("bill_id = ?", bill.ID).First(&deletedItem).Error
	assert.Error(t, err, "家計簿項目が削除されていません")
}

// TestDeleteBillHandler_NotFound 存在しない家計簿の削除でエラーが返されることを検証
func TestDeleteBillHandler_NotFound(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	factory := testfactory.NewTestDataFactory(db)
	requester, err := factory.NewUser().Build()
	assert.NoError(t, err)

	router := setupRouter()
	router.DELETE("/bills/:id", setUserID(requester.ID), DeleteBillHandlerWithDB(db))

	req := httptest.NewRequest("DELETE", "/bills/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "存在しない家計簿の削除で404エラーが返されませんでした")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "家計簿が見つかりません", response["error"])
}

// TestDeleteBillHandler_NonRequester 支払者が削除しようとした場合にエラーが返されることを検証
func TestDeleteBillHandler_NonRequester(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	factory := testfactory.NewTestDataFactory(db)

	requester, err := factory.NewUser().Build()
	assert.NoError(t, err)

	payer, err := factory.NewUser().Build()
	assert.NoError(t, err)

	// テスト用家計簿を作成
	bill := &models.MonthlyBill{
		Year:        2025,
		Month:       3,
		RequesterID: requester.ID,
		PayerID:     payer.ID,
		Status:      "pending",
	}
	err = db.Create(bill).Error
	assert.NoError(t, err)

	// 支払者として削除を試行
	router := setupRouter()
	router.DELETE("/bills/:id", setUserID(payer.ID), DeleteBillHandlerWithDB(db))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/bills/%d", bill.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "支払者による削除で404エラーが返されませんでした")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "家計簿が見つかりません", response["error"])
}

// TestDeleteBillHandler_NonPendingStatus 確定済み家計簿の削除でエラーが返されることを検証
func TestDeleteBillHandler_NonPendingStatus(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err, "テスト用データベースのセットアップに失敗しました")
	defer cleanupTestResources(db)

	factory := testfactory.NewTestDataFactory(db)

	requester, err := factory.NewUser().Build()
	assert.NoError(t, err)

	payer, err := factory.NewUser().Build()
	assert.NoError(t, err)

	// requested状態の家計簿を作成
	bill := &models.MonthlyBill{
		Year:        2025,
		Month:       4,
		RequesterID: requester.ID,
		PayerID:     payer.ID,
		Status:      "requested", // 削除不可のステータス
	}
	err = db.Create(bill).Error
	assert.NoError(t, err)

	// 請求者として削除を試行
	router := setupRouter()
	router.DELETE("/bills/:id", setUserID(requester.ID), DeleteBillHandlerWithDB(db))

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/bills/%d", bill.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "確定済み家計簿の削除で400エラーが返されませんでした")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "確定済みの家計簿は削除できません", response["error"])

	// 家計簿が削除されていないことを確認
	var stillExists models.MonthlyBill
	err = db.Where("id = ?", bill.ID).First(&stillExists).Error
	assert.NoError(t, err, "家計簿が誤って削除されました")
}
