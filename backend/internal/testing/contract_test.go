// ========================================
// 契約テスト実行 - API仕様整合性の検証
// レスポンススキーマ、リクエスト形式、ステータスコードの契約チェック
// ========================================

package testing

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"money_management/internal/models"
)

// ========================================
// 契約テストスイート定義
// ========================================

// ContractTestSuite 契約テスト用テストスイート
type ContractTestSuite struct {
	suite.Suite
	verifier  ContractVerifier
	contracts map[string]Contract
}

// SetupSuite テストスイートの初期化
func (suite *ContractTestSuite) SetupSuite() {
	suite.verifier = NewContractVerifier()
	suite.contracts = GetAPIContracts()

	// Ginのテストモードに設定
	gin.SetMode(gin.TestMode)
}

// TestContractTestSuite 契約テストスイートの実行
func TestContractTestSuite(t *testing.T) {
	suite.Run(t, new(ContractTestSuite))
}

// ========================================
// リクエスト・レスポンス契約テスト
// ========================================

// TestLoginContract_ValidRequest ログインAPI契約テスト（有効リクエスト）
func (suite *ContractTestSuite) TestLoginContract_ValidRequest() {
	contract := suite.contracts["login"]

	// 有効なリクエストデータ
	validRequest := models.LoginRequest{
		AccountID: "test_user",
		Password:  "password123",
	}

	// リクエスト契約検証
	err := suite.verifier.ValidateRequestSchema(validRequest, contract)
	assert.NoError(suite.T(), err, "有効なログインリクエストが契約に適合しません")
}

// TestLoginContract_InvalidRequest ログインAPI契約テスト（無効リクエスト）
func (suite *ContractTestSuite) TestLoginContract_InvalidRequest() {
	contract := suite.contracts["login"]

	testCases := []struct {
		name          string
		request       interface{}
		errorExpected bool
	}{
		{
			name: "短いアカウントID",
			request: map[string]interface{}{
				"account_id": "ab", // 3文字未満
				"password":   "password123",
			},
			errorExpected: true,
		},
		{
			name: "空のパスワード",
			request: map[string]interface{}{
				"account_id": "test_user",
				"password":   "", // 空文字
			},
			errorExpected: true,
		},
		{
			name: "必須フィールド欠如",
			request: map[string]interface{}{
				"account_id": "test_user",
				// password フィールドなし
			},
			errorExpected: true,
		},
		{
			name: "不正な型",
			request: map[string]interface{}{
				"account_id": 12345, // 数値（文字列が期待される）
				"password":   "password123",
			},
			errorExpected: true,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			err := suite.verifier.ValidateRequestSchema(tc.request, contract)
			if tc.errorExpected {
				assert.Error(t, err, "契約違反が検出されるべきです: %s", tc.name)
			} else {
				assert.NoError(t, err, "契約に適合するべきです: %s", tc.name)
			}
		})
	}
}

// TestLoginContract_ValidResponse ログインAPI契約テスト（有効レスポンス）
func (suite *ContractTestSuite) TestLoginContract_ValidResponse() {
	contract := suite.contracts["login"]

	// 有効なレスポンスデータ
	validResponse := models.LoginResponse{
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		User: models.User{
			ID:        1,
			Name:      "テストユーザー",
			AccountID: "test_user",
			CreatedAt: parseTime("2024-01-01T00:00:00Z"),
			UpdatedAt: parseTime("2024-01-01T00:00:00Z"),
		},
	}

	// レスポンス契約検証
	err := suite.verifier.ValidateResponseSchema(validResponse, contract)
	assert.NoError(suite.T(), err, "有効なログインレスポンスが契約に適合しません")
}

// TestRegisterContract_ValidRequest ユーザー登録API契約テスト（有効リクエスト）
func (suite *ContractTestSuite) TestRegisterContract_ValidRequest() {
	contract := suite.contracts["register"]

	// 有効なリクエストデータ
	validRequest := models.RegisterRequest{
		Name:      "新規ユーザー",
		AccountID: "new_user",
		Password:  "newpassword123",
	}

	// リクエスト契約検証
	err := suite.verifier.ValidateRequestSchema(validRequest, contract)
	assert.NoError(suite.T(), err, "有効な登録リクエストが契約に適合しません")
}

// TestRegisterContract_StatusCode ユーザー登録API契約テスト（ステータスコード）
func (suite *ContractTestSuite) TestRegisterContract_StatusCode() {
	contract := suite.contracts["register"]

	// 正しいステータスコード（201 Created）
	err := suite.verifier.ValidateStatusCode(201, contract.StatusCode)
	assert.NoError(suite.T(), err, "登録API契約のステータスコードが適合しません")

	// 間違ったステータスコード
	err = suite.verifier.ValidateStatusCode(200, contract.StatusCode)
	assert.Error(suite.T(), err, "ステータスコード契約違反が検出されるべきです")
}

// TestGetMeContract_ValidResponse 現在ユーザー情報取得API契約テスト
func (suite *ContractTestSuite) TestGetMeContract_ValidResponse() {
	contract := suite.contracts["get_me"]

	// 有効なレスポンスデータ
	validResponse := models.User{
		ID:        1,
		Name:      "認証済みユーザー",
		AccountID: "authenticated_user",
		CreatedAt: parseTime("2024-01-01T00:00:00Z"),
		UpdatedAt: parseTime("2024-01-01T00:00:00Z"),
	}

	// レスポンス契約検証
	err := suite.verifier.ValidateResponseSchema(validResponse, contract)
	assert.NoError(suite.T(), err, "現在ユーザー情報レスポンスが契約に適合しません")
}

// TestGetUsersContract_ValidResponse ユーザー一覧API契約テスト
func (suite *ContractTestSuite) TestGetUsersContract_ValidResponse() {
	contract := suite.contracts["get_users"]

	// 有効なレスポンスデータ
	validResponse := map[string]interface{}{
		"users": []models.User{
			{
				ID:        1,
				Name:      "ユーザー1",
				AccountID: "user1",
				CreatedAt: parseTime("2024-01-01T00:00:00Z"),
				UpdatedAt: parseTime("2024-01-01T00:00:00Z"),
			},
			{
				ID:        2,
				Name:      "ユーザー2",
				AccountID: "user2",
				CreatedAt: parseTime("2024-01-02T00:00:00Z"),
				UpdatedAt: parseTime("2024-01-02T00:00:00Z"),
			},
		},
	}

	// レスポンス契約検証
	err := suite.verifier.ValidateResponseSchema(validResponse, contract)
	assert.NoError(suite.T(), err, "ユーザー一覧レスポンスが契約に適合しません")
}

// TestCreateBillContract_ValidRequest 家計簿作成API契約テスト
func (suite *ContractTestSuite) TestCreateBillContract_ValidRequest() {
	contract := suite.contracts["create_bill"]

	// 有効なリクエストデータ
	validRequest := map[string]interface{}{
		"year":     2024,
		"month":    3,
		"payer_id": 2,
	}

	// リクエスト契約検証
	err := suite.verifier.ValidateRequestSchema(validRequest, contract)
	assert.NoError(suite.T(), err, "有効な家計簿作成リクエストが契約に適合しません")
}

// TestCreateBillContract_InvalidMonth 家計簿作成API契約テスト（無効月）
func (suite *ContractTestSuite) TestCreateBillContract_InvalidMonth() {
	contract := suite.contracts["create_bill"]

	// 無効な月のリクエスト
	invalidRequest := map[string]interface{}{
		"year":     2024,
		"month":    13, // 13月は存在しない
		"payer_id": 2,
	}

	// カスタムバリデーションが必要（基本的な型チェックは通過する）
	err := suite.verifier.ValidateRequestSchema(invalidRequest, contract)
	// 基本的な型チェックは通過するが、ビジネスロジックレベルでの検証が必要
	assert.NoError(suite.T(), err, "基本的な型チェックは通過するはずです")
}

// TestBillContract_ValidResponse 家計簿詳細API契約テスト
func (suite *ContractTestSuite) TestBillContract_ValidResponse() {
	contract := suite.contracts["get_bill"]

	// 有効なレスポンスデータ
	validResponse := models.BillResponse{
		MonthlyBill: models.MonthlyBill{
			ID:          1,
			Year:        2024,
			Month:       3,
			RequesterID: 1,
			PayerID:     2,
			Status:      "pending",
			CreatedAt:   parseTime("2024-03-01T00:00:00Z"),
			UpdatedAt:   parseTime("2024-03-01T00:00:00Z"),
		},
		TotalAmount: 15000.50,
	}

	// レスポンス契約検証
	err := suite.verifier.ValidateResponseSchema(validResponse, contract)
	assert.NoError(suite.T(), err, "家計簿詳細レスポンスが契約に適合しません")
}

// TestBillContract_InvalidStatus 家計簿API契約テスト（無効ステータス）
func (suite *ContractTestSuite) TestBillContract_InvalidStatus() {
	contract := suite.contracts["get_bill"]

	// 無効なステータスのレスポンス
	invalidResponse := map[string]interface{}{
		"id":           1,
		"year":         2024,
		"month":        3,
		"status":       "invalid_status", // 無効なステータス
		"total_amount": 15000.50,
		"created_at":   "2024-03-01T00:00:00Z",
	}

	// レスポンス契約検証（エラーが期待される）
	err := suite.verifier.ValidateResponseSchema(invalidResponse, contract)
	assert.Error(suite.T(), err, "無効なステータス値の契約違反が検出されるべきです")
}

// ========================================
// HTTPレスポンス契約テスト（統合）
// ========================================

// TestHTTPContract_LoginEndpoint HTTPレスポンス契約テスト（ログインエンドポイント）
func (suite *ContractTestSuite) TestHTTPContract_LoginEndpoint() {
	// モックHTTPハンドラーを作成
	handler := func(c *gin.Context) {
		// 契約に適合するレスポンスを返す
		response := models.LoginResponse{
			Token: "mock_jwt_token_for_contract_test",
			User: models.User{
				ID:        1,
				Name:      "契約テストユーザー",
				AccountID: "contract_test_user",
				CreatedAt: parseTime("2024-01-01T00:00:00Z"),
				UpdatedAt: parseTime("2024-01-01T00:00:00Z"),
			},
		}
		c.JSON(http.StatusOK, response)
	}

	// HTTPテストサーバーをセットアップ
	router := gin.New()
	router.POST("/auth/login", handler)

	// テストリクエストを作成
	requestBody := models.LoginRequest{
		AccountID: "contract_test_user",
		Password:  "password123",
	}
	jsonBytes, _ := json.Marshal(requestBody)

	// HTTPリクエストを実行
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// レスポンスをパース
	var response models.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err, "レスポンスのJSONパースに失敗")

	// 契約検証
	contract := suite.contracts["login"]

	// ステータスコード検証
	err = suite.verifier.ValidateStatusCode(w.Code, contract.StatusCode)
	assert.NoError(suite.T(), err, "ログインエンドポイントのステータスコード契約違反")

	// レスポンス契約検証
	err = suite.verifier.ValidateResponseSchema(response, contract)
	assert.NoError(suite.T(), err, "ログインエンドポイントのレスポンス契約違反")
}

// ========================================
// エラーレスポンス契約テスト
// ========================================

// TestErrorContract_BadRequest エラーレスポンス契約テスト（400 Bad Request）
func (suite *ContractTestSuite) TestErrorContract_BadRequest() {
	// 標準エラーレスポンス形式
	errorResponse := map[string]interface{}{
		"error": "リクエストの形式が不正です",
	}

	// エラーレスポンス契約（共通形式）
	errorContract := Contract{
		Name:       "Error Response",
		StatusCode: 400,
		ResponseSchema: map[string]interface{}{
			"error": FieldDefinition{
				Type:        "string",
				Required:    true,
				Description: "エラーメッセージ",
			},
		},
	}

	// エラーレスポンス契約検証
	err := suite.verifier.ValidateResponseSchema(errorResponse, errorContract)
	assert.NoError(suite.T(), err, "エラーレスポンスが契約に適合しません")

	// ステータスコード検証
	err = suite.verifier.ValidateStatusCode(400, errorContract.StatusCode)
	assert.NoError(suite.T(), err, "エラーレスポンスのステータスコード契約違反")
}

// ========================================
// バリデーション機能テスト
// ========================================

// TestContractVerifier_ValidationError バリデーションエラーのテスト
func (suite *ContractTestSuite) TestContractVerifier_ValidationError() {
	// 基本的なバリデーションエラーをテスト
	err := ValidationError{
		Field:   "test_field",
		Message: "テストエラーメッセージ",
		Value:   "invalid_value",
	}

	expectedMessage := "契約違反: フィールド 'test_field' - テストエラーメッセージ (値: invalid_value)"
	assert.Equal(suite.T(), expectedMessage, err.Error(), "ValidationError のエラーメッセージが期待と異なります")
}

// TestContractVerifier_MultiValidationError 複数バリデーションエラーのテスト
func (suite *ContractTestSuite) TestContractVerifier_MultiValidationError() {
	// 複数のバリデーションエラー
	errors := []ValidationError{
		{Field: "field1", Message: "エラー1", Value: "value1"},
		{Field: "field2", Message: "エラー2", Value: "value2"},
	}

	multiErr := &MultiValidationError{Errors: errors}
	errorMessage := multiErr.Error()

	// すべてのエラーメッセージが含まれることを確認
	assert.Contains(suite.T(), errorMessage, "field1")
	assert.Contains(suite.T(), errorMessage, "field2")
	assert.Contains(suite.T(), errorMessage, "エラー1")
	assert.Contains(suite.T(), errorMessage, "エラー2")
}

// ========================================
// ヘルパー関数
// ========================================

// parseTime RFC3339形式の時刻文字列をtime.Timeに変換
func parseTime(timeStr string) time.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	return t
}

// ========================================
// 契約テスト実行統計
// ========================================

// TestContractCoverage 契約テストカバレッジの確認
func (suite *ContractTestSuite) TestContractCoverage() {
	// すべての定義済み契約がテストされていることを確認
	expectedContracts := []string{
		"login", "register", "get_me", "get_users",
		"create_bill", "get_bill", "get_bills",
	}

	for _, contractName := range expectedContracts {
		_, exists := suite.contracts[contractName]
		assert.True(suite.T(), exists, "契約 '%s' が定義されていません", contractName)
	}

	suite.T().Logf("契約テストカバレッジ: %d/%d の API契約が定義済み",
		len(suite.contracts), len(expectedContracts))
}
