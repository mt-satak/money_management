package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"money_management/internal/database"
	"money_management/internal/middleware"
	"money_management/internal/models"
)

// LoginHandler ログインハンドラー
// ユーザーの認証情報を確認し、有効な場合はJWTトークンを発行する
func LoginHandler(c *gin.Context) {
	LoginHandlerWithDB(database.GetDB())(c)
}

// LoginHandlerWithDB DB接続を注入可能なログインハンドラー
func LoginHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.LoginRequest

		// リクエストボディをバインド（JSONをGoの構造体に変換）
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// データベースからユーザーを取得
		var user models.User
		if err := db.Where("account_id = ?", req.AccountID).First(&user).Error; err != nil {
			// セキュリティ上、具体的なエラー内容は返さない
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証情報が無効です"})
			return
		}

		// パスワードを検証
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証情報が無効です"})
			return
		}

		// JWTトークンを生成
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{
			UserID: user.ID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24時間有効
			},
		})

		// トークンに署名
		tokenString, err := token.SignedString(middleware.GetJWTSecret())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "トークンを生成できませんでした"})
			return
		}

		// ログイン成功レスポンス
		c.JSON(http.StatusOK, models.LoginResponse{
			Token: tokenString,
			User:  user,
		})
	}
}

// RegisterHandler ユーザー登録ハンドラー
// 新規ユーザーを登録し、登録完了時にJWTトークンを発行する
func RegisterHandler(c *gin.Context) {
	RegisterHandlerWithDB(database.GetDB())(c)
}

// RegisterHandlerWithDB DB接続を注入可能な登録ハンドラー
func RegisterHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.RegisterRequest

		// リクエストボディをバインド
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// パスワードの長さを検証（最小6文字）
		if len(req.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "パスワードは6文字以上で入力してください"})
			return
		}

		// アカウントIDの形式チェック（英数字とアンダースコアのみ、3-20文字）
		if len(req.AccountID) < 3 || len(req.AccountID) > 20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "アカウントIDは3文字以上20文字以下で入力してください"})
			return
		}

		// 既存のユーザーをチェック（アカウントIDの重複確認）
		var existingUser models.User
		if err := db.Where("account_id = ?", req.AccountID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "このアカウントIDは既に使用されています"})
			return
		}

		// パスワードをハッシュ化
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "パスワードの暗号化に失敗しました"})
			return
		}

		// 新しいユーザーを作成
		user := models.User{
			Name:         req.Name,
			AccountID:    req.AccountID,
			PasswordHash: string(hashedPassword),
		}

		// データベースにユーザーを保存
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ユーザーの作成に失敗しました"})
			return
		}

		// JWTトークンを生成
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{
			UserID: user.ID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		})

		// トークンに署名
		tokenString, err := token.SignedString(middleware.GetJWTSecret())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "トークンを生成できませんでした"})
			return
		}

		// 登録成功レスポンス
		c.JSON(http.StatusCreated, models.LoginResponse{
			Token: tokenString,
			User:  user,
		})
	}
}

// GetMeHandler 現在のユーザー情報取得ハンドラー
// JWTトークンから取得したユーザーIDを使用してユーザー情報を返す
func GetMeHandler(c *gin.Context) {
	GetMeHandlerWithDB(database.GetDB())(c)
}

// GetMeHandlerWithDB DB接続を注入可能なユーザー情報取得ハンドラー
func GetMeHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ミドルウェアで設定されたユーザーIDを取得
		userID := c.GetUint("user_id")

		// データベースからユーザーを取得
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "ユーザーが見つかりません"})
			return
		}

		// ユーザー情報を返す
		c.JSON(http.StatusOK, user)
	}
}

// GetUsersHandler ユーザー一覧取得ハンドラー
// 家計簿作成時の支払者選択用にユーザー一覧を取得
func GetUsersHandler(c *gin.Context) {
	GetUsersHandlerWithDB(database.GetDB())(c)
}

// GetUsersHandlerWithDB DB接続を注入可能なユーザー一覧取得ハンドラー
func GetUsersHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User

		// パスワード情報を除外してユーザー一覧を取得
		err := db.Select("id, name, account_id, created_at, updated_at").Find(&users).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ユーザー一覧の取得に失敗しました"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}
