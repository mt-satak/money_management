// ========================================
// JWT認証ミドルウェアの自動テスト
// セキュリティクリティカルな機能のテスト
// ========================================

package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// setupTestEnvironment テスト環境を設定する
func setupTestEnvironment() {
	// テスト用のJWTシークレットを設定
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-jwt-secret-for-testing-purposes-32chars")
	}
}

// generateValidToken 有効なJWTトークンを生成するヘルパー関数
func generateValidToken(userID uint) string {
	setupTestEnvironment()

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(GetJWTSecret())
	return tokenString
}

// generateExpiredToken 期限切れのJWTトークンを生成するヘルパー関数
func generateExpiredToken(userID uint) string {
	setupTestEnvironment()

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 1時間前に期限切れ
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(GetJWTSecret())
	return tokenString
}

// setupTestRouter テスト用のGinルーターを設定
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 保護されたエンドポイント
	protected := router.Group("/protected")
	protected.Use(AuthMiddleware())
	protected.GET("/test", func(c *gin.Context) {
		userID := c.GetUint("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "message": "success"})
	})

	return router
}

// TestGetJWTSecret JWTシークレットキーの取得をテスト
func TestGetJWTSecret(t *testing.T) {
	t.Parallel()

	// テスト用環境変数を設定
	originalSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	testSecret := "test-jwt-secret-for-testing-purposes-32chars"
	os.Setenv("JWT_SECRET", testSecret)

	secret := GetJWTSecret()
	assert.NotNil(t, secret, "JWTシークレットがnilです")
	assert.Greater(t, len(secret), 0, "JWTシークレットが空です")
	assert.Equal(t, []byte(testSecret), secret, "期待されるシークレットキーと異なります")
}

// TestAuthMiddleware_Success 有効なJWTトークンで認証が成功することをテスト
func TestAuthMiddleware_Success(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	// 有効なトークンを生成
	token := generateValidToken(123)

	req := httptest.NewRequest("GET", "/protected/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "有効なトークンで認証が失敗しました")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "レスポンスJSONの解析に失敗しました")
	assert.Equal(t, float64(123), response["user_id"], "ユーザーIDが正しく設定されていません")
	assert.Equal(t, "success", response["message"], "成功メッセージが期待値と異なります")
}

// TestAuthMiddleware_MissingHeader Authorizationヘッダーが無い場合のテスト
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	req := httptest.NewRequest("GET", "/protected/test", nil)
	// Authorizationヘッダーを設定しない
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Authorizationヘッダーなしでアクセスが許可されました")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "認証ヘッダーが必要です", response["error"], "期待されるエラーメッセージと異なります")
}

// TestAuthMiddleware_EmptyHeader 空のAuthorizationヘッダーの場合のテスト
func TestAuthMiddleware_EmptyHeader(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	req := httptest.NewRequest("GET", "/protected/test", nil)
	req.Header.Set("Authorization", "")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "空のAuthorizationヘッダーでアクセスが許可されました")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "認証ヘッダーが必要です", response["error"], "期待されるエラーメッセージと異なります")
}

// TestAuthMiddleware_WithoutBearerPrefix Bearer プレフィックスなしのテスト
func TestAuthMiddleware_WithoutBearerPrefix(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	// 有効なトークンだがBearerプレフィックスなし
	token := generateValidToken(123)

	req := httptest.NewRequest("GET", "/protected/test", nil)
	req.Header.Set("Authorization", token) // Bearerプレフィックスなし
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Bearerプレフィックスがなくてもトークンとして処理される
	assert.Equal(t, http.StatusOK, w.Code, "Bearerプレフィックスなしで認証が失敗しました")
}

// TestAuthMiddleware_InvalidToken 無効なトークンのテスト
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	req := httptest.NewRequest("GET", "/protected/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "無効なトークンでアクセスが許可されました")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "無効なトークンです", response["error"], "期待されるエラーメッセージと異なります")
}

// TestAuthMiddleware_ExpiredToken 期限切れトークンのテスト
func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	// 期限切れトークンを生成
	token := generateExpiredToken(123)

	req := httptest.NewRequest("GET", "/protected/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "期限切れトークンでアクセスが許可されました")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "エラーレスポンスJSONの解析に失敗しました")
	assert.Equal(t, "無効なトークンです", response["error"], "期待されるエラーメッセージと異なります")
}

// TestAuthMiddleware_WrongSigningKey 間違った署名キーで作成されたトークンのテスト
func TestAuthMiddleware_WrongSigningKey(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	// 間違ったキーでトークンを生成
	claims := Claims{
		UserID: 123,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	wrongKey := []byte("wrong-secret-key")
	tokenString, _ := token.SignedString(wrongKey)

	req := httptest.NewRequest("GET", "/protected/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "間違った署名キーのトークンでアクセスが許可されました")
}

// TestAuthMiddleware_MalformedToken 不正な形式のトークンのテスト
func TestAuthMiddleware_MalformedToken(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	testCases := []struct {
		name  string
		token string
	}{
		{"空文字", "Bearer "},
		{"不完全なJWT", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"JSON形式でない", "Bearer {invalid-json}"},
		{"Base64でない文字", "Bearer invalid-base64-characters-@#$%"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected/test", nil)
			req.Header.Set("Authorization", tc.token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code, fmt.Sprintf("不正な形式のトークン '%s' でアクセスが許可されました", tc.name))
		})
	}
}

// TestGetJWTSecret_ShortSecret JWTシークレットの最小長チェックをテスト
// 注意: このテストは log.Fatal により実際のプロセスが終了するため無効化
// 本番環境では log.Fatal の代わりに error return を使用することを推奨
func TestGetJWTSecret_ShortSecret(t *testing.T) {
	t.Skip("log.Fatal を使用しているためテストスキップ。本番環境では error return を使用すべき")
}

// TestAuthMiddleware_DifferentUserIDs 異なるユーザーIDでのテスト
func TestAuthMiddleware_DifferentUserIDs(t *testing.T) {
	t.Parallel()

	router := setupTestRouter()

	testUserIDs := []uint{1, 999, 0, 4294967295} // 様々なユーザーID

	for _, userID := range testUserIDs {
		t.Run(fmt.Sprintf("UserID_%d", userID), func(t *testing.T) {
			token := generateValidToken(userID)

			req := httptest.NewRequest("GET", "/protected/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, fmt.Sprintf("ユーザーID %d での認証が失敗しました", userID))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "レスポンスJSONの解析に失敗しました")
			assert.Equal(t, float64(userID), response["user_id"], fmt.Sprintf("ユーザーID %d が正しく設定されていません", userID))
		})
	}
}
