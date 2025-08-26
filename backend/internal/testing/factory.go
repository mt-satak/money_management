// ========================================
// テストデータファクトリモジュール
// Builder PatternとFactory Patternを組み合わせた効率的なテストデータ生成システム
// ========================================

package testing

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"money_management/internal/models"
)

// TestDataFactory テストデータ生成の中央管理クラス
type TestDataFactory struct {
	db *gorm.DB
}

// NewTestDataFactory テストデータファクトリの初期化
func NewTestDataFactory(db *gorm.DB) *TestDataFactory {
	return &TestDataFactory{db: db}
}

// ========================================
// UserBuilder - ユーザーデータのBuilder Pattern実装
// ========================================

// UserBuilder ユーザー作成のためのBuilder
type UserBuilder struct {
	factory *TestDataFactory
	user    models.User
	persist bool // データベースに保存するか
}

// NewUser 新しいUserBuilderを開始
func (f *TestDataFactory) NewUser() *UserBuilder {
	timestamp := time.Now().UnixNano()
	randomNum := timestamp % 10000

	return &UserBuilder{
		factory: f,
		user: models.User{
			Name:         fmt.Sprintf("テストユーザー_%d", randomNum),
			AccountID:    fmt.Sprintf("test_user_%d", randomNum),
			PasswordHash: fmt.Sprintf("hashedpassword_%d", randomNum),
		},
		persist: true, // デフォルトで保存
	}
}

// WithName ユーザー名を設定
func (b *UserBuilder) WithName(name string) *UserBuilder {
	b.user.Name = name
	return b
}

// WithAccountID アカウントIDを設定
func (b *UserBuilder) WithAccountID(accountID string) *UserBuilder {
	b.user.AccountID = accountID
	return b
}

// WithPassword パスワードハッシュを設定
func (b *UserBuilder) WithPassword(passwordHash string) *UserBuilder {
	b.user.PasswordHash = passwordHash
	return b
}

// WithID 特定のIDを設定（主にテスト用）
func (b *UserBuilder) WithID(id uint) *UserBuilder {
	b.user.ID = id
	return b
}

// AsTransient データベースに保存せず、メモリ上のみで生成
func (b *UserBuilder) AsTransient() *UserBuilder {
	b.persist = false
	return b
}

// Build ユーザーを構築
func (b *UserBuilder) Build() (models.User, error) {
	if !b.persist {
		return b.user, nil
	}

	if err := b.factory.db.Create(&b.user).Error; err != nil {
		return models.User{}, fmt.Errorf("ユーザーの作成に失敗しました: %w", err)
	}

	return b.user, nil
}

// MustBuild ユーザーを構築（エラー時panic）
func (b *UserBuilder) MustBuild() models.User {
	user, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("ユーザー作成失敗: %v", err))
	}
	return user
}

// ========================================
// BillBuilder - 家計簿データのBuilder Pattern実装
// ========================================

// BillBuilder 家計簿作成のためのBuilder
type BillBuilder struct {
	factory *TestDataFactory
	bill    models.MonthlyBill
	items   []models.BillItem
	persist bool
}

// NewBill 新しいBillBuilderを開始
func (f *TestDataFactory) NewBill() *BillBuilder {
	now := time.Now()
	return &BillBuilder{
		factory: f,
		bill: models.MonthlyBill{
			Year:   now.Year(),
			Month:  int(now.Month()),
			Status: "pending",
		},
		items:   []models.BillItem{},
		persist: true,
	}
}

// WithRequester 請求者を設定
func (b *BillBuilder) WithRequester(requesterID uint) *BillBuilder {
	b.bill.RequesterID = requesterID
	return b
}

// WithPayer 支払者を設定
func (b *BillBuilder) WithPayer(payerID uint) *BillBuilder {
	b.bill.PayerID = payerID
	return b
}

// WithYearMonth 年月を設定
func (b *BillBuilder) WithYearMonth(year, month int) *BillBuilder {
	b.bill.Year = year
	b.bill.Month = month
	return b
}

// WithStatus ステータスを設定
func (b *BillBuilder) WithStatus(status string) *BillBuilder {
	b.bill.Status = status
	return b
}

// WithRequestDate 請求日を設定
func (b *BillBuilder) WithRequestDate(date time.Time) *BillBuilder {
	b.bill.RequestDate = &date
	return b
}

// WithPaymentDate 支払日を設定
func (b *BillBuilder) WithPaymentDate(date time.Time) *BillBuilder {
	b.bill.PaymentDate = &date
	return b
}

// AddItem 家計簿項目を追加
func (b *BillBuilder) AddItem(itemName string, amount float64) *BillBuilder {
	item := models.BillItem{
		ItemName: itemName,
		Amount:   amount,
	}
	b.items = append(b.items, item)
	return b
}

// AddItems 複数の家計簿項目を追加
func (b *BillBuilder) AddItems(items ...BillItemSpec) *BillBuilder {
	for _, spec := range items {
		b.AddItem(spec.Name, spec.Amount)
	}
	return b
}

// AsTransient データベースに保存せず、メモリ上のみで生成
func (b *BillBuilder) AsTransient() *BillBuilder {
	b.persist = false
	return b
}

// Build 家計簿を構築
func (b *BillBuilder) Build() (models.MonthlyBill, []models.BillItem, error) {
	if !b.persist {
		// メモリ上のみで生成する場合、IDを仮設定
		b.bill.ID = 1
		for i := range b.items {
			b.items[i].BillID = b.bill.ID
			b.items[i].ID = uint(i + 1)
		}
		return b.bill, b.items, nil
	}

	// データベースに保存
	err := b.factory.db.Transaction(func(tx *gorm.DB) error {
		// 家計簿本体を作成
		if err := tx.Create(&b.bill).Error; err != nil {
			return fmt.Errorf("家計簿の作成に失敗: %w", err)
		}

		// 家計簿項目を作成
		for i := range b.items {
			b.items[i].BillID = b.bill.ID
			if err := tx.Create(&b.items[i]).Error; err != nil {
				return fmt.Errorf("家計簿項目%dの作成に失敗: %w", i+1, err)
			}
		}

		return nil
	})

	if err != nil {
		return models.MonthlyBill{}, nil, err
	}

	return b.bill, b.items, nil
}

// MustBuild 家計簿を構築（エラー時panic）
func (b *BillBuilder) MustBuild() (models.MonthlyBill, []models.BillItem) {
	bill, items, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("家計簿作成失敗: %v", err))
	}
	return bill, items
}

// ========================================
// 便利な型定義とヘルパー
// ========================================

// BillItemSpec 家計簿項目の仕様
type BillItemSpec struct {
	Name   string
	Amount float64
}

// Item 家計簿項目の簡易作成ヘルパー
func Item(name string, amount float64) BillItemSpec {
	return BillItemSpec{Name: name, Amount: amount}
}

// ========================================
// よく使用されるテストデータパターン
// ========================================

// CreateStandardTestScenario 標準的なテストシナリオを作成
// ユーザー3名、家計簿1件、項目2件の基本セット
// 設定により軽量データまたは完全データを作成
func (f *TestDataFactory) CreateStandardTestScenario() (*StandardTestData, error) {
	config := GetGlobalConfig()

	// 高速モードまたはインメモリDBの場合は軽量データを作成
	if config.FastTestMode || config.UseInMemoryDB {
		return f.CreateLightweightTestScenario()
	}

	// 通常の完全データ作成
	return f.CreateFullTestScenario()
}

// CreateLightweightTestScenario 軽量テストシナリオ作成（高速・最小限）
func (f *TestDataFactory) CreateLightweightTestScenario() (*StandardTestData, error) {
	data := &StandardTestData{}

	// 最小限のユーザーデータ（短い名前・ID）
	timestamp := time.Now().UnixNano()
	suffix := fmt.Sprintf("_%d", timestamp%1000) // より短いサフィックス

	// Step 1: ユーザー作成
	var err error
	data.User1, err = f.NewUser().WithName("U1").WithAccountID("u1" + suffix).Build()
	if err != nil {
		return nil, fmt.Errorf("軽量テスト User1作成失敗: %w", err)
	}

	data.User2, err = f.NewUser().WithName("U2").WithAccountID("u2" + suffix).Build()
	if err != nil {
		return nil, fmt.Errorf("軽量テスト User2作成失敗: %w", err)
	}

	data.User3, err = f.NewUser().WithName("U3").WithAccountID("u3" + suffix).Build()
	if err != nil {
		return nil, fmt.Errorf("軽量テスト User3作成失敗: %w", err)
	}

	// Step 2: IDの確認
	if data.User1.ID == 0 || data.User2.ID == 0 {
		return nil, fmt.Errorf("軽量テスト: ユーザーIDが正しく設定されていません: User1.ID=%d, User2.ID=%d", data.User1.ID, data.User2.ID)
	}

	// Step 3: シンプルな家計簿（1つのアイテムのみ）
	data.Bill, data.Items, err = f.NewBill().
		WithRequester(data.User1.ID).
		WithPayer(data.User2.ID).
		AddItems(
			Item("テスト", 1000.0), // 最小限のアイテム
		).
		Build()

	if err != nil {
		return nil, fmt.Errorf("軽量テスト 家計簿作成失敗: %w", err)
	}

	return data, nil
}

// CreateFullTestScenario 完全なテストシナリオ作成（通常モード）
func (f *TestDataFactory) CreateFullTestScenario() (*StandardTestData, error) {
	data := &StandardTestData{}

	// 一意性を保証するためのタイムスタンプサフィックス
	timestamp := time.Now().UnixNano()
	suffix := fmt.Sprintf("_%d", timestamp%10000)

	// Step 1: ユーザー作成（トランザクションを使わずに個別に作成）
	var err error
	data.User1, err = f.NewUser().WithName("山田太郎").WithAccountID("yamada" + suffix).Build()
	if err != nil {
		return nil, fmt.Errorf("User1作成失敗: %w", err)
	}

	data.User2, err = f.NewUser().WithName("佐藤花子").WithAccountID("sato" + suffix).Build()
	if err != nil {
		return nil, fmt.Errorf("User2作成失敗: %w", err)
	}

	data.User3, err = f.NewUser().WithName("田中三郎").WithAccountID("tanaka" + suffix).Build()
	if err != nil {
		return nil, fmt.Errorf("User3作成失敗: %w", err)
	}

	// Step 2: IDの確認
	if data.User1.ID == 0 || data.User2.ID == 0 {
		return nil, fmt.Errorf("ユーザーIDが正しく設定されていません: User1.ID=%d, User2.ID=%d", data.User1.ID, data.User2.ID)
	}

	// Step 3: 家計簿作成（User1が請求者、User2が支払者）
	data.Bill, data.Items, err = f.NewBill().
		WithRequester(data.User1.ID).
		WithPayer(data.User2.ID).
		AddItems(
			Item("食費", 5000.0),
			Item("光熱費", 8000.0),
		).
		Build()

	if err != nil {
		return nil, fmt.Errorf("家計簿作成失敗: %w", err)
	}

	return data, nil
}

// StandardTestData 標準的なテストデータセット
type StandardTestData struct {
	User1 models.User
	User2 models.User
	User3 models.User
	Bill  models.MonthlyBill
	Items []models.BillItem
}

// ========================================
// 高速テストデータ生成（メモリ上のみ）
// ========================================

// CreateQuickTestData 高速テストデータ生成（DB保存なし）
func (f *TestDataFactory) CreateQuickTestData() *StandardTestData {
	user1 := f.NewUser().WithName("クイックユーザー1").WithAccountID("quick1").AsTransient().MustBuild()
	user2 := f.NewUser().WithName("クイックユーザー2").WithAccountID("quick2").AsTransient().MustBuild()
	user3 := f.NewUser().WithName("クイックユーザー3").WithAccountID("quick3").AsTransient().MustBuild()

	bill, items := f.NewBill().
		WithRequester(user1.ID).
		WithPayer(user2.ID).
		AddItems(
			Item("テスト項目1", 1000.0),
			Item("テスト項目2", 2000.0),
		).
		AsTransient().
		MustBuild()

	return &StandardTestData{
		User1: user1,
		User2: user2,
		User3: user3,
		Bill:  bill,
		Items: items,
	}
}
