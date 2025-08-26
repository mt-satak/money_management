package models

// LoginRequest ログインリクエスト
// ユーザーのログイン時に送信されるデータ構造
type LoginRequest struct {
	AccountID string `json:"account_id" binding:"required"` // アカウントID（必須）
	Password  string `json:"password" binding:"required"`   // パスワード（必須）
}

// RegisterRequest ユーザー登録リクエスト
// 新規ユーザー登録時に送信されるデータ構造
type RegisterRequest struct {
	Name      string `json:"name" binding:"required"`       // ユーザー名（必須）
	AccountID string `json:"account_id" binding:"required"` // アカウントID（必須）
	Password  string `json:"password" binding:"required"`   // パスワード（必須）
}

// LoginResponse ログイン・登録レスポンス
// ログインまたは登録成功時に返されるデータ構造
type LoginResponse struct {
	Token string `json:"token"` // JWTトークン
	User  User   `json:"user"`  // ユーザー情報
}

// BillResponse 家計簿レスポンス
// 家計簿データ取得時に返されるデータ構造（計算済みの金額情報を含む）
type BillResponse struct {
	MonthlyBill         // 月次家計簿の基本情報
	TotalAmount float64 `json:"total_amount"` // 総金額（請求金額）
}
