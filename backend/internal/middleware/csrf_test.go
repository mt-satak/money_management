package middleware

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCSRFSecret(t *testing.T) {
	// 元の環境変数を保存
	originalSecret := os.Getenv("CSRF_SECRET")
	defer os.Setenv("CSRF_SECRET", originalSecret)

	// テスト用環境変数を設定
	testSecret := "test-csrf-secret-32-characters-long"
	os.Setenv("CSRF_SECRET", testSecret)

	result := GetCSRFSecret()
	assert.Equal(t, testSecret, result)
}

func TestGetCSRFSecret_Default(t *testing.T) {
	// 元の環境変数を保存
	originalSecret := os.Getenv("CSRF_SECRET")
	defer os.Setenv("CSRF_SECRET", originalSecret)

	// 環境変数をクリア
	os.Unsetenv("CSRF_SECRET")

	result := GetCSRFSecret()
	expected := "default-csrf-secret-change-in-production-32chars"
	assert.Equal(t, expected, result)
}

func TestGetSessionSecret(t *testing.T) {
	// 元の環境変数を保存
	originalSecret := os.Getenv("SESSION_SECRET")
	defer os.Setenv("SESSION_SECRET", originalSecret)

	// テスト用環境変数を設定
	testSecret := "test-session-secret-32-characters-long"
	os.Setenv("SESSION_SECRET", testSecret)

	result := GetSessionSecret()
	assert.Equal(t, []byte(testSecret), result)
}

func TestGetSessionSecret_Default(t *testing.T) {
	// 元の環境変数を保存
	originalSecret := os.Getenv("SESSION_SECRET")
	defer os.Setenv("SESSION_SECRET", originalSecret)

	// 環境変数をクリア
	os.Unsetenv("SESSION_SECRET")

	result := GetSessionSecret()
	expected := []byte("default-session-secret-change-in-production-32chars")
	assert.Equal(t, expected, result)
}

func TestSecretLength_Warning(t *testing.T) {
	// 元の環境変数を保存
	originalSecret := os.Getenv("CSRF_SECRET")
	defer os.Setenv("CSRF_SECRET", originalSecret)

	// 短いシークレットをテスト
	shortSecret := "short"
	os.Setenv("CSRF_SECRET", shortSecret)

	result := GetCSRFSecret()
	assert.Equal(t, shortSecret, result)
	// ログで警告が出力されることを確認（実際のテストでは標準出力をキャプチャして確認することも可能）
}
