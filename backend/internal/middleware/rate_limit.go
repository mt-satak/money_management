package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter クライアントIPごとのレート制限管理
type RateLimiter struct {
	clients map[string]*rate.Limiter
	mu      sync.RWMutex
	limit   rate.Limit
	burst   int
}

// NewRateLimiter 新しいレート制限器を作成
func NewRateLimiter(requestsPerMinute float64, burst int) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*rate.Limiter),
		limit:   rate.Limit(requestsPerMinute / 60), // 秒間レートに変換
		burst:   burst,
	}
}

// GetLimiter 指定されたIPのレート制限器を取得（存在しない場合は作成）
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.clients[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.limit, rl.burst)
		rl.clients[ip] = limiter
	}

	return limiter
}

// CleanupClients 非アクティブなクライアントを定期的にクリーンアップ
func (rl *RateLimiter) CleanupClients() {
	ticker := time.NewTicker(time.Minute * 10) // 10分ごとにクリーンアップ
	go func() {
		for {
			select {
			case <-ticker.C:
				rl.mu.Lock()
				// 簡単なクリーンアップ: 全クライアントを削除（実装簡素化）
				rl.clients = make(map[string]*rate.Limiter)
				rl.mu.Unlock()
			}
		}
	}()
}

// グローバルなレート制限器インスタンス
var (
	// 一般的なAPIエンドポイント用: 1分間に100リクエスト、バースト20
	generalLimiter = NewRateLimiter(100, 20)
	// 認証エンドポイント用: 1分間に10リクエスト、バースト3（ブルートフォース対策）
	authLimiter = NewRateLimiter(10, 3)
	// 作成系エンドポイント用: 1分間に30リクエスト、バースト10
	createLimiter = NewRateLimiter(30, 10)
)

func init() {
	// 定期クリーンアップを開始
	generalLimiter.CleanupClients()
	authLimiter.CleanupClients()
	createLimiter.CleanupClients()
}

// RateLimitMiddleware 一般的なAPIエンドポイント用のレート制限ミドルウェア
func RateLimitMiddleware() gin.HandlerFunc {
	return createRateLimitHandler(generalLimiter, "一般API")
}

// AuthRateLimitMiddleware 認証エンドポイント用のレート制限ミドルウェア
func AuthRateLimitMiddleware() gin.HandlerFunc {
	return createRateLimitHandler(authLimiter, "認証")
}

// CreateRateLimitMiddleware 作成系エンドポイント用のレート制限ミドルウェア
func CreateRateLimitMiddleware() gin.HandlerFunc {
	return createRateLimitHandler(createLimiter, "作成")
}

// createRateLimitHandler レート制限ハンドラーを生成する共通関数
func createRateLimitHandler(limiter *RateLimiter, category string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// クライアントIPを取得
		ip := getClientIP(c)

		// IPごとのレート制限器を取得
		rateLimiter := limiter.GetLimiter(ip)

		// リクエストが制限に引っかかるかチェック
		if !rateLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "リクエストが多すぎます。しばらくしてから再試行してください。",
				"code":        "RATE_LIMIT_EXCEEDED",
				"category":    category,
				"ip":          ip,
				"retry_after": "60秒後",
			})
			c.Abort()
			return
		}

		// リクエスト処理を続行
		c.Next()
	}
}

// getClientIP クライアントの実際のIPアドレスを取得
func getClientIP(c *gin.Context) string {
	// X-Forwarded-For ヘッダーをチェック（プロキシ経由の場合）
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// 最初のIPを取得（カンマ区切りの場合）
		if len(xff) > 0 {
			for i, char := range xff {
				if char == ',' || char == ' ' {
					return xff[:i]
				}
			}
			return xff
		}
	}

	// X-Real-IP ヘッダーをチェック
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}

	// RemoteAddr から IP を抽出
	return c.ClientIP()
}

// GetRateLimitStatus レート制限の現在状況を取得（監視用）
func GetRateLimitStatus() map[string]interface{} {
	generalLimiter.mu.RLock()
	authLimiter.mu.RLock()
	createLimiter.mu.RLock()

	defer func() {
		generalLimiter.mu.RUnlock()
		authLimiter.mu.RUnlock()
		createLimiter.mu.RUnlock()
	}()

	return map[string]interface{}{
		"general_clients": len(generalLimiter.clients),
		"auth_clients":    len(authLimiter.clients),
		"create_clients":  len(createLimiter.clients),
		"limits": map[string]interface{}{
			"general": map[string]interface{}{
				"requests_per_minute": float64(generalLimiter.limit) * 60,
				"burst":               generalLimiter.burst,
			},
			"auth": map[string]interface{}{
				"requests_per_minute": float64(authLimiter.limit) * 60,
				"burst":               authLimiter.burst,
			},
			"create": map[string]interface{}{
				"requests_per_minute": float64(createLimiter.limit) * 60,
				"burst":               createLimiter.burst,
			},
		},
	}
}
