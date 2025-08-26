package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/utrack/gin-csrf"

	"money_management/internal/database"
	"money_management/internal/handlers"
	"money_management/internal/middleware"
)

// main メイン関数
// アプリケーションのエントリーポイント
// データベース接続、ルーティング設定、サーバー起動を行う
func main() {
	// タイムゾーンを日本時間に設定
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatal("タイムゾーン設定に失敗しました:", err)
	}
	time.Local = loc

	// データベース接続を初期化
	if err := database.Init(); err != nil {
		log.Fatal("データベースに接続できませんでした:", err)
	}

	// Gin設定（本番環境ではリリースモードに設定）
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Ginルーターを作成
	r := gin.Default()

	// セッション設定（CSRF保護に必要）
	// セッションシークレットを安全に取得
	sessionSecret := middleware.GetSessionSecret()
	store := cookie.NewStore(sessionSecret)
	r.Use(sessions.Sessions("household_budget_session", store))

	// CORS設定
	// フロントエンドからのアクセスを許可するための設定
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://frontend:80"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// エラーハンドリングミドルウェアを適用（最初に適用）
	r.Use(middleware.ErrorHandlingMiddleware())

	// セキュリティ監査ログミドルウェア
	r.Use(middleware.SecurityAuditMiddleware())

	// セキュリティヘッダーミドルウェアを適用
	r.Use(middleware.SecurityHeadersMiddleware())

	// 開発環境用ヘッダー（本番では無効化）
	if os.Getenv("GIN_MODE") != "release" {
		r.Use(middleware.DevelopmentHeadersMiddleware())
	}

	// 入力値検証ミドルウェアを適用
	r.Use(middleware.InputValidationMiddleware())

	// レート制限ミドルウェアを適用（全体的な制限）
	r.Use(middleware.RateLimitMiddleware())

	// CSRF保護ミドルウェアを適用
	// CSRFシークレットを安全に取得
	csrfSecret := middleware.GetCSRFSecret()
	r.Use(csrf.Middleware(csrf.Options{
		Secret: csrfSecret,
		ErrorFunc: func(c *gin.Context) {
			c.JSON(403, gin.H{
				"error": "CSRF トークンが無効です。ページを更新してやり直してください。",
				"code":  "INVALID_CSRF_TOKEN",
			})
			c.Abort()
		},
	}))

	// ルート設定
	setupRoutes(r)

	// ヘルスチェックエンドポイント
	// アプリケーションの稼働状況を確認するためのエンドポイント
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "message": "家計簿API稼働中"})
	})

	// CSRFトークン取得エンドポイント
	// フロントエンドがCSRFトークンを取得するためのエンドポイント
	r.GET("/api/csrf-token", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"csrf_token": csrf.GetToken(c),
		})
	})

	// セキュリティ監視エンドポイント（開発・管理用）
	r.GET("/api/security-status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"rate_limit":      middleware.GetRateLimitStatus(),
			"token_blacklist": middleware.GetTokenBlacklistStatus(),
		})
	})

	// サーバー起動
	log.Println("🚀 サーバーを開始します: :8080")
	r.Run(":8080")
}

// setupRoutes ルーティングを設定する
// API endpoints を定義し、各ハンドラーを関連付ける
func setupRoutes(r *gin.Engine) {
	// API group を作成
	api := r.Group("/api")
	api.Use(middleware.APISecurityHeadersMiddleware()) // API専用セキュリティヘッダー
	{
		// 認証関連のエンドポイント（厳格なレート制限）
		auth := api.Group("/auth")
		auth.Use(middleware.AuthRateLimitMiddleware())         // 認証用レート制限
		auth.Use(middleware.StrictInputValidationMiddleware()) // 厳格な入力値検証
		{
			auth.POST("/login", handlers.LoginHandler)                          // ログイン
			auth.POST("/register", handlers.RegisterHandler)                    // ユーザー登録
			auth.GET("/me", middleware.AuthMiddleware(), handlers.GetMeHandler) // 現在のユーザー情報取得

			// セッション管理エンドポイント（認証が必要）
			auth.POST("/logout", middleware.AuthMiddleware(), handlers.LogoutHandler)              // ログアウト
			auth.POST("/logout-all", middleware.AuthMiddleware(), handlers.LogoutAllHandler)       // 全デバイスログアウト
			auth.GET("/token-status", middleware.AuthMiddleware(), handlers.GetTokenStatusHandler) // トークンステータス確認
		}

		// ユーザー関連のエンドポイント
		users := api.Group("/users")
		users.Use(middleware.AuthMiddleware())
		{
			users.GET("", handlers.GetUsersHandler) // ユーザー一覧取得
		}

		// 家計簿関連のエンドポイント（認証が必要）
		bills := api.Group("/bills")
		bills.Use(middleware.AuthMiddleware())
		{
			bills.GET("", handlers.GetBillsListHandler)         // 家計簿一覧取得
			bills.GET("/:year/:month", handlers.GetBillHandler) // 特定年月の家計簿取得

			// 作成系操作には追加のレート制限
			billsCreate := bills.Group("")
			billsCreate.Use(middleware.CreateRateLimitMiddleware())
			{
				billsCreate.POST("", handlers.CreateBillHandler)             // 新規家計簿作成
				billsCreate.PUT("/:id/items", handlers.UpdateItemsHandler)   // 家計簿項目更新
				billsCreate.PUT("/:id/request", handlers.RequestBillHandler) // 家計簿請求
				billsCreate.PUT("/:id/payment", handlers.PaymentBillHandler) // 家計簿支払い確認
				billsCreate.DELETE("/:id", handlers.DeleteBillHandler)       // 家計簿削除
			}
		}
	}
}
