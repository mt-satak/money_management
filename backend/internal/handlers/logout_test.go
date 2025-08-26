package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"money_management/internal/middleware"
)

func TestLogoutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// テスト用JWTトークンを生成
	claims := &middleware.Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middleware.GetJWTSecret())
	assert.NoError(t, err)

	t.Run("正常なログアウト", func(t *testing.T) {
		router := gin.New()
		router.POST("/logout", LogoutHandler)

		req := httptest.NewRequest("POST", "/logout", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ログアウトしました")

		// トークンがブラックリストに追加されたか確認
		assert.True(t, middleware.IsTokenBlacklisted(tokenString))
	})

	t.Run("Authorizationヘッダーなし", func(t *testing.T) {
		router := gin.New()
		router.POST("/logout", LogoutHandler)

		req := httptest.NewRequest("POST", "/logout", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "MISSING_TOKEN")
	})

	t.Run("無効なトークン", func(t *testing.T) {
		router := gin.New()
		router.POST("/logout", LogoutHandler)

		req := httptest.NewRequest("POST", "/logout", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "INVALID_TOKEN")
	})
}

func TestGetTokenStatusHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// テスト用JWTトークンを生成
	claims := &middleware.Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middleware.GetJWTSecret())
	assert.NoError(t, err)

	t.Run("有効なトークンのステータス", func(t *testing.T) {
		router := gin.New()
		router.GET("/token-status", GetTokenStatusHandler)

		req := httptest.NewRequest("GET", "/token-status", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "token_status")
		assert.Contains(t, w.Body.String(), "valid")
	})

	t.Run("ブラックリスト化されたトークンのステータス", func(t *testing.T) {
		// まずトークンをブラックリストに追加
		middleware.AddTokenToBlacklist(tokenString, time.Now().Add(24*time.Hour), "test")

		router := gin.New()
		router.GET("/token-status", GetTokenStatusHandler)

		req := httptest.NewRequest("GET", "/token-status", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "blacklisted")
	})
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Bearer付きトークン",
			header:   "Bearer abc123",
			expected: "abc123",
		},
		{
			name:     "Bearer付きトークン（長いトークン）",
			header:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:     "Bearer無しトークン",
			header:   "abc123",
			expected: "abc123",
		},
		{
			name:     "空文字列",
			header:   "",
			expected: "",
		},
		{
			name:     "Bearer のみ",
			header:   "Bearer ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTokenFromHeader(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}
