// ========================================
// 契約テスト仕様書自動生成
// API契約からOpenAPI/Swagger仕様書を生成
// ========================================

package testing

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ========================================
// OpenAPI仕様書生成
// ========================================

// OpenAPISpec OpenAPI 3.0仕様書構造体
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Servers    []OpenAPIServer        `json:"servers"`
	Paths      map[string]OpenAPIPath `json:"paths"`
	Components OpenAPIComponents      `json:"components"`
}

// OpenAPIInfo API基本情報
type OpenAPIInfo struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Contact     OpenAPIContact `json:"contact"`
	License     OpenAPILicense `json:"license"`
}

// OpenAPIContact 連絡先情報
type OpenAPIContact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

// OpenAPILicense ライセンス情報
type OpenAPILicense struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// OpenAPIServer サーバー情報
type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// OpenAPIPath パス定義
type OpenAPIPath map[string]OpenAPIOperation

// OpenAPIOperation オペレーション定義
type OpenAPIOperation struct {
	Summary     string                     `json:"summary"`
	Description string                     `json:"description"`
	Tags        []string                   `json:"tags"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

// OpenAPIRequestBody リクエストボディ定義
type OpenAPIRequestBody struct {
	Description string                    `json:"description"`
	Required    bool                      `json:"required"`
	Content     map[string]OpenAPIContent `json:"content"`
}

// OpenAPIResponse レスポンス定義
type OpenAPIResponse struct {
	Description string                    `json:"description"`
	Content     map[string]OpenAPIContent `json:"content,omitempty"`
}

// OpenAPIContent コンテンツ定義
type OpenAPIContent struct {
	Schema OpenAPISchema `json:"schema"`
}

// OpenAPISchema スキーマ定義
type OpenAPISchema struct {
	Type        string                   `json:"type,omitempty"`
	Format      string                   `json:"format,omitempty"`
	Properties  map[string]OpenAPISchema `json:"properties,omitempty"`
	Items       *OpenAPISchema           `json:"items,omitempty"`
	Required    []string                 `json:"required,omitempty"`
	Example     interface{}              `json:"example,omitempty"`
	Description string                   `json:"description,omitempty"`
	Enum        []string                 `json:"enum,omitempty"`
	MinLength   *int                     `json:"minLength,omitempty"`
	MaxLength   *int                     `json:"maxLength,omitempty"`
}

// OpenAPIComponents コンポーネント定義
type OpenAPIComponents struct {
	Schemas         map[string]OpenAPISchema `json:"schemas"`
	SecuritySchemes map[string]interface{}   `json:"securitySchemes"`
}

// ========================================
// 契約からOpenAPI仕様書生成
// ========================================

// GenerateOpenAPISpec 契約定義からOpenAPI仕様書を生成
func GenerateOpenAPISpec() (*OpenAPISpec, error) {
	contracts := GetAPIContracts()

	spec := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:       "Money Management API",
			Description: "家計簿管理システムのRESTful API仕様書\n\n# 概要\n本APIは家計簿の作成、編集、請求、支払い機能を提供します。\n\n# 認証\nJWT (JSON Web Token) ベースの認証を使用します。\n\n# 契約テスト\n本仕様書は契約テスト（Contract Testing）により自動生成され、実装との整合性が保証されています。",
			Version:     "1.0.0",
			Contact: OpenAPIContact{
				Name:  "Money Management Team",
				Email: "support@moneymanagement.com",
				URL:   "https://moneymanagement.com/contact",
			},
			License: OpenAPILicense{
				Name: "MIT License",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []OpenAPIServer{
			{
				URL:         "https://api.moneymanagement.com/v1",
				Description: "本番環境",
			},
			{
				URL:         "https://staging-api.moneymanagement.com/v1",
				Description: "ステージング環境",
			},
			{
				URL:         "http://localhost:8080",
				Description: "開発環境",
			},
		},
		Paths: make(map[string]OpenAPIPath),
		Components: OpenAPIComponents{
			Schemas: make(map[string]OpenAPISchema),
			SecuritySchemes: map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
	}

	// 各契約をOpenAPIパスに変換
	for _, contract := range contracts {
		err := addContractToSpec(spec, contract)
		if err != nil {
			return nil, fmt.Errorf("契約 %s の変換に失敗: %v", contract.Name, err)
		}
	}

	// 共通スキーマを追加
	addCommonSchemas(spec)

	return spec, nil
}

// addContractToSpec 契約をOpenAPI仕様に追加
func addContractToSpec(spec *OpenAPISpec, contract Contract) error {
	method := strings.ToLower(contract.Method)
	path := contract.Path

	// パスが存在しない場合は作成
	if spec.Paths[path] == nil {
		spec.Paths[path] = make(OpenAPIPath)
	}

	// オペレーションを作成
	operation := OpenAPIOperation{
		Summary:     contract.Name,
		Description: contract.Description,
		Tags:        getTagsFromPath(path),
		Responses:   make(map[string]OpenAPIResponse),
	}

	// リクエストボディがある場合は追加
	if len(contract.RequestSchema) > 0 {
		requestBody, err := createRequestBody(contract.RequestSchema)
		if err != nil {
			return err
		}
		operation.RequestBody = requestBody
	}

	// レスポンスを追加
	responseSchema, err := createResponseSchema(contract.ResponseSchema)
	if err != nil {
		return err
	}

	operation.Responses[fmt.Sprintf("%d", contract.StatusCode)] = OpenAPIResponse{
		Description: "成功レスポンス",
		Content: map[string]OpenAPIContent{
			"application/json": {
				Schema: responseSchema,
			},
		},
	}

	// エラーレスポンスを追加
	operation.Responses["400"] = OpenAPIResponse{
		Description: "リクエストエラー",
		Content: map[string]OpenAPIContent{
			"application/json": {
				Schema: OpenAPISchema{
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"error": {
							Type:        "string",
							Description: "エラーメッセージ",
							Example:     "リクエストの形式が不正です",
						},
					},
					Required: []string{"error"},
				},
			},
		},
	}

	// 認証が必要なエンドポイントにはセキュリティを追加
	if needsAuth(path) {
		operation.Security = []map[string][]string{
			{"bearerAuth": {}},
		}

		// 401エラーを追加
		operation.Responses["401"] = OpenAPIResponse{
			Description: "認証エラー",
			Content: map[string]OpenAPIContent{
				"application/json": {
					Schema: OpenAPISchema{
						Type: "object",
						Properties: map[string]OpenAPISchema{
							"error": {
								Type:        "string",
								Description: "認証エラーメッセージ",
								Example:     "認証情報が無効です",
							},
						},
						Required: []string{"error"},
					},
				},
			},
		}
	}

	spec.Paths[path][method] = operation

	return nil
}

// createRequestBody リクエストボディを作成
func createRequestBody(schema map[string]interface{}) (*OpenAPIRequestBody, error) {
	openAPISchema, err := convertToOpenAPISchema(schema)
	if err != nil {
		return nil, err
	}

	return &OpenAPIRequestBody{
		Description: "リクエストボディ",
		Required:    true,
		Content: map[string]OpenAPIContent{
			"application/json": {
				Schema: openAPISchema,
			},
		},
	}, nil
}

// createResponseSchema レスポンススキーマを作成
func createResponseSchema(schema map[string]interface{}) (OpenAPISchema, error) {
	return convertToOpenAPISchema(schema)
}

// convertToOpenAPISchema 契約スキーマをOpenAPIスキーマに変換
func convertToOpenAPISchema(schema map[string]interface{}) (OpenAPISchema, error) {
	openAPISchema := OpenAPISchema{
		Type:       "object",
		Properties: make(map[string]OpenAPISchema),
	}

	var required []string

	for fieldName, fieldDef := range schema {
		// フィールド定義をパース
		defBytes, _ := json.Marshal(fieldDef)
		var fieldDefinition FieldDefinition
		if err := json.Unmarshal(defBytes, &fieldDefinition); err != nil {
			// ネストされたオブジェクトの場合
			if nestedMap, ok := fieldDef.(map[string]interface{}); ok {
				nestedSchema, err := convertToOpenAPISchema(nestedMap)
				if err != nil {
					return openAPISchema, err
				}
				openAPISchema.Properties[fieldName] = nestedSchema
			}
			continue
		}

		// フィールドスキーマを作成
		fieldSchema := OpenAPISchema{
			Type:        mapTypeToOpenAPI(fieldDefinition.Type),
			Format:      fieldDefinition.Format,
			Description: fieldDefinition.Description,
			Enum:        fieldDefinition.Enum,
			MinLength:   fieldDefinition.MinLength,
			MaxLength:   fieldDefinition.MaxLength,
		}

		// 例値を設定
		if fieldDefinition.Default != nil {
			fieldSchema.Example = fieldDefinition.Default
		} else {
			fieldSchema.Example = generateExampleValue(fieldDefinition)
		}

		openAPISchema.Properties[fieldName] = fieldSchema

		// 必須フィールドを追加
		if fieldDefinition.Required {
			required = append(required, fieldName)
		}
	}

	openAPISchema.Required = required
	return openAPISchema, nil
}

// mapTypeToOpenAPI 契約テストの型をOpenAPI型にマッピング
func mapTypeToOpenAPI(contractType string) string {
	switch contractType {
	case "datetime":
		return "string"
	case "integer":
		return "integer"
	case "number", "float":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}

// generateExampleValue 型に基づいて例値を生成
func generateExampleValue(def FieldDefinition) interface{} {
	switch def.Type {
	case "string":
		if def.Format == "datetime" {
			return "2024-01-01T00:00:00Z"
		}
		return "example_" + strings.ToLower(def.Description)
	case "integer":
		return 1
	case "number", "float":
		return 100.50
	case "boolean":
		return true
	case "datetime":
		return "2024-01-01T00:00:00Z"
	default:
		return "example_value"
	}
}

// getTagsFromPath パスからタグを生成
func getTagsFromPath(path string) []string {
	if strings.Contains(path, "/auth") {
		return []string{"認証"}
	}
	if strings.Contains(path, "/users") {
		return []string{"ユーザー管理"}
	}
	if strings.Contains(path, "/bills") {
		return []string{"家計簿"}
	}
	return []string{"その他"}
}

// needsAuth 認証が必要かチェック
func needsAuth(path string) bool {
	// ログインと登録以外は認証が必要
	return !strings.Contains(path, "/login") && !strings.Contains(path, "/register")
}

// addCommonSchemas 共通スキーマを追加
func addCommonSchemas(spec *OpenAPISpec) {
	// エラーレスポンススキーマ
	spec.Components.Schemas["Error"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"error": {
				Type:        "string",
				Description: "エラーメッセージ",
			},
		},
		Required: []string{"error"},
	}

	// ユーザースキーマ
	spec.Components.Schemas["User"] = OpenAPISchema{
		Type: "object",
		Properties: map[string]OpenAPISchema{
			"id": {
				Type:        "integer",
				Description: "ユーザーID",
				Example:     1,
			},
			"name": {
				Type:        "string",
				Description: "ユーザー名",
				Example:     "山田太郎",
			},
			"account_id": {
				Type:        "string",
				Description: "アカウントID",
				Example:     "yamada_taro",
			},
			"created_at": {
				Type:        "string",
				Format:      "date-time",
				Description: "作成日時",
				Example:     "2024-01-01T00:00:00Z",
			},
			"updated_at": {
				Type:        "string",
				Format:      "date-time",
				Description: "更新日時",
				Example:     "2024-01-01T00:00:00Z",
			},
		},
		Required: []string{"id", "name", "account_id", "created_at", "updated_at"},
	}
}

// ========================================
// 仕様書出力
// ========================================

// GenerateOpenAPIJSON OpenAPI仕様をJSON形式で出力
func GenerateOpenAPIJSON() (string, error) {
	spec, err := GenerateOpenAPISpec()
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// GenerateOpenAPIYAML OpenAPI仕様をYAML形式で出力（基本形式）
func GenerateOpenAPIYAML() (string, error) {
	spec, err := GenerateOpenAPISpec()
	if err != nil {
		return "", err
	}

	// 基本的なYAML出力（フル実装にはyamlライブラリが必要）
	yaml := fmt.Sprintf(`openapi: "%s"
info:
  title: "%s"
  description: "%s"
  version: "%s"
  contact:
    name: "%s"
    email: "%s"
    url: "%s"
  license:
    name: "%s"
    url: "%s"

servers:`,
		spec.OpenAPI,
		spec.Info.Title,
		spec.Info.Description,
		spec.Info.Version,
		spec.Info.Contact.Name,
		spec.Info.Contact.Email,
		spec.Info.Contact.URL,
		spec.Info.License.Name,
		spec.Info.License.URL,
	)

	for _, server := range spec.Servers {
		yaml += fmt.Sprintf(`
  - url: "%s"
    description: "%s"`, server.URL, server.Description)
	}

	yaml += "\n\npaths:\n"

	for path, operations := range spec.Paths {
		yaml += fmt.Sprintf("  %s:\n", path)
		for method, operation := range operations {
			yaml += fmt.Sprintf("    %s:\n", method)
			yaml += fmt.Sprintf("      summary: \"%s\"\n", operation.Summary)
			yaml += fmt.Sprintf("      description: \"%s\"\n", operation.Description)
			yaml += "      tags:\n"
			for _, tag := range operation.Tags {
				yaml += fmt.Sprintf("        - \"%s\"\n", tag)
			}
		}
	}

	return yaml, nil
}

// GenerateMarkdownDoc API仕様をMarkdown形式で出力
func GenerateMarkdownDoc() (string, error) {
	contracts := GetAPIContracts()

	markdown := `# Money Management API 仕様書

**バージョン**: 1.0.0
**生成日時**: ` + time.Now().Format("2006-01-02 15:04:05") + `
**生成方法**: 契約テスト（Contract Testing）による自動生成

## 概要

Money Management APIは家計簿管理システムのRESTful APIです。ユーザー認証、家計簿の作成・編集・請求・支払い機能を提供します。

## 認証

JWT (JSON Web Token) ベースの認証を使用します。認証が必要なエンドポイントでは、リクエストヘッダーに以下を含めてください：

` + "```" + `
Authorization: Bearer <JWT_TOKEN>
` + "```" + `

## エンドポイント一覧

`

	// 認証エンドポイント
	markdown += "\n### 認証\n\n"
	authContracts := []string{"login", "register", "get_me"}
	for _, contractName := range authContracts {
		if contract, exists := contracts[contractName]; exists {
			markdown += generateEndpointDoc(contract)
		}
	}

	// ユーザー管理エンドポイント
	markdown += "\n### ユーザー管理\n\n"
	userContracts := []string{"get_users"}
	for _, contractName := range userContracts {
		if contract, exists := contracts[contractName]; exists {
			markdown += generateEndpointDoc(contract)
		}
	}

	// 家計簿エンドポイント
	markdown += "\n### 家計簿\n\n"
	billContracts := []string{"create_bill", "get_bill", "get_bills"}
	for _, contractName := range billContracts {
		if contract, exists := contracts[contractName]; exists {
			markdown += generateEndpointDoc(contract)
		}
	}

	// エラーレスポンス
	markdown += `
## エラーレスポンス

すべてのエラーは以下の形式で返されます：

` + "```json" + `
{
  "error": "エラーメッセージ"
}
` + "```" + `

### HTTPステータスコード

| ステータスコード | 説明 |
|-----------------|------|
| 200 | 成功 |
| 201 | 作成成功 |
| 400 | リクエストエラー |
| 401 | 認証エラー |
| 404 | リソースが見つからない |
| 409 | 競合エラー |
| 500 | サーバーエラー |

## 契約テスト

本API仕様書は契約テスト（Contract Testing）により自動生成されており、実装との整合性が保証されています。

- **テスト実行**: ` + "`go test ./internal/testing`" + `
- **カバレッジ**: ` + fmt.Sprintf("%d個のAPI契約", len(contracts)) + `
- **最終更新**: ` + time.Now().Format("2006-01-02") + `
`

	return markdown, nil
}

// generateEndpointDoc エンドポイントのMarkdownドキュメントを生成
func generateEndpointDoc(contract Contract) string {
	doc := fmt.Sprintf("#### %s\n\n", contract.Name)
	doc += fmt.Sprintf("**%s** `%s`\n\n", contract.Method, contract.Path)
	doc += fmt.Sprintf("%s\n\n", contract.Description)

	// リクエスト例
	if len(contract.RequestSchema) > 0 {
		doc += "**リクエスト例:**\n\n"
		doc += "```json\n"
		doc += generateJSONExample(contract.RequestSchema)
		doc += "\n```\n\n"
	}

	// レスポンス例
	doc += fmt.Sprintf("**レスポンス例** (HTTP %d):\n\n", contract.StatusCode)
	doc += "```json\n"
	doc += generateJSONExample(contract.ResponseSchema)
	doc += "\n```\n\n"

	return doc
}

// generateJSONExample スキーマからJSON例を生成
func generateJSONExample(schema map[string]interface{}) string {
	example := make(map[string]interface{})

	for fieldName, fieldDef := range schema {
		defBytes, _ := json.Marshal(fieldDef)
		var fieldDefinition FieldDefinition
		if err := json.Unmarshal(defBytes, &fieldDefinition); err == nil {
			example[fieldName] = generateExampleValue(fieldDefinition)
		}
	}

	jsonBytes, _ := json.MarshalIndent(example, "", "  ")
	return string(jsonBytes)
}
