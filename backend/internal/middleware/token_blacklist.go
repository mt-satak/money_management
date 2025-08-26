package middleware

import (
	"sync"
	"time"
)

// BlacklistedToken ブラックリストに登録されたトークンの情報
type BlacklistedToken struct {
	Token     string
	ExpiresAt time.Time
	Reason    string
}

// TokenBlacklist JWTトークンのブラックリスト管理
type TokenBlacklist struct {
	tokens map[string]*BlacklistedToken
	mutex  sync.RWMutex
}

// グローバルなブラックリストインスタンス
var globalBlacklist = &TokenBlacklist{
	tokens: make(map[string]*BlacklistedToken),
}

func init() {
	// 定期的にexpiredトークンをクリーンアップ（10分間隔）
	go globalBlacklist.startCleanup()
}

// AddToBlacklist トークンをブラックリストに追加
func (tb *TokenBlacklist) AddToBlacklist(token string, expiresAt time.Time, reason string) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.tokens[token] = &BlacklistedToken{
		Token:     token,
		ExpiresAt: expiresAt,
		Reason:    reason,
	}
}

// IsBlacklisted トークンがブラックリストに含まれているかチェック
func (tb *TokenBlacklist) IsBlacklisted(token string) bool {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()

	blacklistedToken, exists := tb.tokens[token]
	if !exists {
		return false
	}

	// 有効期限が切れている場合はブラックリストから削除
	if time.Now().After(blacklistedToken.ExpiresAt) {
		go tb.removeExpiredToken(token)
		return false
	}

	return true
}

// removeExpiredToken 有効期限切れのトークンを削除（非同期実行用）
func (tb *TokenBlacklist) removeExpiredToken(token string) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	delete(tb.tokens, token)
}

// startCleanup 定期的に有効期限切れのトークンをクリーンアップ
func (tb *TokenBlacklist) startCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		tb.cleanupExpiredTokens()
	}
}

// cleanupExpiredTokens 有効期限切れのトークンをすべてクリーンアップ
func (tb *TokenBlacklist) cleanupExpiredTokens() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	for token, blacklistedToken := range tb.tokens {
		if now.After(blacklistedToken.ExpiresAt) {
			delete(tb.tokens, token)
		}
	}
}

// GetBlacklistSize ブラックリストのサイズを取得（監視用）
func (tb *TokenBlacklist) GetBlacklistSize() int {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	return len(tb.tokens)
}

// GetBlacklistStatus ブラックリストの状況を取得（監視用）
func (tb *TokenBlacklist) GetBlacklistStatus() map[string]interface{} {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()

	validTokens := 0
	expiredTokens := 0
	now := time.Now()

	for _, blacklistedToken := range tb.tokens {
		if now.After(blacklistedToken.ExpiresAt) {
			expiredTokens++
		} else {
			validTokens++
		}
	}

	return map[string]interface{}{
		"total_tokens":   len(tb.tokens),
		"valid_tokens":   validTokens,
		"expired_tokens": expiredTokens,
	}
}

// グローバル関数（外部から簡単にアクセス可能）

// AddTokenToBlacklist トークンをグローバルブラックリストに追加
func AddTokenToBlacklist(token string, expiresAt time.Time, reason string) {
	globalBlacklist.AddToBlacklist(token, expiresAt, reason)
}

// IsTokenBlacklisted トークンがブラックリストに含まれているかチェック
func IsTokenBlacklisted(token string) bool {
	return globalBlacklist.IsBlacklisted(token)
}

// GetTokenBlacklistStatus ブラックリストの状況を取得
func GetTokenBlacklistStatus() map[string]interface{} {
	return globalBlacklist.GetBlacklistStatus()
}
