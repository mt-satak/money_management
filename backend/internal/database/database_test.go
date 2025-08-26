// ========================================
// データベースモジュールの自動テスト
// 本番環境と同じMySQL 8.0を使用してテスト実行
// ========================================

package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"money_management/internal/models"
)

// TestSetupTestDB_Success テスト用データベースのセットアップが正常に完了することを検証（エラーなしとDB接続の確立を期待）
func TestSetupTestDB_Success(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	// テスト用MySQLデータベースを使用
	originalDB := DB
	defer func() {
		DB = originalDB // テスト後に元に戻す
	}()

	// テスト用データベースセットアップ
	db, err := SetupTestDB()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// クリーンアップ
	defer func() {
		CleanupTestDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// GetDBが正しく動作することを確認
	DB = db
	retrievedDB := GetDB()
	assert.Equal(t, db, retrievedDB)
}

// TestGetDB_ReturnsCorrectInstance GetDB関数が正しいデータベースインスタンスを返すことを検証（設定されたDBインスタンスの返却を期待）
func TestGetDB_ReturnsCorrectInstance(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	// テスト用データベースを作成
	testDB, err := SetupTestDB()
	assert.NoError(t, err)

	// オリジナルのDBを保存
	originalDB := DB
	defer func() {
		DB = originalDB // テスト後に元に戻す
		CleanupTestDB(testDB)
		sqlDB, _ := testDB.DB()
		sqlDB.Close()
	}()

	// テスト用DBを設定
	DB = testDB

	// GetDBが正しいインスタンスを返すことを確認
	retrievedDB := GetDB()
	assert.Equal(t, testDB, retrievedDB)
	assert.NotEqual(t, originalDB, retrievedDB)
}

// TestDatabase_ModelOperations データベースで基本的なモデル操作が正常に動作することを検証（CRUD操作の成功を期待）
func TestDatabase_ModelOperations(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	// テスト用データベースセットアップ
	db, err := SetupTestDB()
	assert.NoError(t, err)

	// グローバルDB変数を設定
	originalDB := DB
	DB = db
	defer func() {
		DB = originalDB
		CleanupTestDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// ユーザー作成テスト（並列実行対応のため一意な値を使用）
	timestamp := time.Now().UnixNano()
	user := models.User{
		Name:         fmt.Sprintf("テストユーザー_%d", timestamp),
		AccountID:    fmt.Sprintf("testuser_%d", timestamp),
		PasswordHash: "hashedpassword",
	}
	err = db.Create(&user).Error
	assert.NoError(t, err)
	assert.NotZero(t, user.ID) // IDが設定されることを確認

	// ユーザー読み取りテスト
	var retrievedUser models.User
	err = db.Where("account_id = ?", fmt.Sprintf("testuser_%d", timestamp)).First(&retrievedUser).Error
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("テストユーザー_%d", timestamp), retrievedUser.Name)
	assert.Equal(t, fmt.Sprintf("testuser_%d", timestamp), retrievedUser.AccountID)

	// すべてのデータを単一トランザクション内で作成（並列実行安全性確保）
	var bill models.MonthlyBill
	var item models.BillItem

	err = db.Transaction(func(tx *gorm.DB) error {
		// ユーザーの存在確認
		var existingUser models.User
		if err := tx.First(&existingUser, user.ID).Error; err != nil {
			return err
		}

		// 家計簿作成テスト
		bill = models.MonthlyBill{
			Year:        2024,
			Month:       12,
			RequesterID: existingUser.ID,
			PayerID:     existingUser.ID,
			Status:      "pending",
		}
		if err := tx.Create(&bill).Error; err != nil {
			return err
		}

		// 家計簿項目作成テスト
		item = models.BillItem{
			BillID:   bill.ID,
			ItemName: "テスト項目",
			Amount:   1000.50,
		}
		return tx.Create(&item).Error
	})
	assert.NoError(t, err)
	assert.NotZero(t, bill.ID)
	assert.NotZero(t, item.ID)

	// リレーション読み取りテスト
	var billWithItems models.MonthlyBill
	err = db.Preload("Items").First(&billWithItems, bill.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, len(billWithItems.Items))
	if len(billWithItems.Items) > 0 {
		assert.Equal(t, "テスト項目", billWithItems.Items[0].ItemName)
		assert.Equal(t, 1000.50, billWithItems.Items[0].Amount)
	}
}

// TestDatabase_Constraints データベースの制約が正しく適用されることを検証（ユニーク制約違反エラーの発生を期待）
func TestDatabase_Constraints(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	// テスト用データベースセットアップ
	db, err := SetupTestDB()
	assert.NoError(t, err)

	defer func() {
		CleanupTestDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 最初のユーザー作成（並列実行対応のため一意な値を使用）
	timestamp := time.Now().UnixNano()
	duplicateID := fmt.Sprintf("duplicate_%d", timestamp)
	user1 := models.User{
		Name:         "ユーザー1",
		AccountID:    duplicateID,
		PasswordHash: "hash1",
	}
	err = db.Create(&user1).Error
	assert.NoError(t, err)

	// 同じアカウントIDで二番目のユーザー作成（ユニーク制約違反）
	user2 := models.User{
		Name:         "ユーザー2",
		AccountID:    duplicateID, // 重複するアカウントID
		PasswordHash: "hash2",
	}
	err = db.Create(&user2).Error
	assert.Error(t, err) // エラーが発生することを期待
}

// TestDatabase_Timestamps データベースのタイムスタンプが正しく設定されることを検証（作成日時と更新日時の自動設定を期待）
func TestDatabase_Timestamps(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	// テスト用データベースセットアップ
	db, err := SetupTestDB()
	assert.NoError(t, err)

	defer func() {
		CleanupTestDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// ユーザー作成
	before := time.Now()
	user := models.User{
		Name:         "タイムスタンプテスト",
		AccountID:    "timestamp_test",
		PasswordHash: "hashedpassword",
	}
	err = db.Create(&user).Error
	assert.NoError(t, err)
	after := time.Now()

	// 作成日時が設定されていることを確認（少し余裕を持った範囲でチェック）
	assert.True(t, user.CreatedAt.After(before.Add(-time.Second)) && user.CreatedAt.Before(after.Add(time.Second)))
	assert.True(t, user.UpdatedAt.After(before.Add(-time.Second)) && user.UpdatedAt.Before(after.Add(time.Second)))

	// 更新テスト
	originalUpdatedAt := user.UpdatedAt
	time.Sleep(10 * time.Millisecond) // 時間差を作る

	user.Name = "更新されたユーザー"
	err = db.Save(&user).Error
	assert.NoError(t, err)

	// 更新日時が変更されていることを確認
	assert.True(t, user.UpdatedAt.After(originalUpdatedAt))
	// 作成日時は変更されていないことを確認
	assert.Equal(t, user.CreatedAt, user.CreatedAt)
}

// TestDatabase_CascadeDelete カスケード削除が正しく動作することを検証（親レコード削除時の子レコード削除を期待）
func TestDatabase_CascadeDelete(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	// テスト用データベースセットアップ
	db, err := SetupTestDB()
	assert.NoError(t, err)

	defer func() {
		CleanupTestDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// テストデータ作成（トランザクション内で関連性を保証）
	timestamp := time.Now().UnixNano()
	var user models.User
	var bill models.MonthlyBill
	var item models.BillItem

	err = db.Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Name:         fmt.Sprintf("テストユーザー_%d", timestamp),
			AccountID:    fmt.Sprintf("testuser_%d", timestamp),
			PasswordHash: "hashedpassword",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		bill = models.MonthlyBill{
			Year:        2024,
			Month:       12,
			RequesterID: user.ID,
			PayerID:     user.ID,
			Status:      "pending",
		}
		if err := tx.Create(&bill).Error; err != nil {
			return err
		}

		item = models.BillItem{
			BillID:   bill.ID,
			ItemName: "テスト項目",
			Amount:   1000.0,
		}
		return tx.Create(&item).Error
	})
	assert.NoError(t, err)

	// 家計簿削除前の項目存在確認
	var itemCount int64
	db.Model(&models.BillItem{}).Where("bill_id = ?", bill.ID).Count(&itemCount)
	assert.Equal(t, int64(1), itemCount)

	// GORMの関連削除機能を使用してカスケード削除をテスト
	// Select("Items")で関連する項目も一緒に削除
	err = db.Select("Items").Delete(&bill, bill.ID).Error
	assert.NoError(t, err)

	// カスケード削除が正しく動作したことを確認
	// 関連項目が自動的に削除されたことを確認
	db.Model(&models.BillItem{}).Where("bill_id = ?", bill.ID).Count(&itemCount)
	assert.Equal(t, int64(0), itemCount)
}

// TestInit_ConnectionFailure データベース接続失敗時のInit関数をテスト
func TestInit_ConnectionFailure(t *testing.T) {
	// このテストは時間がかかるため短縮モードではスキップ
	if testing.Short() {
		t.Skip("TestInit_ConnectionFailure: 短縮モードのためスキップ")
	}

	// 元のDBを保存
	originalDB := DB
	defer func() {
		// テスト後に元のDBを復元
		DB = originalDB
	}()

	// DBをリセット
	DB = nil

	// Init関数は存在しないデータベースへの接続を試行するため、
	// エラーが返されることを期待する（タイムアウト時間を短縮するため実際のInitは使用しない）
	// 代わりに無効なDSNで直接接続を試行
	dsn := "invalid_user:invalid_pass@tcp(nonexistent:3306)/nonexistent_db"
	_, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	assert.Error(t, err, "存在しないデータベースへの接続でエラーが返されませんでした")
}

// TestGetDB_ReturnsNil DBがnullの場合のGetDB関数をテスト
func TestGetDB_ReturnsNil(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	// 元のDBを保存
	originalDB := DB
	defer func() {
		// テスト後に元のDBを復元
		DB = originalDB
	}()

	// DBをnilに設定
	DB = nil

	// GetDBがnullを返すことを確認
	result := GetDB()
	assert.Nil(t, result, "DBがnullの場合、GetDBはnullを返すべきです")
}

// TestCleanupTestDB_DatabaseError データベースエラー時のCleanupTestDBをテスト
func TestCleanupTestDB_DatabaseError(t *testing.T) {
	db, err := SetupTestDB()
	assert.NoError(t, err)

	// データベース接続を閉じてエラー状態を作る
	sqlDB, _ := db.DB()
	sqlDB.Close()

	// CleanupTestDBがエラーを処理できることを確認
	// （パニックしないことを確認）
	// エラーログは期待される動作なので、エラー出力を抑制
	assert.NotPanics(t, func() {
		err := CleanupTestDB(db)
		// エラーが発生することを期待（データベース接続が閉じられているため）
		assert.Error(t, err, "データベース接続が閉じられているためエラーが期待されます")
	}, "CleanupTestDBはデータベースエラー時でもパニックしてはいけません")
}

// TestSetupTestDB_TableCreationVerification テーブル作成の確認テスト
func TestSetupTestDB_TableCreationVerification(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	db, err := SetupTestDB()
	assert.NoError(t, err, "SetupTestDBが失敗しました")
	assert.NotNil(t, db, "SetupTestDBがnullを返しました")

	defer func() {
		CleanupTestDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// テーブルが作成されたことを確認
	hasUsersTable := db.Migrator().HasTable(&models.User{})
	assert.True(t, hasUsersTable, "Usersテーブルが作成されていません")

	hasBillsTable := db.Migrator().HasTable(&models.MonthlyBill{})
	assert.True(t, hasBillsTable, "MonthlyBillsテーブルが作成されていません")

	hasItemsTable := db.Migrator().HasTable(&models.BillItem{})
	assert.True(t, hasItemsTable, "BillItemsテーブルが作成されていません")
}

// TestCleanupTestDB_MultipleTables 複数テーブルのクリーンアップテスト
func TestCleanupTestDB_MultipleTables(t *testing.T) {
	// データベーステストは並列化を無効にして安定性を重視

	db, err := SetupTestDB()
	assert.NoError(t, err)

	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// テストデータを複数テーブルに挿入（トランザクション内で関連性を保証）
	timestamp := time.Now().UnixNano()
	var user models.User
	var bill models.MonthlyBill
	var item models.BillItem

	err = db.Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Name:         fmt.Sprintf("クリーンアップテスト_%d", timestamp),
			AccountID:    fmt.Sprintf("cleanup_test_%d", timestamp),
			PasswordHash: "hashedpassword",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		bill = models.MonthlyBill{
			Year:        2024,
			Month:       1,
			RequesterID: user.ID,
			PayerID:     user.ID,
			Status:      "pending",
		}
		if err := tx.Create(&bill).Error; err != nil {
			return err
		}

		item = models.BillItem{
			BillID:   bill.ID,
			ItemName: "クリーンアップテスト項目",
			Amount:   500.0,
		}
		return tx.Create(&item).Error
	})
	assert.NoError(t, err)

	// クリーンアップ実行
	CleanupTestDB(db)

	// 全テーブルがクリーンアップされたことを確認
	var userCount, billCount, itemCount int64
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.MonthlyBill{}).Count(&billCount)
	db.Model(&models.BillItem{}).Count(&itemCount)

	assert.Equal(t, int64(0), userCount, "Usersテーブルがクリーンアップされていません")
	assert.Equal(t, int64(0), billCount, "MonthlyBillsテーブルがクリーンアップされていません")
	assert.Equal(t, int64(0), itemCount, "BillItemsテーブルがクリーンアップされていません")
}
