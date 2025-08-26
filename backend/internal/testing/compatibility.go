// ========================================
// SQLite-MySQL互換性チェック機能
// 本番環境との差異を最小化する検証システム
// ========================================

package testing

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
	"money_management/internal/models"
)

// CompatibilityChecker SQLite-MySQL互換性チェッカー
type CompatibilityChecker struct {
	sqliteDB *gorm.DB
	mysqlDB  *gorm.DB
}

// NewCompatibilityChecker 互換性チェッカー作成
func NewCompatibilityChecker(sqliteDB, mysqlDB *gorm.DB) *CompatibilityChecker {
	return &CompatibilityChecker{
		sqliteDB: sqliteDB,
		mysqlDB:  mysqlDB,
	}
}

// CompatibilityResult 互換性チェック結果
type CompatibilityResult struct {
	TestCase     string   `json:"test_case"`
	SQLiteResult string   `json:"sqlite_result"`
	MySQLResult  string   `json:"mysql_result"`
	Compatible   bool     `json:"compatible"`
	Issues       []string `json:"issues"`
}

// CheckDataTypeCompatibility データ型互換性チェック
func (cc *CompatibilityChecker) CheckDataTypeCompatibility() []CompatibilityResult {
	results := []CompatibilityResult{}

	// 1. DECIMAL精度チェック
	results = append(results, cc.checkDecimalPrecision())

	// 2. ENUM型チェック
	results = append(results, cc.checkEnumHandling())

	// 3. 外部キー制約チェック
	results = append(results, cc.checkForeignKeyConstraints())

	// 4. 日付・時刻処理チェック
	results = append(results, cc.checkDateTimeHandling())

	return results
}

// checkDecimalPrecision DECIMAL精度チェック
func (cc *CompatibilityChecker) checkDecimalPrecision() CompatibilityResult {
	result := CompatibilityResult{
		TestCase: "DECIMAL精度チェック",
		Issues:   []string{},
	}

	// 高精度の金額でテスト
	testAmount := 999999.99

	// SQLiteテスト
	item := models.BillItem{ItemName: "精度テスト", Amount: testAmount}
	cc.sqliteDB.Create(&item)

	var sqliteItem models.BillItem
	cc.sqliteDB.First(&sqliteItem, item.ID)
	result.SQLiteResult = fmt.Sprintf("%.2f", sqliteItem.Amount)

	// MySQLテスト
	cc.mysqlDB.Create(&item)
	var mysqlItem models.BillItem
	cc.mysqlDB.First(&mysqlItem, item.ID)
	result.MySQLResult = fmt.Sprintf("%.2f", mysqlItem.Amount)

	// 互換性判定
	result.Compatible = result.SQLiteResult == result.MySQLResult
	if !result.Compatible {
		result.Issues = append(result.Issues, "DECIMAL精度にDBエンジン間で差異あり")
	}

	return result
}

// checkEnumHandling ENUM型処理チェック
func (cc *CompatibilityChecker) checkEnumHandling() CompatibilityResult {
	result := CompatibilityResult{
		TestCase: "ENUM型ハンドリング",
		Issues:   []string{},
	}

	// 不正なステータス値でのテスト
	invalidStatus := "invalid_status"

	// SQLiteでの動作確認
	sqliteErr := cc.sqliteDB.Exec("INSERT INTO monthly_bills (year, month, requester_id, payer_id, status) VALUES (2024, 1, 1, 1, ?)", invalidStatus).Error
	result.SQLiteResult = fmt.Sprintf("Error: %v", sqliteErr != nil)

	// MySQLでの動作確認
	mysqlErr := cc.mysqlDB.Exec("INSERT INTO monthly_bills (year, month, requester_id, payer_id, status) VALUES (2024, 1, 1, 1, ?)", invalidStatus).Error
	result.MySQLResult = fmt.Sprintf("Error: %v", mysqlErr != nil)

	// 互換性判定
	result.Compatible = (sqliteErr != nil) == (mysqlErr != nil)
	if !result.Compatible {
		result.Issues = append(result.Issues, "ENUM制約のエラーハンドリングが異なる")
	}

	return result
}

// checkForeignKeyConstraints 外部キー制約チェック
func (cc *CompatibilityChecker) checkForeignKeyConstraints() CompatibilityResult {
	result := CompatibilityResult{
		TestCase: "外部キー制約",
		Issues:   []string{},
	}

	// 存在しないユーザーIDで家計簿作成テスト
	nonExistentUserID := uint(99999)

	// SQLiteテスト
	sqliteErr := cc.sqliteDB.Exec("INSERT INTO monthly_bills (year, month, requester_id, payer_id, status) VALUES (2024, 1, ?, ?, 'pending')",
		nonExistentUserID, nonExistentUserID).Error
	result.SQLiteResult = fmt.Sprintf("FK Error: %v", sqliteErr != nil)

	// MySQLテスト
	mysqlErr := cc.mysqlDB.Exec("INSERT INTO monthly_bills (year, month, requester_id, payer_id, status) VALUES (2024, 1, ?, ?, 'pending')",
		nonExistentUserID, nonExistentUserID).Error
	result.MySQLResult = fmt.Sprintf("FK Error: %v", mysqlErr != nil)

	result.Compatible = (sqliteErr != nil) == (mysqlErr != nil)
	if !result.Compatible {
		result.Issues = append(result.Issues, "外部キー制約の動作が異なる")
	}

	return result
}

// checkDateTimeHandling 日付・時刻処理チェック
func (cc *CompatibilityChecker) checkDateTimeHandling() CompatibilityResult {
	result := CompatibilityResult{
		TestCase: "日付・時刻処理",
		Issues:   []string{},
	}

	// 現在時刻での比較テスト
	query := "SELECT datetime('now') as current_time"

	var sqliteTime string
	cc.sqliteDB.Raw(query).Scan(&sqliteTime)
	result.SQLiteResult = sqliteTime

	// MySQL版に調整
	mysqlQuery := "SELECT NOW() as current_time"
	var mysqlTime string
	cc.mysqlDB.Raw(mysqlQuery).Scan(&mysqlTime)
	result.MySQLResult = mysqlTime

	// フォーマットが異なることを前提として警告のみ
	result.Compatible = true
	result.Issues = append(result.Issues, "日付フォーマットがDBエンジン間で異なる（警告）")

	return result
}

// RunCompatibilityTest 包括的互換性テスト実行
func (cc *CompatibilityChecker) RunCompatibilityTest() {
	log.Printf("🔍 SQLite-MySQL互換性チェック開始")

	results := cc.CheckDataTypeCompatibility()

	var compatibleCount, totalCount int
	for _, result := range results {
		totalCount++
		if result.Compatible {
			compatibleCount++
		}

		status := "✅"
		if !result.Compatible {
			status = "⚠️"
		}

		log.Printf("%s [%s]", status, result.TestCase)
		log.Printf("   SQLite: %s", result.SQLiteResult)
		log.Printf("   MySQL:  %s", result.MySQLResult)

		for _, issue := range result.Issues {
			log.Printf("   ⚠️ 課題: %s", issue)
		}
	}

	compatibilityRate := float64(compatibleCount) / float64(totalCount) * 100
	log.Printf("📊 互換性レート: %.1f%% (%d/%d)", compatibilityRate, compatibleCount, totalCount)

	if compatibilityRate < 80 {
		log.Printf("⚠️ 互換性が低いため、重要なテストはMySQLでも実行することを推奨")
	}
}

// DatabaseFeatureMatrix DB機能対応マトリックス
type DatabaseFeatureMatrix struct {
	Features map[string]DatabaseSupport `json:"features"`
}

type DatabaseSupport struct {
	MySQL  bool   `json:"mysql"`
	SQLite bool   `json:"sqlite"`
	Notes  string `json:"notes"`
}

// GetFeatureMatrix サポート機能マトリックス取得
func GetFeatureMatrix() DatabaseFeatureMatrix {
	return DatabaseFeatureMatrix{
		Features: map[string]DatabaseSupport{
			"ENUM型": {
				MySQL:  true,
				SQLite: false,
				Notes:  "SQLiteはTEXT+CHECK制約で代替",
			},
			"JSON型": {
				MySQL:  true,
				SQLite: true,
				Notes:  "SQLite 3.38+でサポート",
			},
			"全文検索": {
				MySQL:  true,
				SQLite: true,
				Notes:  "実装方法が大きく異なる",
			},
			"トランザクション分離レベル": {
				MySQL:  true,
				SQLite: false,
				Notes:  "SQLiteはSERIALIZABLEのみ",
			},
			"デッドロック検出": {
				MySQL:  true,
				SQLite: false,
				Notes:  "SQLiteは軽量ロック機構",
			},
			"外部キー制約": {
				MySQL:  true,
				SQLite: true,
				Notes:  "SQLiteはデフォルト無効",
			},
		},
	}
}

// RecommendTestStrategy テスト戦略推奨
func RecommendTestStrategy(testType string) string {
	strategies := map[string]string{
		"unit":        "SQLite推奨 - 高速実行でフィードバックループ短縮",
		"integration": "MySQL推奨 - 本番環境と同等の動作確認",
		"performance": "両方実行 - 相対性能と絶対性能の両面評価",
		"regression":  "MySQL必須 - 本番環境での回帰検証",
		"ci":          "段階実行 - SQLite→MySQL の順で効率的検証",
	}

	if strategy, exists := strategies[strings.ToLower(testType)]; exists {
		return strategy
	}

	return "不明なテストタイプ - integration テストとして MySQL 使用を推奨"
}
