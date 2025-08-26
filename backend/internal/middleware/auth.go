package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWTトークンのクレーム構造体
// JWTトークンに含まれるユーザー情報を表現
type Claims struct {
	UserID               uint `json:"user_id"` // ユーザーID
	jwt.RegisteredClaims      // JWT標準クレーム（有効期限など）
}

// GetJWTSecret JWTシークレットキーを環境変数またはDocker Secretsから安全に取得する
// 1. Docker Secrets (/run/secrets/jwt_secret) を優先的に確認
// 2. JWT_SECRET環境変数をフォールバック
// 設定されていない場合はアプリケーションを停止する
func GetJWTSecret() []byte {
	var secret string

	// Docker Secretsから読み込みを試行
	secretPath := "/run/secrets/jwt_secret"
	if secretBytes, err := os.ReadFile(secretPath); err == nil {
		secret = string(secretBytes)
		log.Println("📋 JWT Secret loaded from Docker Secrets")
	} else {
		// 環境変数からフォールバック
		secret = os.Getenv("JWT_SECRET")
		if secret == "" {
			log.Fatal("JWT_SECRET environment variable or Docker Secret is required but not set. Please set a strong secret key.")
		}
		log.Println("📋 JWT Secret loaded from environment variable")
	}

	// 改行文字を除去（Docker Secretsファイルに含まれる可能性があるため）
	secret = strings.TrimSpace(secret)

	// セキュリティのため、最小長をチェック
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long for security")
	}

	return []byte(secret)
}

// AuthMiddleware JWT認証ミドルウェア
// リクエストヘッダーのAuthorizationからJWTトークンを検証し、
// 有効な場合はユーザーIDをコンテキストに設定する
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// AuthorizationヘッダーからJWTトークンを取得
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証ヘッダーが必要です"})
			c.Abort()
			return
		}

		// "Bearer "プレフィックスを除去
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		// トークンがブラックリストに含まれているかチェック
		if IsTokenBlacklisted(tokenString) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "このトークンは無効です。再度ログインしてください。",
				"code":  "TOKEN_BLACKLISTED",
			})
			c.Abort()
			return
		}

		// JWTトークンを解析・検証
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return GetJWTSecret(), nil
		})

		// トークンが無効またはエラーが発生した場合
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無効なトークンです"})
			c.Abort()
			return
		}

		// クレームを取得してユーザーIDを設定
		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無効なトークンクレームです"})
			c.Abort()
			return
		}

		// ユーザーIDをコンテキストに設定（後続のハンドラーで使用可能）
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
