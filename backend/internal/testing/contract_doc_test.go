// ========================================
// 契約テスト仕様書生成テスト
// OpenAPI/Markdownドキュメント生成の検証
// ========================================

package testing

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// OpenAPI仕様書生成テスト
// ========================================

// TestGenerateOpenAPISpec OpenAPI仕様書生成テスト
func TestGenerateOpenAPISpec(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	spec, err := GenerateOpenAPISpec()
	assert.NoError(t, err, "OpenAPI仕様書の生成に失敗しました")
	assert.NotNil(t, spec, "OpenAPI仕様書がnilです")

	// 基本情報の検証
	assert.Equal(t, "3.0.3", spec.OpenAPI, "OpenAPIバージョンが不正です")
	assert.Equal(t, "Money Management API", spec.Info.Title, "APIタイトルが不正です")
	assert.Equal(t, "1.0.0", spec.Info.Version, "APIバージョンが不正です")

	// サーバー情報の検証
	assert.Len(t, spec.Servers, 3, "サーバー数が期待値と異なります")
	assert.Contains(t, spec.Servers[0].URL, "api.moneymanagement.com", "本番サーバーURLが含まれていません")

	// パス情報の検証
	assert.NotEmpty(t, spec.Paths, "APIパスが定義されていません")

	// 認証エンドポイントの検証
	assert.Contains(t, spec.Paths, "/auth/login", "ログインエンドポイントが定義されていません")
	assert.Contains(t, spec.Paths, "/auth/register", "登録エンドポイントが定義されていません")

	// 家計簿エンドポイントの検証
	assert.Contains(t, spec.Paths, "/bills", "家計簿エンドポイントが定義されていません")

	// セキュリティスキームの検証
	assert.Contains(t, spec.Components.SecuritySchemes, "bearerAuth", "JWT認証が定義されていません")
}

// TestGenerateOpenAPIJSON OpenAPI JSON生成テスト
func TestGenerateOpenAPIJSON(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	jsonOutput, err := GenerateOpenAPIJSON()
	assert.NoError(t, err, "OpenAPI JSON生成に失敗しました")
	assert.NotEmpty(t, jsonOutput, "JSON出力が空です")

	// 有効なJSONかチェック
	var parsedJSON map[string]interface{}
	err = json.Unmarshal([]byte(jsonOutput), &parsedJSON)
	assert.NoError(t, err, "生成されたJSONが無効です")

	// 必須フィールドの検証
	assert.Contains(t, parsedJSON, "openapi", "OpenAPIフィールドが含まれていません")
	assert.Contains(t, parsedJSON, "info", "infoフィールドが含まれていません")
	assert.Contains(t, parsedJSON, "paths", "pathsフィールドが含まれていません")
	assert.Contains(t, parsedJSON, "components", "componentsフィールドが含まれていません")
}

// TestGenerateOpenAPIYAML OpenAPI YAML生成テスト
func TestGenerateOpenAPIYAML(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	yamlOutput, err := GenerateOpenAPIYAML()
	assert.NoError(t, err, "OpenAPI YAML生成に失敗しました")
	assert.NotEmpty(t, yamlOutput, "YAML出力が空です")

	// YAML形式の基本的な検証
	assert.Contains(t, yamlOutput, "openapi:", "openapi フィールドが含まれていません")
	assert.Contains(t, yamlOutput, "info:", "info フィールドが含まれていません")
	assert.Contains(t, yamlOutput, "paths:", "paths フィールドが含まれていません")
	assert.Contains(t, yamlOutput, "servers:", "servers フィールドが含まれていません")

	// API情報の検証
	assert.Contains(t, yamlOutput, "Money Management API", "APIタイトルが含まれていません")
	assert.Contains(t, yamlOutput, "1.0.0", "APIバージョンが含まれていません")
}

// ========================================
// Markdownドキュメント生成テスト
// ========================================

// TestGenerateMarkdownDoc Markdownドキュメント生成テスト
func TestGenerateMarkdownDoc(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	markdownOutput, err := GenerateMarkdownDoc()
	assert.NoError(t, err, "Markdownドキュメント生成に失敗しました")
	assert.NotEmpty(t, markdownOutput, "Markdown出力が空です")

	// ドキュメント構造の検証
	assert.Contains(t, markdownOutput, "# Money Management API 仕様書", "メインタイトルが含まれていません")
	assert.Contains(t, markdownOutput, "## 概要", "概要セクションが含まれていません")
	assert.Contains(t, markdownOutput, "## 認証", "認証セクションが含まれていません")
	assert.Contains(t, markdownOutput, "## エンドポイント一覧", "エンドポイント一覧が含まれていません")

	// 認証セクションの検証
	assert.Contains(t, markdownOutput, "### 認証", "認証サブセクションが含まれていません")
	assert.Contains(t, markdownOutput, "JWT", "JWT認証の説明が含まれていません")
	assert.Contains(t, markdownOutput, "Authorization: Bearer", "認証ヘッダーの例が含まれていません")

	// エンドポイントの検証
	assert.Contains(t, markdownOutput, "#### Login API", "ログインAPIが含まれていません")
	assert.Contains(t, markdownOutput, "POST", "POSTメソッドが含まれていません")
	assert.Contains(t, markdownOutput, "/auth/login", "ログインパスが含まれていません")

	// コードブロックの検証
	assert.Contains(t, markdownOutput, "```json", "JSONコードブロックが含まれていません")
	assert.Contains(t, markdownOutput, "```", "コードブロックの終了タグが含まれていません")

	// エラーレスポンスの検証
	assert.Contains(t, markdownOutput, "## エラーレスポンス", "エラーレスポンスセクションが含まれていません")
	assert.Contains(t, markdownOutput, "HTTPステータスコード", "ステータスコード表が含まれていません")

	// 契約テストの説明
	assert.Contains(t, markdownOutput, "## 契約テスト", "契約テストセクションが含まれていません")
	assert.Contains(t, markdownOutput, "Contract Testing", "契約テストの説明が含まれていません")
}

// ========================================
// スキーマ変換テスト
// ========================================

// TestConvertToOpenAPISchema スキーマ変換テスト
func TestConvertToOpenAPISchema(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	// テスト用スキーマ
	testSchema := map[string]interface{}{
		"id": FieldDefinition{
			Type:        "integer",
			Required:    true,
			Description: "ユーザーID",
		},
		"name": FieldDefinition{
			Type:        "string",
			Required:    true,
			MinLength:   &[]int{1}[0],
			MaxLength:   &[]int{50}[0],
			Description: "ユーザー名",
		},
		"email": FieldDefinition{
			Type:        "string",
			Required:    false,
			Format:      "email",
			Description: "メールアドレス",
		},
		"status": FieldDefinition{
			Type:        "string",
			Required:    true,
			Enum:        []string{"active", "inactive"},
			Description: "ユーザーステータス",
		},
	}

	// スキーマ変換を実行
	openAPISchema, err := convertToOpenAPISchema(testSchema)
	assert.NoError(t, err, "スキーマ変換に失敗しました")

	// 基本構造の検証
	assert.Equal(t, "object", openAPISchema.Type, "オブジェクト型が設定されていません")
	assert.Len(t, openAPISchema.Properties, 4, "プロパティ数が期待値と異なります")

	// 個別フィールドの検証
	idField := openAPISchema.Properties["id"]
	assert.Equal(t, "integer", idField.Type, "IDフィールドの型が不正です")
	assert.Equal(t, "ユーザーID", idField.Description, "IDフィールドの説明が不正です")

	nameField := openAPISchema.Properties["name"]
	assert.Equal(t, "string", nameField.Type, "nameフィールドの型が不正です")
	assert.Equal(t, 1, *nameField.MinLength, "nameフィールドの最小長が不正です")
	assert.Equal(t, 50, *nameField.MaxLength, "nameフィールドの最大長が不正です")

	emailField := openAPISchema.Properties["email"]
	assert.Equal(t, "email", emailField.Format, "emailフィールドのフォーマットが不正です")

	statusField := openAPISchema.Properties["status"]
	assert.Equal(t, []string{"active", "inactive"}, statusField.Enum, "statusフィールドの列挙値が不正です")

	// 必須フィールドの検証
	assert.Contains(t, openAPISchema.Required, "id", "IDが必須フィールドに含まれていません")
	assert.Contains(t, openAPISchema.Required, "name", "nameが必須フィールドに含まれていません")
	assert.Contains(t, openAPISchema.Required, "status", "statusが必須フィールドに含まれていません")
	assert.NotContains(t, openAPISchema.Required, "email", "emailが必須フィールドに含まれています")
}

// ========================================
// 型マッピングテスト
// ========================================

// TestMapTypeToOpenAPI 型マッピングテスト
func TestMapTypeToOpenAPI(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	testCases := []struct {
		contractType string
		expected     string
	}{
		{"string", "string"},
		{"integer", "integer"},
		{"number", "number"},
		{"float", "number"},
		{"boolean", "boolean"},
		{"datetime", "string"},
		{"unknown", "string"}, // デフォルト
	}

	for _, tc := range testCases {
		result := mapTypeToOpenAPI(tc.contractType)
		assert.Equal(t, tc.expected, result, "型マッピングが不正です: %s -> %s", tc.contractType, tc.expected)
	}
}

// TestGenerateExampleValue 例値生成テスト
func TestGenerateExampleValue(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	testCases := []struct {
		fieldDef FieldDefinition
		expected interface{}
	}{
		{
			FieldDefinition{Type: "string", Description: "ユーザー名"},
			"example_ユーザー名",
		},
		{
			FieldDefinition{Type: "integer"},
			1,
		},
		{
			FieldDefinition{Type: "number"},
			100.50,
		},
		{
			FieldDefinition{Type: "boolean"},
			true,
		},
		{
			FieldDefinition{Type: "datetime"},
			"2024-01-01T00:00:00Z",
		},
		{
			FieldDefinition{Type: "string", Format: "datetime"},
			"2024-01-01T00:00:00Z",
		},
	}

	for _, tc := range testCases {
		result := generateExampleValue(tc.fieldDef)
		assert.Equal(t, tc.expected, result, "例値生成が不正です: %+v", tc.fieldDef)
	}
}

// ========================================
// ユーティリティ関数テスト
// ========================================

// TestGetTagsFromPath パスからタグ生成テスト
func TestGetTagsFromPath(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	testCases := []struct {
		path     string
		expected []string
	}{
		{"/auth/login", []string{"認証"}},
		{"/auth/register", []string{"認証"}},
		{"/users", []string{"ユーザー管理"}},
		{"/bills", []string{"家計簿"}},
		{"/bills/123", []string{"家計簿"}},
		{"/other", []string{"その他"}},
	}

	for _, tc := range testCases {
		result := getTagsFromPath(tc.path)
		assert.Equal(t, tc.expected, result, "タグ生成が不正です: %s", tc.path)
	}
}

// TestNeedsAuth 認証要否チェックテスト
func TestNeedsAuth(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	testCases := []struct {
		path     string
		expected bool
	}{
		{"/auth/login", false},    // ログインは認証不要
		{"/auth/register", false}, // 登録は認証不要
		{"/auth/me", true},        // 現在ユーザー情報は認証必要
		{"/users", true},          // ユーザー一覧は認証必要
		{"/bills", true},          // 家計簿は認証必要
		{"/bills/123", true},      // 家計簿詳細は認証必要
	}

	for _, tc := range testCases {
		result := needsAuth(tc.path)
		assert.Equal(t, tc.expected, result, "認証要否チェックが不正です: %s", tc.path)
	}
}

// ========================================
// 統合テスト
// ========================================

// TestDocumentationIntegrity ドキュメント整合性テスト
func TestDocumentationIntegrity(t *testing.T) {
	// 契約テストドキュメント生成は並列化を無効にして安定性を重視

	// 契約とドキュメントの整合性を確認
	contracts := GetAPIContracts()

	// OpenAPI仕様書生成
	spec, err := GenerateOpenAPISpec()
	assert.NoError(t, err, "OpenAPI仕様書生成に失敗")

	// 各契約がOpenAPI仕様書に含まれていることを確認
	for contractName, contract := range contracts {
		assert.Contains(t, spec.Paths, contract.Path, "契約 %s のパスが仕様書に含まれていません", contractName)

		method := strings.ToLower(contract.Method)
		pathSpec := spec.Paths[contract.Path]
		assert.Contains(t, pathSpec, method, "契約 %s のメソッドが仕様書に含まれていません", contractName)
	}

	// Markdownドキュメント生成
	markdown, err := GenerateMarkdownDoc()
	assert.NoError(t, err, "Markdownドキュメント生成に失敗")

	// 各契約がMarkdownに含まれていることを確認
	for contractName, contract := range contracts {
		assert.Contains(t, markdown, contract.Name, "契約 %s がMarkdownに含まれていません", contractName)
		assert.Contains(t, markdown, contract.Path, "契約 %s のパスがMarkdownに含まれていません", contractName)
		assert.Contains(t, markdown, contract.Method, "契約 %s のメソッドがMarkdownに含まれていません", contractName)
	}

	t.Logf("ドキュメント整合性テスト完了: %d個の契約を検証", len(contracts))
}
