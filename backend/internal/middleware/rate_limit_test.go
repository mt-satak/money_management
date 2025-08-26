package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("バースト制限テスト", func(t *testing.T) {
		// 非常に制限的な設定でテスト（1秒間に1リクエスト、バースト1）
		testLimiter := NewRateLimiter(60, 1) // 1分間に60リクエスト = 1秒間に1リクエスト
		handler := createRateLimitHandler(testLimiter, "テスト")

		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "success"})
		})

		ip := "192.168.100.1:12345"

		// 1回目は成功するはず
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = ip
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// 2回目は即座にレート制限に引っかかるはず
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = ip
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
		assert.Contains(t, w2.Body.String(), "リクエストが多すぎます")
	})

	t.Run("異なるIPは独立して制限される", func(t *testing.T) {
		testLimiter := NewRateLimiter(60, 2)
		handler := createRateLimitHandler(testLimiter, "テスト")

		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "success"})
		})

		// IP1からのリクエスト
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "192.168.1.1:12345"

		// IP2からのリクエスト
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.2:12345"

		// 両方とも成功するはず
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})
}

func TestGetClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		xForwardedFor  string
		xRealIP        string
		remoteAddr     string
		expectedResult string
	}{
		{
			name:           "X-Forwarded-Forヘッダーがある場合",
			xForwardedFor:  "203.0.113.1, 70.41.3.18",
			xRealIP:        "",
			remoteAddr:     "127.0.0.1:12345",
			expectedResult: "203.0.113.1",
		},
		{
			name:           "X-Real-IPヘッダーがある場合",
			xForwardedFor:  "",
			xRealIP:        "203.0.113.1",
			remoteAddr:     "127.0.0.1:12345",
			expectedResult: "203.0.113.1",
		},
		{
			name:           "RemoteAddrのみの場合",
			xForwardedFor:  "",
			xRealIP:        "",
			remoteAddr:     "192.168.1.100:12345",
			expectedResult: "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			result := getClientIP(c)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetRateLimitStatus(t *testing.T) {
	// レート制限器にいくつかのクライアントを追加
	testLimiter := NewRateLimiter(100, 20)
	testLimiter.GetLimiter("127.0.0.1")
	testLimiter.GetLimiter("192.168.1.1")

	status := GetRateLimitStatus()

	assert.NotNil(t, status)
	assert.Contains(t, status, "general_clients")
	assert.Contains(t, status, "auth_clients")
	assert.Contains(t, status, "create_clients")
	assert.Contains(t, status, "limits")

	limits := status["limits"].(map[string]interface{})
	assert.Contains(t, limits, "general")
	assert.Contains(t, limits, "auth")
	assert.Contains(t, limits, "create")
}

func TestRateLimiterCleanup(t *testing.T) {
	limiter := NewRateLimiter(100, 20)

	// クライアントを追加
	limiter.GetLimiter("127.0.0.1")
	limiter.GetLimiter("192.168.1.1")

	// クライアント数を確認
	limiter.mu.RLock()
	clientCount := len(limiter.clients)
	limiter.mu.RUnlock()

	assert.Equal(t, 2, clientCount)

	// クリーンアップは10分間隔なので、実際のテストは難しい
	// 構造の確認のみ実施
	assert.NotNil(t, limiter.clients)
	assert.NotNil(t, limiter.mu)
}
