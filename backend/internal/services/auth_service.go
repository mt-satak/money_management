// ========================================
// 認証サービス - ビジネスロジックの分離
// モック/スタブパターンによるテスト容易性の実現
// ========================================

package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"money_management/internal/middleware"
	"money_management/internal/models"
	testmocks "money_management/internal/testing"
)

// ========================================
// 認証サービスのインターフェース定義
// ========================================

// AuthServiceInterface 認証サービスの抽象化
type AuthServiceInterface interface {
	Login(accountID, password string) (*LoginResult, error)
	Register(name, accountID, password string) (*models.User, error)
	GetUserByID(userID uint) (*models.User, error)
	GetAllUsers() ([]models.User, error)
}

// LoginResult ログイン結果
type LoginResult struct {
	User  *models.User `json:"user"`
	Token string       `json:"token"`
}

// ========================================
// 実際の認証サービス実装
// ========================================

// AuthService 認証サービスの実装
type AuthService struct {
	db             testmocks.DBInterface
	passwordHasher testmocks.PasswordHasherInterface
	jwtService     testmocks.JWTServiceInterface
}

// NewAuthService 認証サービスの初期化
func NewAuthService(db testmocks.DBInterface, passwordHasher testmocks.PasswordHasherInterface, jwtService testmocks.JWTServiceInterface) *AuthService {
	return &AuthService{
		db:             db,
		passwordHasher: passwordHasher,
		jwtService:     jwtService,
	}
}

// NewAuthServiceWithDB 実際のデータベースを使用する認証サービスの初期化
func NewAuthServiceWithDB(db *gorm.DB) *AuthService {
	return &AuthService{
		db:             testmocks.NewGormDBWrapper(db),
		passwordHasher: &BcryptPasswordHasher{},
		jwtService:     &JWTService{},
	}
}

// Login ユーザーログイン
func (s *AuthService) Login(accountID, password string) (*LoginResult, error) {
	// データベースからユーザーを取得
	var user models.User
	if err := s.db.Where("account_id = ?", accountID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("認証情報が無効です")
		}
		return nil, err
	}

	// パスワードを検証
	if err := s.passwordHasher.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, errors.New("認証情報が無効です")
	}

	// JWTトークンを生成
	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		return nil, errors.New("トークンを生成できませんでした")
	}

	return &LoginResult{
		User:  &user,
		Token: token,
	}, nil
}

// Register ユーザー登録
func (s *AuthService) Register(name, accountID, password string) (*models.User, error) {
	// パスワードのバリデーション
	if len(password) < 8 {
		return nil, errors.New("パスワードは8文字以上で入力してください")
	}

	if len(accountID) < 3 || len(accountID) > 20 {
		return nil, errors.New("アカウントIDは3文字以上20文字以下で入力してください")
	}

	// 既存ユーザーの重複チェック
	var existingUser models.User
	if err := s.db.Where("account_id = ?", accountID).First(&existingUser).Error; err == nil {
		return nil, errors.New("このアカウントIDは既に使用されています")
	}

	// パスワードハッシュ化
	hashedPassword, err := s.passwordHasher.HashPassword(password)
	if err != nil {
		return nil, errors.New("パスワードの処理に失敗しました")
	}

	// ユーザー作成
	user := models.User{
		Name:         name,
		AccountID:    accountID,
		PasswordHash: hashedPassword,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, errors.New("ユーザーの作成に失敗しました")
	}

	return &user, nil
}

// GetUserByID ユーザーIDでユーザーを取得
func (s *AuthService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("ユーザーが見つかりません")
		}
		return nil, err
	}

	return &user, nil
}

// GetAllUsers 全ユーザーを取得
func (s *AuthService) GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, errors.New("ユーザー一覧の取得に失敗しました")
	}

	return users, nil
}

// ========================================
// 実際の実装クラス（本番用）
// ========================================

// BcryptPasswordHasher 実際のbcryptハッシャー
type BcryptPasswordHasher struct{}

// HashPassword パスワードハッシュ化
func (h *BcryptPasswordHasher) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// ComparePassword パスワード比較
func (h *BcryptPasswordHasher) ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// JWTService 実際のJWTサービス
type JWTService struct{}

// GenerateToken JWTトークン生成
func (j *JWTService) GenerateToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})

	tokenString, err := token.SignedString(middleware.GetJWTSecret())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken JWTトークン検証
func (j *JWTService) ValidateToken(tokenString string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return middleware.GetJWTSecret(), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(*middleware.Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return 0, errors.New("invalid token")
}
