package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorType エラータイプの定義
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "VALIDATION_ERROR"
	ErrorTypeAuthentication ErrorType = "AUTHENTICATION_ERROR"
	ErrorTypeAuthorization  ErrorType = "AUTHORIZATION_ERROR"
	ErrorTypeNotFound       ErrorType = "NOT_FOUND"
	ErrorTypeInternal       ErrorType = "INTERNAL_ERROR"
	ErrorTypeRateLimit      ErrorType = "RATE_LIMIT_ERROR"
	ErrorTypeCSRF           ErrorType = "CSRF_ERROR"
)

// SecureError セキュアなエラー応答
type SecureError struct {
	Type      ErrorType `json:"type"`
	Message   string    `json:"message"`
	Code      string    `json:"code"`
	Timestamp string    `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
}

// ErrorHandlingMiddleware セキュアなエラーハンドリングミドルウェア
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// リクエストID生成（トレース用）
		requestID := generateRequestID()
		c.Set("request_id", requestID)

		// パニック回復とエラー処理
		defer func() {
			if err := recover(); err != nil {
				// パニック情報をログに記録（内部診断用）
				logPanicError(err, requestID, c)

				// 外部には汎用エラーメッセージのみ返す
				secureError := SecureError{
					Type:      ErrorTypeInternal,
					Message:   "内部サーバーエラーが発生しました",
					Code:      "INTERNAL_SERVER_ERROR",
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					RequestID: requestID,
				}

				c.JSON(http.StatusInternalServerError, secureError)
				c.Abort()
			}
		}()

		c.Next()

		// エラーレスポンスの後処理
		if len(c.Errors) > 0 {
			handleGinErrors(c, requestID)
		}
	}
}

// generateRequestID リクエストIDを生成
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// logPanicError パニックエラーを安全にログ出力
func logPanicError(err interface{}, requestID string, c *gin.Context) {
	// スタックトレースを取得
	stack := make([]byte, 2048)
	length := runtime.Stack(stack, false)

	log.Printf("🚨 PANIC [%s]: %v\n", requestID, err)
	log.Printf("📍 Request: %s %s\n", c.Request.Method, c.Request.URL.Path)
	log.Printf("🔍 User-Agent: %s\n", c.Request.UserAgent())
	log.Printf("📋 Stack Trace:\n%s\n", stack[:length])

	// 機微情報を除いたリクエスト情報をログ出力
	logSafeRequestInfo(c, requestID)
}

// logSafeRequestInfo 機微情報を除外したリクエスト情報をログ出力
func logSafeRequestInfo(c *gin.Context, requestID string) {
	// 機微情報を含む可能性のあるヘッダーを除外
	safeHeaders := make(map[string]string)
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"x-csrf-token":  true,
	}

	for name, values := range c.Request.Header {
		lowerName := strings.ToLower(name)
		if !sensitiveHeaders[lowerName] && len(values) > 0 {
			safeHeaders[name] = values[0]
		}
	}

	log.Printf("🔍 Safe Headers [%s]: %+v\n", requestID, safeHeaders)
}

// handleGinErrors Ginのエラーを処理
func handleGinErrors(c *gin.Context, requestID string) {
	lastError := c.Errors.Last()
	if lastError == nil {
		return
	}

	// エラーを内部ログに記録
	log.Printf("⚠️ API Error [%s]: %v\n", requestID, lastError.Err)
	log.Printf("📍 Endpoint: %s %s\n", c.Request.Method, c.Request.URL.Path)

	// 既にレスポンスが送信されている場合はスキップ
	if c.Writer.Written() {
		return
	}

	// エラータイプに基づいてセキュアなレスポンスを生成
	secureError := mapErrorToSecureResponse(lastError.Err, requestID)
	statusCode := getHTTPStatusFromError(lastError.Err)

	c.JSON(statusCode, secureError)
}

// mapErrorToSecureResponse エラーをセキュアなレスポンスにマップ
func mapErrorToSecureResponse(err error, requestID string) SecureError {
	errorMessage := err.Error()

	// 既知のエラーパターンに基づいて分類
	switch {
	case strings.Contains(errorMessage, "VALIDATION"):
		return SecureError{
			Type:      ErrorTypeValidation,
			Message:   "入力データが無効です",
			Code:      "VALIDATION_FAILED",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: requestID,
		}
	case strings.Contains(errorMessage, "UNAUTHORIZED") || strings.Contains(errorMessage, "TOKEN"):
		return SecureError{
			Type:      ErrorTypeAuthentication,
			Message:   "認証が必要です",
			Code:      "AUTHENTICATION_REQUIRED",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: requestID,
		}
	case strings.Contains(errorMessage, "FORBIDDEN"):
		return SecureError{
			Type:      ErrorTypeAuthorization,
			Message:   "この操作は許可されていません",
			Code:      "ACCESS_FORBIDDEN",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: requestID,
		}
	case strings.Contains(errorMessage, "NOT_FOUND"):
		return SecureError{
			Type:      ErrorTypeNotFound,
			Message:   "要求されたリソースが見つかりません",
			Code:      "RESOURCE_NOT_FOUND",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: requestID,
		}
	default:
		// 詳細を隠した汎用エラー
		return SecureError{
			Type:      ErrorTypeInternal,
			Message:   "処理中にエラーが発生しました",
			Code:      "PROCESSING_ERROR",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: requestID,
		}
	}
}

// getHTTPStatusFromError エラーからHTTPステータスコードを決定
func getHTTPStatusFromError(err error) int {
	errorMessage := err.Error()

	switch {
	case strings.Contains(errorMessage, "VALIDATION"):
		return http.StatusBadRequest
	case strings.Contains(errorMessage, "UNAUTHORIZED"):
		return http.StatusUnauthorized
	case strings.Contains(errorMessage, "FORBIDDEN"):
		return http.StatusForbidden
	case strings.Contains(errorMessage, "NOT_FOUND"):
		return http.StatusNotFound
	case strings.Contains(errorMessage, "RATE_LIMIT"):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// SecurityAuditMiddleware セキュリティ監査用のログミドルウェア
func SecurityAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		// 監査ログ出力
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		// セキュリティ関連の操作をログに記録
		if isSecuritySensitiveOperation(c.Request.URL.Path, c.Request.Method) {
			log.Printf("🔐 Security Audit: %s %s | Status: %d | Duration: %v | IP: %s | User-Agent: %s\n",
				c.Request.Method,
				c.Request.URL.Path,
				statusCode,
				duration,
				c.ClientIP(),
				c.Request.UserAgent(),
			)
		}
	}
}

// isSecuritySensitiveOperation セキュリティに関連する操作かどうか判定
func isSecuritySensitiveOperation(path, method string) bool {
	sensitiveEndpoints := []string{
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/logout",
		"/api/auth/me",
	}

	for _, endpoint := range sensitiveEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}

	// POST, PUT, DELETE操作は基本的に監査対象
	return method == "POST" || method == "PUT" || method == "DELETE"
}
