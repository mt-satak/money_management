package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MonthlyBill 月次家計簿モデル
// 特定の月の家計簿情報を表現するモデル
type MonthlyBill struct {
	ID          uint       `json:"id" gorm:"primaryKey"`                                                    // 家計簿ID（主キー）
	Year        int        `json:"year"`                                                                    // 対象年
	Month       int        `json:"month"`                                                                   // 対象月
	RequesterID uint       `json:"requester_id"`                                                            // 請求者のユーザーID
	PayerID     uint       `json:"payer_id"`                                                                // 支払者のユーザーID
	Status      string     `json:"status" gorm:"type:enum('pending','requested','paid');default:'pending'"` // 状態（pending: 作成中, requested: 請求済み, paid: 支払済み）
	RequestDate *time.Time `json:"request_date"`                                                            // 請求日時（請求時に設定）
	PaymentDate *time.Time `json:"payment_date"`                                                            // 支払日時（支払時に設定）
	Requester   User       `json:"requester" gorm:"foreignKey:RequesterID"`                                 // 請求者のユーザー情報
	Payer       User       `json:"payer" gorm:"foreignKey:PayerID"`                                         // 支払者のユーザー情報
	Items       []BillItem `json:"items" gorm:"foreignKey:BillID"`                                          // 家計簿項目リスト
	CreatedAt   time.Time  `json:"created_at"`                                                              // 作成日時
	UpdatedAt   time.Time  `json:"updated_at"`                                                              // 更新日時
}

// TableName テーブル名を明示的に指定
func (MonthlyBill) TableName() string {
	return "monthly_bills"
}

// BeforeCreate 作成前のフック（ユニーク制約チェック用）
func (bill *MonthlyBill) BeforeCreate(tx *gorm.DB) error {
	// 同じ請求者・年・月の家計簿が既に存在するかチェック
	var count int64
	tx.Model(&MonthlyBill{}).Where("year = ? AND month = ? AND requester_id = ?",
		bill.Year, bill.Month, bill.RequesterID).Count(&count)

	if count > 0 {
		return fmt.Errorf("Duplicate entry '%d-%d-%d' for key 'unique_month_requester'",
			bill.Year, bill.Month, bill.RequesterID)
	}
	return nil
}

// BillItem 家計簿項目モデル
// 家計簿の個々の支出項目を表現するモデル
type BillItem struct {
	ID        uint      `json:"id" gorm:"primaryKey"`             // 項目ID（主キー）
	BillID    uint      `json:"bill_id"`                          // 所属する家計簿のID
	ItemName  string    `json:"item_name"`                        // 項目名（例: 食費、交通費など）
	Amount    float64   `json:"amount" gorm:"type:decimal(10,2)"` // 金額（小数点以下2桁まで対応）
	CreatedAt time.Time `json:"created_at"`                       // 作成日時
	UpdatedAt time.Time `json:"updated_at"`                       // 更新日時
}
