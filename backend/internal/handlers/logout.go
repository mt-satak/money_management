package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"money_management/internal/middleware"
)

// LogoutHandler ログアウトハンドラー
// JWTトークンをブラックリストに追加し、セッションを無効化する
func LogoutHandler(c *gin.Context) {
	// Authorizationヘッダーからトークンを取得
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ログアウトにはAuthorizationヘッダーが必要です",
			"code":  "MISSING_TOKEN",
		})
		return
	}

	// "Bearer "プレフィックスを除去
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// JWTトークンを解析してExpiration時刻を取得
	token, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		secret, err := middleware.GetJWTSecret()
		if err != nil {
			return nil, err
		}
		return secret, nil
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "無効なトークンです",
			"code":  "INVALID_TOKEN",
		})
		return
	}

	// クレームからExpiration時刻を取得
	claims, ok := token.Claims.(*middleware.Claims)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "トークンクレームの解析に失敗しました",
			"code":  "CLAIMS_PARSE_ERROR",
		})
		return
	}

	// トークンをブラックリストに追加
	expiresAt := claims.ExpiresAt.Time
	reason := "user_logout"
	middleware.AddTokenToBlacklist(tokenString, expiresAt, reason)

	// セキュリティ監査用ヘッダーを設定
	if _, exists := c.Get("user_id"); exists {
		c.Header("X-User-Logout", "success")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ログアウトしました",
		"code":    "LOGOUT_SUCCESS",
	})
}

// LogoutAllHandler 全デバイスからのログアウトハンドラー
// 特定ユーザーの全トークンを無効化（将来の拡張用）
func LogoutAllHandler(c *gin.Context) {
	// 現在は単一トークンのみ対応
	// 将来的には、ユーザーIDに基づいて全トークンをブラックリスト化
	LogoutHandler(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "全デバイスからログアウトしました",
		"code":    "LOGOUT_ALL_SUCCESS",
		"note":    "現在は単一トークンのみ対応しています",
	})
}

// GetTokenStatusHandler 現在のトークンステータス確認ハンドラー
func GetTokenStatusHandler(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Authorizationヘッダーが必要です",
			"code":  "MISSING_TOKEN",
		})
		return
	}

	// "Bearer "プレフィックスを除去
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// トークンがブラックリストに含まれているかチェック
	isBlacklisted := middleware.IsTokenBlacklisted(tokenString)

	// JWTトークンを解析
	token, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return middleware.GetJWTSecret(), nil
	})

	var tokenInfo map[string]interface{}
	if err != nil {
		tokenInfo = map[string]interface{}{
			"valid":       false,
			"error":       "トークン解析エラー",
			"blacklisted": isBlacklisted,
		}
	} else if claims, ok := token.Claims.(*middleware.Claims); ok {
		tokenInfo = map[string]interface{}{
			"valid":       token.Valid && !isBlacklisted,
			"blacklisted": isBlacklisted,
			"user_id":     claims.UserID,
			"expires_at":  claims.ExpiresAt.Time,
			"issued_at":   claims.IssuedAt.Time,
		}
	} else {
		tokenInfo = map[string]interface{}{
			"valid":       false,
			"error":       "クレーム解析エラー",
			"blacklisted": isBlacklisted,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"token_status": tokenInfo,
	})
}

// ExtractTokenFromHeader Authorizationヘッダーからトークンを抽出する共通関数
func ExtractTokenFromHeader(authorization string) string {
	if authorization == "" {
		return ""
	}

	// "Bearer "プレフィックスを除去
	if strings.HasPrefix(authorization, "Bearer ") {
		token := authorization[7:]
		// Bearer だけの場合は空文字を返す
		if token == "" {
			return ""
		}
		return token
	}

	return authorization
}
