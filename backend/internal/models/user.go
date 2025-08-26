package models

import "time"

// User ユーザーモデル
// 家計簿アプリのユーザー情報を表現するモデル
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`     // ユーザーID（主キー）
	Name         string    `json:"name"`                     // ユーザー名
	AccountID    string    `json:"account_id" gorm:"unique"` // アカウントID（ログイン用、ユニーク制約）
	PasswordHash string    `json:"-"`                        // パスワードハッシュ（JSONには含めない）
	CreatedAt    time.Time `json:"created_at"`               // 作成日時
	UpdatedAt    time.Time `json:"updated_at"`               // 更新日時
}
