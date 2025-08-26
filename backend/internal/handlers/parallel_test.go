// ========================================
// 並列テスト実行デモ - ハンドラーテスト
// 独立DB分離による安全な並列実行の実証
// ========================================

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"money_management/internal/database"
	"money_management/internal/models"
)

// TestLoginHandler_Parallel_Success 並列実行対応ログイン成功テスト
func TestLoginHandler_Parallel_Success(t *testing.T) {
	// 並列テストが有効な場合のみ並列実行
	if database.IsParallelTestEnabled() {
		t.Parallel()
	}

	// 並列テスト用の独立DB接続を取得
	db, cleanup, err := database.SetupOptimizedTestDB("LoginHandler_Parallel_Success")
	assert.NoError(t, err, "並列テスト用データベースのセットアップに失敗しました")
	defer cleanup()

	// テストユーザーを作成（独立DBなので競合なし）
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := models.User{
		Name:         "並列テストユーザー",
		AccountID:    "parallel_test_user",
		PasswordHash: string(hashedPassword),
	}

	result := db.Create(&user)
	assert.NoError(t, result.Error, "テストユーザーの作成に失敗しました")

	// ログインリクエストを作成
	loginRequest := models.LoginRequest{
		AccountID: "parallel_test_user",
		Password:  "password123",
	}

	jsonBytes, _ := json.Marshal(loginRequest)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	// レスポンスレコーダーを作成
	w := httptest.NewRecorder()

	// Ginルーターを設定
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/login", LoginHandlerWithDB(db))

	// リクエストを実行
	router.ServeHTTP(w, req)

	// レスポンスを検証
	assert.Equal(t, http.StatusOK, w.Code, "ログインが成功していません")

	var response models.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "レスポンスのJSONパースに失敗しました")

	assert.NotEmpty(t, response.Token, "JWTトークンが返されていません")
	assert.Equal(t, user.ID, response.User.ID, "ユーザーIDが一致しません")
	assert.Equal(t, "parallel_test_user", response.User.AccountID, "アカウントIDが一致しません")
}

// TestLoginHandler_Parallel_InvalidCredentials 並列実行対応無効な認証情報テスト
func TestLoginHandler_Parallel_InvalidCredentials(t *testing.T) {
	if database.IsParallelTestEnabled() {
		t.Parallel()
	}

	db, cleanup, err := database.SetupOptimizedTestDB("LoginHandler_Parallel_InvalidCredentials")
	assert.NoError(t, err, "並列テスト用データベースのセットアップに失敗しました")
	defer cleanup()

	// 存在しないユーザーでログイン試行
	loginRequest := models.LoginRequest{
		AccountID: "nonexistent_user",
		Password:  "wrongpassword",
	}

	jsonBytes, _ := json.Marshal(loginRequest)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/login", LoginHandlerWithDB(db))

	router.ServeHTTP(w, req)

	// 認証失敗を検証
	assert.Equal(t, http.StatusUnauthorized, w.Code, "認証エラーが正しく返されていません")

	var errorResponse map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err, "エラーレスポンスのJSONパースに失敗しました")
	assert.Contains(t, errorResponse["error"], "認証情報が無効です", "エラーメッセージが正しくありません")
}

// TestRegisterHandler_Parallel_Success 並列実行対応ユーザー登録成功テスト
func TestRegisterHandler_Parallel_Success(t *testing.T) {
	if database.IsParallelTestEnabled() {
		t.Parallel()
	}

	db, cleanup, err := database.SetupOptimizedTestDB("RegisterHandler_Parallel_Success")
	assert.NoError(t, err, "並列テスト用データベースのセットアップに失敗しました")
	defer cleanup()

	// 新規ユーザー登録リクエスト
	registerRequest := models.RegisterRequest{
		Name:      "新規並列ユーザー",
		AccountID: "new_parallel_user",
		Password:  "newpassword123",
	}

	jsonBytes, _ := json.Marshal(registerRequest)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/register", RegisterHandlerWithDB(db))

	router.ServeHTTP(w, req)

	// 登録成功を検証
	assert.Equal(t, http.StatusCreated, w.Code, "ユーザー登録が成功していません")

	var response models.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "レスポンスのJSONパースに失敗しました")

	assert.NotEmpty(t, response.Token, "JWTトークンが返されていません")
	assert.Equal(t, "新規並列ユーザー", response.User.Name, "ユーザー名が一致しません")
	assert.Equal(t, "new_parallel_user", response.User.AccountID, "アカウントIDが一致しません")

	// データベースでユーザーが作成されていることを確認
	var createdUser models.User
	result := db.Where("account_id = ?", "new_parallel_user").First(&createdUser)
	assert.NoError(t, result.Error, "作成されたユーザーが見つかりません")
	assert.Equal(t, "新規並列ユーザー", createdUser.Name, "データベース内のユーザー名が一致しません")
}

// TestBillHandler_Parallel_Create 並列実行対応家計簿作成テスト
func TestBillHandler_Parallel_Create(t *testing.T) {
	if database.IsParallelTestEnabled() {
		t.Parallel()
	}

	db, cleanup, err := database.SetupOptimizedTestDB("BillHandler_Parallel_Create")
	assert.NoError(t, err, "並列テスト用データベースのセットアップに失敗しました")
	defer cleanup()

	// テストユーザーを作成
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	requester := models.User{
		Name:         "並列請求者",
		AccountID:    "parallel_requester",
		PasswordHash: string(hashedPassword),
	}
	payer := models.User{
		Name:         "並列支払者",
		AccountID:    "parallel_payer",
		PasswordHash: string(hashedPassword),
	}

	db.Create(&requester)
	db.Create(&payer)

	// 家計簿作成リクエスト
	createRequest := map[string]interface{}{
		"year":     2024,
		"month":    3,
		"payer_id": payer.ID,
	}

	jsonBytes, _ := json.Marshal(createRequest)
	req, _ := http.NewRequest("POST", "/bills", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	// 認証用のコンテキスト設定（テスト用）
	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", requester.ID)
		c.Next()
	})
	router.POST("/bills", CreateBillHandlerWithDB(db))

	router.ServeHTTP(w, req)

	// 家計簿作成成功を検証
	assert.Equal(t, http.StatusCreated, w.Code, "家計簿作成が成功していません")

	var response models.MonthlyBill
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "レスポンスのJSONパースに失敗しました")

	assert.Equal(t, 2024, response.Year, "年が一致しません")
	assert.Equal(t, 3, response.Month, "月が一致しません")
	assert.Equal(t, requester.ID, response.RequesterID, "請求者IDが一致しません")
	assert.Equal(t, payer.ID, response.PayerID, "支払者IDが一致しません")
	assert.Equal(t, "pending", response.Status, "ステータスが正しくありません")
}

// TestParallelExecution_ConcurrentTests 並列実行の競合状況をテスト
func TestParallelExecution_ConcurrentTests(t *testing.T) {
	if !database.IsParallelTestEnabled() {
		t.Skip("並列テストが無効のためスキップします")
	}

	// 複数の並列テストを同時実行してデータ競合がないことを確認
	// 各テストの内容を直接実行（t.Parallel()の重複を回避）
	t.Run("Concurrent_Login_Success", func(t *testing.T) {
		t.Parallel()

		// 並列テスト用の独立DB接続を取得
		db, cleanup, err := database.SetupOptimizedTestDB("ConcurrentLoginSuccess")
		assert.NoError(t, err)
		defer cleanup()

		// テストユーザーを作成
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		user := models.User{
			Name:         "並列テストユーザー1",
			AccountID:    "parallel_test_user_1",
			PasswordHash: string(hashedPassword),
		}
		db.Create(&user)

		// ログインテスト実行
		loginRequest := models.LoginRequest{
			AccountID: "parallel_test_user_1",
			Password:  "password123",
		}

		jsonBytes, _ := json.Marshal(loginRequest)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/auth/login", LoginHandlerWithDB(db))

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Concurrent_Register_Success", func(t *testing.T) {
		t.Parallel()

		db, cleanup, err := database.SetupOptimizedTestDB("ConcurrentRegisterSuccess")
		assert.NoError(t, err)
		defer cleanup()

		// ユーザー登録テスト実行
		registerRequest := models.RegisterRequest{
			Name:      "新規並列ユーザー2",
			AccountID: "new_parallel_user_2",
			Password:  "newpassword123",
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
	})

	t.Run("Concurrent_Login_Invalid", func(t *testing.T) {
		t.Parallel()

		db, cleanup, err := database.SetupOptimizedTestDB("ConcurrentLoginInvalid")
		assert.NoError(t, err)
		defer cleanup()

		// 無効認証テスト実行
		loginRequest := models.LoginRequest{
			AccountID: "nonexistent_user_3",
			Password:  "wrongpassword",
		}

		jsonBytes, _ := json.Marshal(loginRequest)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.POST("/auth/login", LoginHandlerWithDB(db))

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
