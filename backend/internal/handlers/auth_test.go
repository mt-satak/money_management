// ========================================
// 認証ハンドラーの自動テスト
// 本番環境と同じMySQL 8.0を使用してテスト実行
// ========================================

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"money_management/internal/models"
)

// TestLoginHandler_Success 正しい認証情報でログインが成功することを検証（HTTP200とJWTトークンの返却を期待）
func TestLoginHandler_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("データベース接続が必要なためスキップ（-shortフラグ使用時）")
	}

	// ハンドラーテストは並列化を無効にして安定性を重視
	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	// パスワードをハッシュ化してテストユーザーを作成
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := models.User{
		Name:         "テストユーザー",
		AccountID:    "testuser",
		PasswordHash: string(hashedPassword),
	}
	err = db.Create(&user).Error
	assert.NoError(t, err, "テストユーザーの作成に失敗しました")

	router := setupRouter()
	router.POST("/login", LoginHandlerWithDB(db))

	requestBody := map[string]string{
		"account_id": "testuser",
		"password":   "password123",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "テストユーザー", response.User.Name)
	assert.Equal(t, "testuser", response.User.AccountID)
}

// TestLoginHandler_InvalidAccountID 存在しないアカウントIDでログインを試行した際にエラーが返されることを検証（HTTP401と認証エラーメッセージの返却を期待）
func TestLoginHandler_InvalidAccountID(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.POST("/login", LoginHandlerWithDB(db))

	requestBody := map[string]string{
		"account_id": "nonexistent",
		"password":   "password123",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "認証情報が無効です", response["error"])
}

// TestLoginHandler_WrongPassword 正しいアカウントIDだが間違ったパスワードでログインを試行した際にエラーが返されることを検証（HTTP401と認証エラーメッセージの返却を期待）
func TestLoginHandler_WrongPassword(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	// パスワードをハッシュ化してテストユーザーを作成
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := models.User{
		Name:         "テストユーザー",
		AccountID:    "testuser",
		PasswordHash: string(hashedPassword),
	}
	err = db.Create(&user).Error
	assert.NoError(t, err, "テストユーザーの作成に失敗しました")

	router := setupRouter()
	router.POST("/login", LoginHandlerWithDB(db))

	requestBody := map[string]string{
		"account_id": "testuser",
		"password":   "wrongpassword",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "認証情報が無効です", response["error"])
}

// TestLoginHandler_InvalidJSON 不正なJSONリクエストでログインを試行した際にバリデーションエラーが返されることを検証（HTTP400とバリデーションエラーメッセージの返却を期待）
func TestLoginHandler_InvalidJSON(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.POST("/login", LoginHandlerWithDB(db))

	// 不正なJSON（フィールド不足）
	requestBody := map[string]string{
		"account_id": "testuser",
		// passwordが不足
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRegisterHandler_Success 有効な情報で新規ユーザー登録が成功することを検証（HTTP201とJWTトークンの返却を期待）
func TestRegisterHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.POST("/register", RegisterHandlerWithDB(db))

	requestBody := map[string]string{
		"name":       "新規ユーザー",
		"account_id": "newuser",
		"password":   "password123",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "新規ユーザー", response.User.Name)
	assert.Equal(t, "newuser", response.User.AccountID)

	// データベースにユーザーが作成されたことを確認
	var user models.User
	err = db.Where("account_id = ?", "newuser").First(&user).Error
	assert.NoError(t, err)
	assert.Equal(t, "新規ユーザー", user.Name)
}

// TestRegisterHandler_ShortPassword 6文字未満のパスワードで登録を試行した際にバリデーションエラーが返されることを検証（HTTP400とパスワード長エラーメッセージの返却を期待）
func TestRegisterHandler_ShortPassword(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.POST("/register", RegisterHandlerWithDB(db))

	requestBody := map[string]string{
		"name":       "新規ユーザー",
		"account_id": "newuser",
		"password":   "123", // 短すぎるパスワード
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "パスワードは6文字以上で入力してください", response["error"])
}

// TestRegisterHandler_ShortAccountID 3文字未満のアカウントIDで登録を試行した際にバリデーションエラーが返されることを検証（HTTP400とアカウントID長エラーメッセージの返却を期待）
func TestRegisterHandler_ShortAccountID(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.POST("/register", RegisterHandlerWithDB(db))

	requestBody := map[string]string{
		"name":       "新規ユーザー",
		"account_id": "ab", // 短すぎるアカウントID
		"password":   "password123",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "アカウントIDは3文字以上20文字以下で入力してください", response["error"])
}

// TestRegisterHandler_LongAccountID 20文字超過のアカウントIDで登録を試行した際にバリデーションエラーが返されることを検証（HTTP400とアカウントID長エラーメッセージの返却を期待）
func TestRegisterHandler_LongAccountID(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.POST("/register", RegisterHandlerWithDB(db))

	requestBody := map[string]string{
		"name":       "新規ユーザー",
		"account_id": strings.Repeat("a", 21), // 長すぎるアカウントID
		"password":   "password123",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "アカウントIDは3文字以上20文字以下で入力してください", response["error"])
}

// TestRegisterHandler_DuplicateAccountID 既存のアカウントIDで登録を試行した際に重複エラーが返されることを検証（HTTP409と重複エラーメッセージの返却を期待）
func TestRegisterHandler_DuplicateAccountID(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	// 既存ユーザーを作成
	existingUser := models.User{
		Name:         "既存ユーザー",
		AccountID:    "existing",
		PasswordHash: "hashedpassword",
	}
	err = db.Create(&existingUser).Error
	assert.NoError(t, err)

	router := setupRouter()
	router.POST("/register", RegisterHandlerWithDB(db))

	requestBody := map[string]string{
		"name":       "新規ユーザー",
		"account_id": "existing", // 重複するアカウントID
		"password":   "password123",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "このアカウントIDは既に使用されています", response["error"])
}

// TestRegisterHandler_InvalidJSON 不正なJSONリクエストで登録を試行した際にバリデーションエラーが返されることを検証（HTTP400とバリデーションエラーメッセージの返却を期待）
func TestRegisterHandler_InvalidJSON(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.POST("/register", RegisterHandlerWithDB(db))

	// 不正なJSON（必須フィールド不足）
	requestBody := map[string]string{
		"name": "新規ユーザー",
		// account_idとpasswordが不足
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetMeHandler_Success 有効なユーザーIDで現在のユーザー情報が取得できることを検証（HTTP200とユーザー情報の返却を期待）
func TestGetMeHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	// テストユーザーを作成
	user := models.User{
		Name:         "テストユーザー",
		AccountID:    "testuser",
		PasswordHash: "hashedpassword",
	}
	err = db.Create(&user).Error
	assert.NoError(t, err, "テストユーザーの作成に失敗しました")

	router := setupRouter()
	router.GET("/me", setUserID(user.ID), GetMeHandlerWithDB(db))

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.User
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "テストユーザー", response.Name)
	assert.Equal(t, "testuser", response.AccountID)
	assert.Equal(t, user.ID, response.ID)
}

// TestGetMeHandler_UserNotFound 存在しないユーザーIDで情報取得を試行した際にエラーが返されることを検証（HTTP404とエラーメッセージの返却を期待）
func TestGetMeHandler_UserNotFound(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.GET("/me", setUserID(999), GetMeHandlerWithDB(db)) // 存在しないユーザーID

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ユーザーが見つかりません", response["error"])
}

// TestGetUsersHandler_Success 全ユーザー一覧が正常に取得できることを検証（HTTP200とユーザー一覧の返却を期待）
func TestGetUsersHandler_Success(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	// テストユーザーを複数作成
	users := []models.User{
		{Name: "ユーザー1", AccountID: "user1", PasswordHash: "hash1"},
		{Name: "ユーザー2", AccountID: "user2", PasswordHash: "hash2"},
		{Name: "ユーザー3", AccountID: "user3", PasswordHash: "hash3"},
	}

	for _, user := range users {
		err = db.Create(&user).Error
		assert.NoError(t, err)
	}

	router := setupRouter()
	router.GET("/users", GetUsersHandlerWithDB(db))

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	usersList := response["users"].([]interface{})
	assert.Equal(t, 3, len(usersList))

	// パスワード情報が含まれていないことを確認
	firstUser := usersList[0].(map[string]interface{})
	assert.NotNil(t, firstUser["name"])
	assert.NotNil(t, firstUser["account_id"])
	assert.Nil(t, firstUser["password_hash"]) // パスワードは除外されているべき
}

// TestGetUsersHandler_EmptyResult ユーザーが存在しない場合に空の配列が返されることを検証（HTTP200と空のユーザー一覧の返却を期待）
func TestGetUsersHandler_EmptyResult(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	router := setupRouter()
	router.GET("/users", GetUsersHandlerWithDB(db))

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	usersList := response["users"].([]interface{})
	assert.Equal(t, 0, len(usersList))
}

// TestRegisterHandler_DatabaseError データベースエラー時のハンドリングをテスト
func TestRegisterHandler_DatabaseError(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	// データベース接続を闉じてエラーを発生させる
	sqlDB, _ := db.DB()
	sqlDB.Close()

	router := setupRouter()
	router.POST("/register", RegisterHandlerWithDB(db))

	requestBody := map[string]string{
		"name":       "新規ユーザー",
		"account_id": "newuser",
		"password":   "password123",
	}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "データベースエラー時に適切なステータスが返されませんでした")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Contains(t, response["error"], "作成", "期待されるエラーメッセージが含まれていません")
}

// TestGetUsersHandler_DatabaseError データベースエラー時のGetUsersハンドラーをテスト
func TestGetUsersHandler_DatabaseError(t *testing.T) {
	// ハンドラーテストは並列化を無効にして安定性を重視

	db, err := setupTestDB()
	if err != nil {
		t.Skipf("データベース接続失敗のためテストをスキップ: %v", err)
		return
	}
	defer cleanupTestResources(db)

	// データベース接続を闉じてエラーを発生させる
	sqlDB, _ := db.DB()
	sqlDB.Close()

	router := setupRouter()
	router.GET("/users", GetUsersHandlerWithDB(db))

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "データベースエラー時に適切なステータスが返されませんでした")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "ユーザー一覧の取得に失敗しました", response["error"], "期待されるエラーメッセージと異なります")
}
