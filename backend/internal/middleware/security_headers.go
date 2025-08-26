package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware セキュリティヘッダーを追加するミドルウェア
// XSS、CSRF、Clickjackingなどの攻撃を防ぐためのHTTPヘッダーを設定
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Frame-Options: Clickjacking攻撃を防ぐ
		c.Header("X-Frame-Options", "DENY")

		// X-Content-Type-Options: MIMEタイプスニッフィング攻撃を防ぐ
		c.Header("X-Content-Type-Options", "nosniff")

		// X-XSS-Protection: ブラウザのXSSフィルターを有効化
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy: リファラー情報の漏洩を制限
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy: XSS攻撃を防ぐ（APIレスポンス用）
		// API専用なので制限的な設定
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Permissions-Policy: 不要な機能へのアクセスを制限
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")

		// Cache-Control: 機微情報のキャッシュを防ぐ
		if c.Request.URL.Path != "/health" && c.Request.URL.Path != "/api/csrf-token" {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		// Server情報の隠蔽
		c.Header("Server", "")

		c.Next()
	}
}

// APISecurityHeadersMiddleware API専用のセキュリティヘッダー設定
// より厳格なポリシーを適用
func APISecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// API専用の厳格なCSP
		c.Header("Content-Security-Policy", "default-src 'none'")

		// API応答のキャッシュを無効化（機微情報保護）
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		// APIレスポンスのコンテンツタイプを明示
		if c.GetHeader("Content-Type") == "" {
			c.Header("Content-Type", "application/json; charset=utf-8")
		}

		c.Next()
	}
}

// DevelopmentHeadersMiddleware 開発環境用の追加ヘッダー
func DevelopmentHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 開発環境であることを示すヘッダー
		c.Header("X-Environment", "development")

		// デバッグ用ヘッダー（本番では除去）
		c.Header("X-Request-ID", c.GetString("request_id"))

		c.Next()
	}
}
