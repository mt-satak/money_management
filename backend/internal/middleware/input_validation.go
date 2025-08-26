package middleware

import (
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// InputValidationConfig 入力値検証の設定
type InputValidationConfig struct {
	MaxBodySize       int64  // リクエストボディの最大サイズ（バイト）
	MaxFieldLength    int    // 各フィールドの最大長
	AllowedChars      string // 許可する文字セット
	BlockSQLInjection bool   // SQLインジェクション検証を有効にするか
	BlockXSS          bool   // XSS攻撃検証を有効にするか
}

// DefaultInputValidationConfig デフォルトの入力値検証設定
var DefaultInputValidationConfig = InputValidationConfig{
	MaxBodySize:       1024 * 1024, // 1MB
	MaxFieldLength:    1000,        // 1000文字
	BlockSQLInjection: true,
	BlockXSS:          true,
}

// 危険なSQLパターン
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union\s+select|drop\s+table|delete\s+from|update\s+set)`),
	regexp.MustCompile(`(?i)(exec\s*\(|sp_executesql|xp_cmdshell)`),
	regexp.MustCompile(`(?i)(script|javascript:|vbscript:|onload=|onerror=)`),
	regexp.MustCompile(`(?i)(or\s+1\s*=\s*1|and\s+1\s*=\s*1)`),
	regexp.MustCompile(`(?i)(\'\s*or\s*\'|\'\s*and\s*\')`),
}

// 危険なXSSパターン
var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
	regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)on\w+\s*=`),
	regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
	regexp.MustCompile(`(?i)<embed[^>]*>`),
}

// InputValidationMiddleware 入力値検証ミドルウェア
func InputValidationMiddleware() gin.HandlerFunc {
	return InputValidationMiddlewareWithConfig(DefaultInputValidationConfig)
}

// InputValidationMiddlewareWithConfig カスタム設定での入力値検証ミドルウェア
func InputValidationMiddlewareWithConfig(config InputValidationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content-Lengthチェック
		if c.Request.ContentLength > config.MaxBodySize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":    "リクエストサイズが大きすぎます",
				"code":     "REQUEST_TOO_LARGE",
				"max_size": config.MaxBodySize,
			})
			c.Abort()
			return
		}

		// URLパラメーター検証
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if err := validateInput(value, config); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":   "無効な入力値です",
						"code":    "INVALID_INPUT",
						"field":   key,
						"details": err.Error(),
					})
					c.Abort()
					return
				}
			}
		}

		// Content-Type別の検証
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/json") {
			// JSON入力の場合は後続の処理で詳細検証
			c.Set("input_validation_required", true)
		}

		c.Next()
	}
}

// validateInput 入力値の検証
func validateInput(input string, config InputValidationConfig) error {
	// 長さチェック
	if len(input) > config.MaxFieldLength {
		return &ValidationError{
			Type:    "LENGTH_EXCEEDED",
			Message: "入力値が長すぎます",
			Details: map[string]interface{}{
				"max_length":    config.MaxFieldLength,
				"actual_length": len(input),
			},
		}
	}

	// UTF-8有効性チェック
	if !utf8.ValidString(input) {
		return &ValidationError{
			Type:    "INVALID_ENCODING",
			Message: "無効な文字エンコーディングです",
		}
	}

	// 制御文字チェック
	if containsControlChars(input) {
		return &ValidationError{
			Type:    "CONTROL_CHARS",
			Message: "制御文字は許可されていません",
		}
	}

	// SQLインジェクション検証
	if config.BlockSQLInjection && containsSQLInjection(input) {
		return &ValidationError{
			Type:    "SQL_INJECTION",
			Message: "SQLインジェクションの可能性があります",
		}
	}

	// XSS検証
	if config.BlockXSS && containsXSS(input) {
		return &ValidationError{
			Type:    "XSS_ATTACK",
			Message: "XSS攻撃の可能性があります",
		}
	}

	return nil
}

// ValidationError 検証エラー
type ValidationError struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// containsControlChars 制御文字が含まれているかチェック
func containsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
}

// containsSQLInjection SQLインジェクションパターンが含まれているかチェック
func containsSQLInjection(s string) bool {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}

// containsXSS XSSパターンが含まれているかチェック
func containsXSS(s string) bool {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}

// SanitizeInput 入力値のサニタイズ
func SanitizeInput(input string) string {
	// HTMLエスケープ
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")

	// 前後の空白除去
	input = strings.TrimSpace(input)

	return input
}

// StrictInputValidationMiddleware 厳格な入力値検証（認証系エンドポイント用）
func StrictInputValidationMiddleware() gin.HandlerFunc {
	config := InputValidationConfig{
		MaxBodySize:       10 * 1024, // 10KB（認証情報は小さくあるべき）
		MaxFieldLength:    100,       // 100文字
		BlockSQLInjection: true,
		BlockXSS:          true,
	}
	return InputValidationMiddlewareWithConfig(config)
}

// GetValidationStats 検証統計の取得
func GetValidationStats() map[string]interface{} {
	// 将来の拡張用：検証統計の収集
	return map[string]interface{}{
		"validation_enabled":       true,
		"max_body_size":            DefaultInputValidationConfig.MaxBodySize,
		"max_field_length":         DefaultInputValidationConfig.MaxFieldLength,
		"sql_injection_protection": DefaultInputValidationConfig.BlockSQLInjection,
		"xss_protection":           DefaultInputValidationConfig.BlockXSS,
	}
}
