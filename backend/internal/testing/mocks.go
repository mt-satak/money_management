// ========================================
// モック/スタブパターン実装
// 外部依存性の分離とテスト容易性の向上
// ========================================

package testing

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"money_management/internal/models"
)

// ========================================
// データベースインターフェースの抽象化
// ========================================

// DBInterface データベース操作の抽象化インターフェース
type DBInterface interface {
	Create(value interface{}) *gorm.DB
	First(dest interface{}, conds ...interface{}) *gorm.DB
	Where(query interface{}, args ...interface{}) DBInterface
	Find(dest interface{}, conds ...interface{}) *gorm.DB
	Save(value interface{}) *gorm.DB
	Delete(value interface{}, conds ...interface{}) *gorm.DB
	Select(query interface{}, args ...interface{}) DBInterface
	Preload(query string, args ...interface{}) DBInterface
	Transaction(fc func(tx *gorm.DB) error) error
	DB() (*gorm.DB, error)
}

// GormDBWrapper GORMの実装をラップ
type GormDBWrapper struct {
	db *gorm.DB
}

// NewGormDBWrapper GORMラッパーの作成
func NewGormDBWrapper(db *gorm.DB) *GormDBWrapper {
	return &GormDBWrapper{db: db}
}

// Create データ作成
func (w *GormDBWrapper) Create(value interface{}) *gorm.DB {
	return w.db.Create(value)
}

// First 最初のレコードを取得
func (w *GormDBWrapper) First(dest interface{}, conds ...interface{}) *gorm.DB {
	return w.db.First(dest, conds...)
}

// Where 条件指定
func (w *GormDBWrapper) Where(query interface{}, args ...interface{}) DBInterface {
	return &GormDBWrapper{db: w.db.Where(query, args...)}
}

// Find 複数レコード取得
func (w *GormDBWrapper) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	return w.db.Find(dest, conds...)
}

// Save データ保存
func (w *GormDBWrapper) Save(value interface{}) *gorm.DB {
	return w.db.Save(value)
}

// Delete データ削除
func (w *GormDBWrapper) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	return w.db.Delete(value, conds...)
}

// Select 選択
func (w *GormDBWrapper) Select(query interface{}, args ...interface{}) DBInterface {
	return &GormDBWrapper{db: w.db.Select(query, args...)}
}

// Preload プリロード
func (w *GormDBWrapper) Preload(query string, args ...interface{}) DBInterface {
	return &GormDBWrapper{db: w.db.Preload(query, args...)}
}

// Transaction トランザクション
func (w *GormDBWrapper) Transaction(fc func(tx *gorm.DB) error) error {
	return w.db.Transaction(fc)
}

// DB 元のDB取得
func (w *GormDBWrapper) DB() (*gorm.DB, error) {
	_, err := w.db.DB()
	return w.db, err
}

// ========================================
// モックデータベース実装
// ========================================

// MockDB モックデータベース実装
type MockDB struct {
	users       map[string]*models.User      // AccountIDをキーとするマップ
	bills       map[uint]*models.MonthlyBill // IDをキーとするマップ
	items       map[uint]*models.BillItem    // IDをキーとするマップ
	nextUserID  uint
	nextBillID  uint
	nextItemID  uint
	shouldError bool
	errorType   string
}

// NewMockDB モックデータベースの初期化
func NewMockDB() *MockDB {
	return &MockDB{
		users:      make(map[string]*models.User),
		bills:      make(map[uint]*models.MonthlyBill),
		items:      make(map[uint]*models.BillItem),
		nextUserID: 1,
		nextBillID: 1,
		nextItemID: 1,
	}
}

// SetError エラー状態を設定
func (m *MockDB) SetError(errorType string) {
	m.shouldError = true
	m.errorType = errorType
}

// ClearError エラー状態をクリア
func (m *MockDB) ClearError() {
	m.shouldError = false
	m.errorType = ""
}

// Create データ作成のモック
func (m *MockDB) Create(value interface{}) *gorm.DB {
	if m.shouldError {
		return &gorm.DB{Error: errors.New(m.errorType)}
	}

	switch v := value.(type) {
	case *models.User:
		// 重複チェック
		if _, exists := m.users[v.AccountID]; exists {
			return &gorm.DB{Error: errors.New("duplicate key value violates unique constraint")}
		}
		v.ID = m.nextUserID
		m.nextUserID++
		v.CreatedAt = time.Now()
		v.UpdatedAt = time.Now()
		m.users[v.AccountID] = v

	case *models.MonthlyBill:
		v.ID = m.nextBillID
		m.nextBillID++
		v.CreatedAt = time.Now()
		v.UpdatedAt = time.Now()
		m.bills[v.ID] = v

	case *models.BillItem:
		v.ID = m.nextItemID
		m.nextItemID++
		v.CreatedAt = time.Now()
		v.UpdatedAt = time.Now()
		m.items[v.ID] = v
	}

	return &gorm.DB{Error: nil}
}

// MockDBQuery クエリ用の中間構造体
type MockDBQuery struct {
	mockDB *MockDB
	query  string
	args   []interface{}
}

// Create MockDBQueryでのCreate実装
func (q *MockDBQuery) Create(value interface{}) *gorm.DB {
	return q.mockDB.Create(value)
}

// First 最初のレコード取得のモック
func (m *MockDB) First(dest interface{}, conds ...interface{}) *gorm.DB {
	if m.shouldError {
		return &gorm.DB{Error: errors.New(m.errorType)}
	}

	switch d := dest.(type) {
	case *models.User:
		if len(conds) > 0 {
			if userID, ok := conds[0].(uint); ok {
				for _, user := range m.users {
					if user.ID == userID {
						*d = *user
						return &gorm.DB{Error: nil}
					}
				}
			}
		}
		return &gorm.DB{Error: gorm.ErrRecordNotFound}

	case *models.MonthlyBill:
		if len(conds) > 0 {
			if billID, ok := conds[0].(uint); ok {
				if bill, exists := m.bills[billID]; exists {
					*d = *bill
					return &gorm.DB{Error: nil}
				}
			}
		}
		return &gorm.DB{Error: gorm.ErrRecordNotFound}
	}

	return &gorm.DB{Error: gorm.ErrRecordNotFound}
}

// Where 条件指定のモック
func (m *MockDB) Where(query interface{}, args ...interface{}) DBInterface {
	return &MockDBQuery{
		mockDB: m,
		query:  query.(string),
		args:   args,
	}
}

// First MockDBQueryの場合
func (q *MockDBQuery) First(dest interface{}, conds ...interface{}) *gorm.DB {
	if q.mockDB.shouldError {
		return &gorm.DB{Error: errors.New(q.mockDB.errorType)}
	}

	switch d := dest.(type) {
	case *models.User:
		if q.query == "account_id = ?" && len(q.args) > 0 {
			if accountID, ok := q.args[0].(string); ok {
				if user, exists := q.mockDB.users[accountID]; exists {
					*d = *user
					return &gorm.DB{Error: nil}
				}
			}
		}
		return &gorm.DB{Error: gorm.ErrRecordNotFound}

	case *models.MonthlyBill:
		// 年月での検索やその他の条件検索をサポート
		for _, bill := range q.mockDB.bills {
			// 簡単な条件マッチング（実際の実装では正規表現などを使用）
			*d = *bill
			return &gorm.DB{Error: nil}
		}
		return &gorm.DB{Error: gorm.ErrRecordNotFound}
	}

	return &gorm.DB{Error: gorm.ErrRecordNotFound}
}

// Find 複数レコード取得のモック
func (m *MockDB) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	if m.shouldError {
		return &gorm.DB{Error: errors.New(m.errorType)}
	}

	switch d := dest.(type) {
	case *[]models.User:
		users := make([]models.User, 0, len(m.users))
		for _, user := range m.users {
			users = append(users, *user)
		}
		*d = users

	case *[]models.MonthlyBill:
		bills := make([]models.MonthlyBill, 0, len(m.bills))
		for _, bill := range m.bills {
			bills = append(bills, *bill)
		}
		*d = bills
	}

	return &gorm.DB{Error: nil}
}

// 他のメソッドのスタブ実装
func (m *MockDB) Save(value interface{}) *gorm.DB {
	return &gorm.DB{Error: nil}
}

func (m *MockDB) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	return &gorm.DB{Error: nil}
}

func (m *MockDB) Select(query interface{}, args ...interface{}) DBInterface {
	return m
}

func (m *MockDB) Preload(query string, args ...interface{}) DBInterface {
	return m
}

func (m *MockDB) Transaction(fc func(tx *gorm.DB) error) error {
	return nil // 簡単な実装: 常に成功
}

func (m *MockDB) DB() (*gorm.DB, error) {
	return nil, nil
}

func (q *MockDBQuery) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	return q.mockDB.Find(dest, conds...)
}

func (q *MockDBQuery) Save(value interface{}) *gorm.DB {
	return q.mockDB.Save(value)
}

func (q *MockDBQuery) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	return q.mockDB.Delete(value, conds...)
}

func (q *MockDBQuery) Select(query interface{}, args ...interface{}) DBInterface {
	return &MockDBQuery{
		mockDB: q.mockDB,
		query:  q.query,
		args:   q.args,
	}
}

func (q *MockDBQuery) Preload(query string, args ...interface{}) DBInterface {
	return &MockDBQuery{
		mockDB: q.mockDB,
		query:  q.query,
		args:   q.args,
	}
}

func (q *MockDBQuery) Transaction(fc func(tx *gorm.DB) error) error {
	return q.mockDB.Transaction(fc)
}

func (q *MockDBQuery) DB() (*gorm.DB, error) {
	return q.mockDB.DB()
}

func (q *MockDBQuery) Where(query interface{}, args ...interface{}) DBInterface {
	return q.mockDB.Where(query, args...)
}

// ========================================
// パスワードハッシュ化のモック
// ========================================

// PasswordHasherInterface パスワードハッシュ化の抽象化
type PasswordHasherInterface interface {
	HashPassword(password string) (string, error)
	ComparePassword(hashedPassword, password string) error
}

// MockPasswordHasher モックパスワードハッシャー
type MockPasswordHasher struct {
	shouldError bool
}

// NewMockPasswordHasher モックパスワードハッシャーの初期化
func NewMockPasswordHasher() *MockPasswordHasher {
	return &MockPasswordHasher{}
}

// SetError エラー状態を設定
func (m *MockPasswordHasher) SetError(shouldError bool) {
	m.shouldError = shouldError
}

// HashPassword パスワードハッシュ化のモック（実際はハッシュ化しない）
func (m *MockPasswordHasher) HashPassword(password string) (string, error) {
	if m.shouldError {
		return "", errors.New("hashing failed")
	}
	return "mock_hashed_" + password, nil
}

// ComparePassword パスワード比較のモック
func (m *MockPasswordHasher) ComparePassword(hashedPassword, password string) error {
	if m.shouldError {
		return errors.New("comparison failed")
	}

	expected := "mock_hashed_" + password
	if hashedPassword != expected {
		return errors.New("password mismatch")
	}

	return nil
}

// ========================================
// JWTトークンサービスのモック
// ========================================

// JWTServiceInterface JWTサービスの抽象化
type JWTServiceInterface interface {
	GenerateToken(userID uint) (string, error)
	ValidateToken(tokenString string) (uint, error)
}

// MockJWTService モックJWTサービス
type MockJWTService struct {
	shouldError bool
	tokens      map[string]uint // token -> userID のマッピング
}

// NewMockJWTService モックJWTサービスの初期化
func NewMockJWTService() *MockJWTService {
	return &MockJWTService{
		tokens: make(map[string]uint),
	}
}

// SetError エラー状態を設定
func (m *MockJWTService) SetError(shouldError bool) {
	m.shouldError = shouldError
}

// GenerateToken トークン生成のモック
func (m *MockJWTService) GenerateToken(userID uint) (string, error) {
	if m.shouldError {
		return "", errors.New("token generation failed")
	}

	token := "mock_token_" + string(rune(userID+'0'))
	m.tokens[token] = userID
	return token, nil
}

// ValidateToken トークン検証のモック
func (m *MockJWTService) ValidateToken(tokenString string) (uint, error) {
	if m.shouldError {
		return 0, errors.New("token validation failed")
	}

	if userID, exists := m.tokens[tokenString]; exists {
		return userID, nil
	}

	return 0, errors.New("invalid token")
}

// ========================================
// テストヘルパー
// ========================================

// MockTestDependencies モックテスト用の依存性集合
type MockTestDependencies struct {
	DB             *MockDB
	PasswordHasher *MockPasswordHasher
	JWTService     *MockJWTService
}

// NewMockTestDependencies モック依存性の一括初期化
func NewMockTestDependencies() *MockTestDependencies {
	return &MockTestDependencies{
		DB:             NewMockDB(),
		PasswordHasher: NewMockPasswordHasher(),
		JWTService:     NewMockJWTService(),
	}
}

// SetupTestUserWithMock モックを使用したテストユーザーのセットアップ
func (deps *MockTestDependencies) SetupTestUserWithMock(accountID, password string) (*models.User, error) {
	hashedPassword, err := deps.PasswordHasher.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:         "テストユーザー",
		AccountID:    accountID,
		PasswordHash: hashedPassword,
	}

	result := deps.DB.Create(user)
	if result.Error != nil {
		return nil, result.Error
	}

	return user, nil
}
