package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"money_management/internal/database"
	"money_management/internal/models"
)

// GetBillHandler 特定年月の家計簿取得ハンドラー
// 指定された年月の家計簿データを取得し、総額も計算して返す
func GetBillHandler(c *gin.Context) {
	GetBillHandlerWithDB(database.GetDB())(c)
}

// GetBillHandlerWithDB DB接続を注入可能な家計簿取得ハンドラー
func GetBillHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// URLパラメータから年・月・ユーザーIDを取得
		year, _ := strconv.Atoi(c.Param("year"))
		month, _ := strconv.Atoi(c.Param("month"))
		userID := c.GetUint("user_id")

		var bill models.MonthlyBill
		// 関連データ（請求者・支払者・項目）をプリロードして家計簿を検索
		// 対象ユーザーが請求者または支払者である家計簿のみ取得
		err := db.Preload("Requester").Preload("Payer").Preload("Items").
			Where("year = ? AND month = ? AND (requester_id = ? OR payer_id = ?)", year, month, userID, userID).
			First(&bill).Error

		// 家計簿が見つからない場合はnullを返す
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"bill": nil})
			return
		}

		// 総金額を計算
		totalAmount := 0.0
		for _, item := range bill.Items {
			totalAmount += item.Amount
		}

		// レスポンスデータを作成
		response := models.BillResponse{
			MonthlyBill: bill,
			TotalAmount: totalAmount,
		}

		c.JSON(http.StatusOK, response)
	}
}

// CreateBillHandler 新規家計簿作成ハンドラー
// 指定された年月の家計簿を新規作成する
func CreateBillHandler(c *gin.Context) {
	CreateBillHandlerWithDB(database.GetDB())(c)
}

// CreateBillHandlerWithDB DB接続を注入可能な家計簿作成ハンドラー
func CreateBillHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		// リクエストデータの構造体定義
		var req struct {
			Year    int  `json:"year" binding:"required"`     // 対象年（必須）
			Month   int  `json:"month" binding:"required"`    // 対象月（必須）
			PayerID uint `json:"payer_id" binding:"required"` // 支払者ID（必須）
		}

		// リクエストボディをバインド
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 請求者と支払者が同一ユーザーの場合はエラー
		if userID == req.PayerID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "請求者と支払者は異なるユーザーである必要があります"})
			return
		}

		// 新しい家計簿を作成
		bill := models.MonthlyBill{
			Year:        req.Year,
			Month:       req.Month,
			RequesterID: userID,      // 請求者は現在のユーザー
			PayerID:     req.PayerID, // 支払者は指定されたユーザー
			Status:      "pending",   // 初期状態は作成中
		}

		// データベースに保存（デッドロック対応のリトライ機構付き）
		log.Printf("🔍 About to create bill: Year=%d, Month=%d, RequesterID=%d", bill.Year, bill.Month, bill.RequesterID)

		const (
			maxRetries         = 3
			baseBackoffMs      = 100 // ベースバックオフ時間（ミリ秒）
			backoffIncrementMs = 50  // バックオフ増分（ミリ秒）
		)
		var result *gorm.DB
		var err error

		for i := 0; i < maxRetries; i++ {
			result = db.Create(&bill)
			err = result.Error

			if err == nil {
				break
			}

			// デッドロックエラーの場合はリトライ
			if strings.Contains(err.Error(), "Deadlock found when trying to get lock") {
				log.Printf("🔄 Deadlock detected, retrying... (attempt %d/%d)", i+1, maxRetries)
				// 穏やかな指数バックオフ: baseTime + incrementTime * attempt^2
				waitTime := time.Duration(baseBackoffMs+backoffIncrementMs*i*i) * time.Millisecond
				log.Printf("🕐 Waiting %v before retry", waitTime)
				time.Sleep(waitTime)
				continue
			}

			// デッドロック以外のエラーは即座に終了
			break
		}

		log.Printf("🔍 DB Create completed, checking for errors...")

		if err != nil {
			log.Printf("🔍 CreateBill Error detected: %s", err.Error())

			// 制約エラー（重複）の場合は409 Conflictを返す
			errorStr := err.Error()
			log.Printf("🔍 Checking if error contains 'Duplicate entry': %t", strings.Contains(errorStr, "Duplicate entry"))

			if strings.Contains(errorStr, "Duplicate entry") {
				log.Printf("🔍 Returning 409 Conflict for duplicate entry")
				c.JSON(http.StatusConflict, gin.H{
					"error": "指定された年月の家計簿は既に存在します"})
				return
			}

			// その他のエラーの場合は500エラー
			log.Printf("🔍 Returning 500 for other database error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "家計簿の作成に失敗しました"})
			return
		}

		log.Printf("🔍 Bill created successfully")

		// 作成後、関連データをプリロードして再取得
		db.Preload("Requester").Preload("Payer").Preload("Items").First(&bill, bill.ID)

		// レスポンスデータを作成（新規作成時は項目がないので金額は0）
		response := models.BillResponse{
			MonthlyBill: bill,
			TotalAmount: 0,
		}

		c.JSON(http.StatusCreated, response)
	}
}

// UpdateItemsHandler 家計簿項目更新ハンドラー
// 家計簿の項目（支出項目）を更新する
func UpdateItemsHandler(c *gin.Context) {
	UpdateItemsHandlerWithDB(database.GetDB())(c)
}

// UpdateItemsHandlerWithDB DB接続を注入可能な項目更新ハンドラー
func UpdateItemsHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// URLパラメータから家計簿IDを取得
		billID, _ := strconv.Atoi(c.Param("id"))
		userID := c.GetUint("user_id")

		// 対象の家計簿を検索（請求者のみが更新可能）
		var bill models.MonthlyBill
		if err := db.Where("id = ? AND requester_id = ?", billID, userID).First(&bill).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "家計簿が見つかりません"})
			return
		}

		// pending状態（作成中）の家計簿のみ更新可能
		if bill.Status != "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "確定済みの家計簿の項目は更新できません"})
			return
		}

		// リクエストデータの構造体定義
		var req struct {
			Items []struct {
				ID       uint    `json:"id"`        // 項目ID（更新時に使用）
				ItemName string  `json:"item_name"` // 項目名
				Amount   float64 `json:"amount"`    // 金額
			} `json:"items"`
		}

		// リクエストボディをバインド
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 既存の項目を全削除
		db.Where("bill_id = ?", billID).Delete(&models.BillItem{})

		// 新しい項目を追加（名前と金額が有効な項目のみ）
		for _, item := range req.Items {
			if item.ItemName != "" && item.Amount > 0 {
				billItem := models.BillItem{
					BillID:   uint(billID),
					ItemName: item.ItemName,
					Amount:   item.Amount,
				}
				db.Create(&billItem)
			}
		}

		// 更新後のデータを関連データと共に再取得
		db.Preload("Requester").Preload("Payer").Preload("Items").First(&bill, billID)

		// 総金額を再計算
		totalAmount := 0.0
		for _, item := range bill.Items {
			totalAmount += item.Amount
		}

		// レスポンスデータを作成
		response := models.BillResponse{
			MonthlyBill: bill,
			TotalAmount: totalAmount,
		}

		c.JSON(http.StatusOK, response)
	}
}

// RequestBillHandler 家計簿請求ハンドラー
// 家計簿の状態をpending（作成中）からrequested（請求済み）に変更する
func RequestBillHandler(c *gin.Context) {
	RequestBillHandlerWithDB(database.GetDB())(c)
}

// RequestBillHandlerWithDB DB接続を注入可能な請求ハンドラー
func RequestBillHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// URLパラメータから家計簿IDを取得
		billID, _ := strconv.Atoi(c.Param("id"))
		userID := c.GetUint("user_id")

		// 対象の家計簿を検索（請求者のみが請求可能）
		var bill models.MonthlyBill
		if err := db.Where("id = ? AND requester_id = ?", billID, userID).First(&bill).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "家計簿が見つかりません"})
			return
		}

		// pending状態（作成中）の家計簿のみ請求可能
		if bill.Status != "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "家計簿は既に確定済みです"})
			return
		}

		// 状態を請求済みに変更し、請求日時を設定
		now := time.Now()
		bill.Status = "requested"
		bill.RequestDate = &now

		// データベースを更新
		if err := db.Save(&bill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "家計簿の更新に失敗しました"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "家計簿の請求が確定しました"})
	}
}

// PaymentBillHandler 家計簿支払ハンドラー
// 家計簿の状態をrequested（請求済み）からpaid（支払済み）に変更する
func PaymentBillHandler(c *gin.Context) {
	PaymentBillHandlerWithDB(database.GetDB())(c)
}

// PaymentBillHandlerWithDB DB接続を注入可能な支払ハンドラー
func PaymentBillHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// URLパラメータから家計簿IDを取得
		billID, _ := strconv.Atoi(c.Param("id"))
		userID := c.GetUint("user_id")

		// 対象の家計簿を検索（支払者のみが支払い処理可能）
		var bill models.MonthlyBill
		if err := db.Where("id = ? AND payer_id = ?", billID, userID).First(&bill).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "家計簿が見つかりません"})
			return
		}

		// requested状態（請求済み）の家計簿のみ支払い処理可能
		if bill.Status != "requested" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "家計簿が請求中状態ではありません"})
			return
		}

		// 状態を支払済みに変更し、支払日時を設定
		now := time.Now()
		bill.Status = "paid"
		bill.PaymentDate = &now

		// データベースを更新
		if err := db.Save(&bill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "家計簿の更新に失敗しました"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "支払いが確定しました"})
	}
}

// DeleteBillHandler 家計簿削除ハンドラー
// 請求者（requester）が作成中（pending）状態の家計簿を削除する
func DeleteBillHandler(c *gin.Context) {
	DeleteBillHandlerWithDB(database.GetDB())(c)
}

// DeleteBillHandlerWithDB DB接続を注入可能な削除ハンドラー
func DeleteBillHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// URLパラメータから家計簿IDを取得
		billID, _ := strconv.Atoi(c.Param("id"))
		userID := c.GetUint("user_id")

		// 対象の家計簿を検索（請求者のみが削除可能）
		var bill models.MonthlyBill
		if err := db.Where("id = ? AND requester_id = ?", billID, userID).First(&bill).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "家計簿が見つかりません"})
			return
		}

		// pending状態（作成中）の家計簿のみ削除可能
		if bill.Status != "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "確定済みの家計簿は削除できません"})
			return
		}

		// 家計簿に関連する項目を先に削除（外部キー制約対応）
		if err := db.Where("bill_id = ?", billID).Delete(&models.BillItem{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "家計簿項目の削除に失敗しました"})
			return
		}

		// 家計簿本体を削除
		if err := db.Delete(&bill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "家計簿の削除に失敗しました"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "家計簿を削除しました"})
	}
}

// GetBillsListHandler 家計簿一覧取得ハンドラー
// ユーザーに関連する全ての家計簿を取得（請求者・支払者どちらでも）
func GetBillsListHandler(c *gin.Context) {
	GetBillsListHandlerWithDB(database.GetDB())(c)
}

// GetBillsListHandlerWithDB DB接続を注入可能な一覧取得ハンドラー
func GetBillsListHandlerWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		var bills []models.MonthlyBill
		// 関連データと共に家計簿一覧を取得（作成日時の降順）
		err := db.Preload("Requester").Preload("Payer").Preload("Items").
			Where("requester_id = ? OR payer_id = ?", userID, userID).
			Order("year DESC, month DESC").
			Find(&bills).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "家計簿一覧の取得に失敗しました"})
			return
		}

		// 各家計簿の金額情報を計算
		var billResponses []models.BillResponse
		for _, bill := range bills {
			totalAmount := 0.0
			for _, item := range bill.Items {
				totalAmount += item.Amount
			}

			billResponse := models.BillResponse{
				MonthlyBill: bill,
				TotalAmount: totalAmount,
			}
			billResponses = append(billResponses, billResponse)
		}

		c.JSON(http.StatusOK, gin.H{"bills": billResponses})
	}
}
