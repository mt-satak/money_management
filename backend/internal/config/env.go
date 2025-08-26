// ========================================
// 環境変数取得ユーティリティ
// 循環インポートを避けるための独立パッケージ
// ========================================

package config

import (
	"os"
	"strconv"
)

// GetStringEnv 環境変数から文字列を取得（デフォルト値付き）
func GetStringEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// GetBoolEnv 環境変数からboolを取得（デフォルト値付き）
func GetBoolEnv(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	result, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return result
}

// GetIntEnv 環境変数からintを取得（デフォルト値付き）
func GetIntEnv(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return result
}
