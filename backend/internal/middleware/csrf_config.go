package middleware

import (
	"log"
	"os"
	"strings"
)

// GetCSRFSecret CSRF保護用のシークレットキーを環境変数またはDocker Secretsから安全に取得する
// 1. Docker Secrets (/run/secrets/csrf_secret) を優先的に確認
// 2. CSRF_SECRET環境変数をフォールバック
// 3. 最後に開発用デフォルトキーを使用（警告付き）
func GetCSRFSecret() string {
	var secret string

	// Docker Secretsから読み込みを試行
	secretPath := "/run/secrets/csrf_secret"
	if secretBytes, err := os.ReadFile(secretPath); err == nil {
		secret = string(secretBytes)
		log.Println("🔐 CSRF Secret loaded from Docker Secrets")
	} else {
		// 環境変数からフォールバック
		secret = os.Getenv("CSRF_SECRET")
		if secret == "" {
			// 開発用デフォルト（セキュリティ警告付き）
			log.Println("⚠️ WARNING: Using default CSRF secret. Set CSRF_SECRET environment variable or use Docker Secrets for production")
			secret = "default-csrf-secret-change-in-production-32chars"
		} else {
			log.Println("🔐 CSRF Secret loaded from environment variable")
		}
	}

	// 改行文字を除去（Docker Secretsファイルに含まれる可能性があるため）
	secret = strings.TrimSpace(secret)

	// セキュリティのため、最小長をチェック
	if len(secret) < 32 {
		log.Printf("⚠️ WARNING: CSRF Secret should be at least 32 characters long for security (current: %d chars)", len(secret))
	}

	return secret
}

// GetSessionSecret セッション保護用のシークレットキーを環境変数またはDocker Secretsから安全に取得する
// 1. Docker Secrets (/run/secrets/session_secret) を優先的に確認
// 2. SESSION_SECRET環境変数をフォールバック
// 3. 最後に開発用デフォルトキーを使用（警告付き）
func GetSessionSecret() []byte {
	var secret string

	// Docker Secretsから読み込みを試行
	secretPath := "/run/secrets/session_secret"
	if secretBytes, err := os.ReadFile(secretPath); err == nil {
		secret = string(secretBytes)
		log.Println("🔐 Session Secret loaded from Docker Secrets")
	} else {
		// 環境変数からフォールバック
		secret = os.Getenv("SESSION_SECRET")
		if secret == "" {
			// 開発用デフォルト（セキュリティ警告付き）
			log.Println("⚠️ WARNING: Using default session secret. Set SESSION_SECRET environment variable or use Docker Secrets for production")
			secret = "default-session-secret-change-in-production-32chars"
		} else {
			log.Println("🔐 Session Secret loaded from environment variable")
		}
	}

	// 改行文字を除去（Docker Secretsファイルに含まれる可能性があるため）
	secret = strings.TrimSpace(secret)

	// セキュリティのため、最小長をチェック
	if len(secret) < 32 {
		log.Printf("⚠️ WARNING: Session Secret should be at least 32 characters long for security (current: %d chars)", len(secret))
	}

	return []byte(secret)
}
