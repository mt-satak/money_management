// ========================================
// 契約テスト（Contract Testing）フレームワーク
// API間の整合性保証とスキーマ検証
// ========================================

package testing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ========================================
// 契約テスト用インターフェース定義
// ========================================

// ContractVerifier 契約検証インターフェース
type ContractVerifier interface {
	ValidateResponseSchema(response interface{}, expectedContract Contract) error
	ValidateRequestSchema(request interface{}, expectedContract Contract) error
	ValidateStatusCode(statusCode int, expectedStatusCode int) error
}

// Contract API契約定義
type Contract struct {
	Name           string                 `json:"name"`
	Method         string                 `json:"method"`
	Path           string                 `json:"path"`
	RequestSchema  map[string]interface{} `json:"request_schema"`
	ResponseSchema map[string]interface{} `json:"response_schema"`
	StatusCode     int                    `json:"status_code"`
	Headers        map[string]string      `json:"headers"`
	Description    string                 `json:"description"`
}

// FieldDefinition フィールド定義
type FieldDefinition struct {
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Format      string      `json:"format,omitempty"`
	MinLength   *int        `json:"min_length,omitempty"`
	MaxLength   *int        `json:"max_length,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// ValidationError 検証エラー
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("契約違反: フィールド '%s' - %s (値: %v)", e.Field, e.Message, e.Value)
}

// ValidationResult 検証結果
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}

// ========================================
// 契約検証実装
// ========================================

// StandardContractVerifier 標準契約検証実装
type StandardContractVerifier struct{}

// NewContractVerifier 契約検証インスタンスの作成
func NewContractVerifier() ContractVerifier {
	return &StandardContractVerifier{}
}

// ValidateResponseSchema レスポンススキーマの検証
func (v *StandardContractVerifier) ValidateResponseSchema(response interface{}, contract Contract) error {
	return v.validateSchema(response, contract.ResponseSchema, "response")
}

// ValidateRequestSchema リクエストスキーマの検証
func (v *StandardContractVerifier) ValidateRequestSchema(request interface{}, contract Contract) error {
	return v.validateSchema(request, contract.RequestSchema, "request")
}

// ValidateStatusCode ステータスコードの検証
func (v *StandardContractVerifier) ValidateStatusCode(statusCode int, expectedStatusCode int) error {
	if statusCode != expectedStatusCode {
		return ValidationError{
			Field:   "status_code",
			Message: fmt.Sprintf("期待値: %d, 実際: %d", expectedStatusCode, statusCode),
			Value:   statusCode,
		}
	}
	return nil
}

// validateSchema 汎用スキーマ検証
func (v *StandardContractVerifier) validateSchema(data interface{}, schema map[string]interface{}, context string) error {
	var errors []ValidationError

	// JSONにシリアライズしてから再度パース（構造の標準化）
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ValidationError{
			Field:   context,
			Message: "JSONシリアライズに失敗",
			Value:   data,
		}
	}

	var parsedData map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsedData); err != nil {
		return ValidationError{
			Field:   context,
			Message: "JSONパースに失敗",
			Value:   string(jsonData),
		}
	}

	// スキーマの各フィールドを検証
	for fieldName, fieldDef := range schema {
		if err := v.validateField(fieldName, parsedData[fieldName], fieldDef, parsedData); err != nil {
			errors = append(errors, err...)
		}
	}

	if len(errors) > 0 {
		return &MultiValidationError{Errors: errors}
	}

	return nil
}

// validateField 個別フィールドの検証
func (v *StandardContractVerifier) validateField(fieldName string, value interface{}, definition interface{}, context map[string]interface{}) []ValidationError {
	var errors []ValidationError

	// フィールド定義をパース
	defBytes, _ := json.Marshal(definition)
	var fieldDef FieldDefinition
	json.Unmarshal(defBytes, &fieldDef)

	// 必須フィールドの検証
	if fieldDef.Required && value == nil {
		errors = append(errors, ValidationError{
			Field:   fieldName,
			Message: "必須フィールドが設定されていません",
			Value:   value,
		})
		return errors
	}

	// 値が存在する場合の型検証
	if value != nil {
		if err := v.validateType(fieldName, value, fieldDef); err != nil {
			errors = append(errors, *err)
		}

		if err := v.validateConstraints(fieldName, value, fieldDef); err != nil {
			errors = append(errors, err...)
		}
	}

	return errors
}

// validateType 型検証
func (v *StandardContractVerifier) validateType(fieldName string, value interface{}, definition FieldDefinition) *ValidationError {
	actualType := reflect.TypeOf(value).String()

	switch definition.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("型が不正です。期待値: string, 実際: %s", actualType),
				Value:   value,
			}
		}
	case "number", "float":
		if _, ok := value.(float64); !ok {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("型が不正です。期待値: number, 実際: %s", actualType),
				Value:   value,
			}
		}
	case "integer":
		switch v := value.(type) {
		case float64:
			if v != float64(int(v)) {
				return &ValidationError{
					Field:   fieldName,
					Message: "整数値が期待されています",
					Value:   value,
				}
			}
		default:
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("型が不正です。期待値: integer, 実際: %s", actualType),
				Value:   value,
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("型が不正です。期待値: boolean, 実際: %s", actualType),
				Value:   value,
			}
		}
	case "datetime":
		if str, ok := value.(string); ok {
			if _, err := time.Parse(time.RFC3339, str); err != nil {
				return &ValidationError{
					Field:   fieldName,
					Message: "日時形式が不正です（RFC3339形式が期待されます）",
					Value:   value,
				}
			}
		} else {
			return &ValidationError{
				Field:   fieldName,
				Message: "日時は文字列形式で指定してください",
				Value:   value,
			}
		}
	}

	return nil
}

// validateConstraints 制約検証
func (v *StandardContractVerifier) validateConstraints(fieldName string, value interface{}, definition FieldDefinition) []ValidationError {
	var errors []ValidationError

	// 文字列長制約
	if str, ok := value.(string); ok {
		if definition.MinLength != nil && len(str) < *definition.MinLength {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("文字列長が短すぎます。最小: %d, 実際: %d", *definition.MinLength, len(str)),
				Value:   value,
			})
		}

		if definition.MaxLength != nil && len(str) > *definition.MaxLength {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("文字列長が長すぎます。最大: %d, 実際: %d", *definition.MaxLength, len(str)),
				Value:   value,
			})
		}

		// 列挙値制約
		if len(definition.Enum) > 0 {
			found := false
			for _, enumValue := range definition.Enum {
				if str == enumValue {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("許可されていない値です。許可値: %v", definition.Enum),
					Value:   value,
				})
			}
		}
	}

	return errors
}

// MultiValidationError 複数の検証エラー
type MultiValidationError struct {
	Errors []ValidationError
}

func (e *MultiValidationError) Error() string {
	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// ========================================
// APIコントラクト定義
// ========================================

// GetAPIContracts すべてのAPI契約を取得
func GetAPIContracts() map[string]Contract {
	return map[string]Contract{
		"login":       GetLoginContract(),
		"register":    GetRegisterContract(),
		"get_me":      GetMeContract(),
		"get_users":   GetUsersContract(),
		"create_bill": GetCreateBillContract(),
		"get_bill":    GetBillContract(),
		"get_bills":   GetBillsContract(),
	}
}

// GetLoginContract ログインAPI契約
func GetLoginContract() Contract {
	return Contract{
		Name:   "Login API",
		Method: "POST",
		Path:   "/auth/login",
		RequestSchema: map[string]interface{}{
			"account_id": FieldDefinition{
				Type:        "string",
				Required:    true,
				MinLength:   &[]int{3}[0],
				MaxLength:   &[]int{20}[0],
				Description: "ログイン用アカウントID",
			},
			"password": FieldDefinition{
				Type:        "string",
				Required:    true,
				MinLength:   &[]int{6}[0],
				Description: "ログインパスワード",
			},
		},
		ResponseSchema: map[string]interface{}{
			"token": FieldDefinition{
				Type:        "string",
				Required:    true,
				Description: "JWTアクセストークン",
			},
			"user": map[string]interface{}{
				"id": FieldDefinition{
					Type:        "integer",
					Required:    true,
					Description: "ユーザーID",
				},
				"name": FieldDefinition{
					Type:        "string",
					Required:    true,
					Description: "ユーザー名",
				},
				"account_id": FieldDefinition{
					Type:        "string",
					Required:    true,
					Description: "アカウントID",
				},
				"created_at": FieldDefinition{
					Type:        "datetime",
					Required:    true,
					Description: "アカウント作成日時",
				},
				"updated_at": FieldDefinition{
					Type:        "datetime",
					Required:    true,
					Description: "アカウント更新日時",
				},
			},
		},
		StatusCode:  200,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Description: "ユーザーログインAPI",
	}
}

// GetRegisterContract ユーザー登録API契約
func GetRegisterContract() Contract {
	return Contract{
		Name:   "Register API",
		Method: "POST",
		Path:   "/auth/register",
		RequestSchema: map[string]interface{}{
			"name": FieldDefinition{
				Type:        "string",
				Required:    true,
				MinLength:   &[]int{1}[0],
				MaxLength:   &[]int{50}[0],
				Description: "ユーザー名",
			},
			"account_id": FieldDefinition{
				Type:        "string",
				Required:    true,
				MinLength:   &[]int{3}[0],
				MaxLength:   &[]int{20}[0],
				Description: "一意のアカウントID",
			},
			"password": FieldDefinition{
				Type:        "string",
				Required:    true,
				MinLength:   &[]int{6}[0],
				Description: "パスワード",
			},
		},
		ResponseSchema: map[string]interface{}{
			"token": FieldDefinition{
				Type:        "string",
				Required:    true,
				Description: "JWTアクセストークン",
			},
			"user": map[string]interface{}{
				"id": FieldDefinition{
					Type:        "integer",
					Required:    true,
					Description: "新規作成されたユーザーID",
				},
				"name": FieldDefinition{
					Type:        "string",
					Required:    true,
					Description: "ユーザー名",
				},
				"account_id": FieldDefinition{
					Type:        "string",
					Required:    true,
					Description: "アカウントID",
				},
				"created_at": FieldDefinition{
					Type:        "datetime",
					Required:    true,
					Description: "アカウント作成日時",
				},
				"updated_at": FieldDefinition{
					Type:        "datetime",
					Required:    true,
					Description: "アカウント更新日時",
				},
			},
		},
		StatusCode:  201,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Description: "新規ユーザー登録API",
	}
}

// GetMeContract 現在ユーザー情報取得API契約
func GetMeContract() Contract {
	return Contract{
		Name:          "Get Me API",
		Method:        "GET",
		Path:          "/auth/me",
		RequestSchema: map[string]interface{}{
			// リクエストボディなし（JWT認証のみ）
		},
		ResponseSchema: map[string]interface{}{
			"id": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "ユーザーID",
			},
			"name": FieldDefinition{
				Type:        "string",
				Required:    true,
				Description: "ユーザー名",
			},
			"account_id": FieldDefinition{
				Type:        "string",
				Required:    true,
				Description: "アカウントID",
			},
			"created_at": FieldDefinition{
				Type:        "datetime",
				Required:    true,
				Description: "アカウント作成日時",
			},
			"updated_at": FieldDefinition{
				Type:        "datetime",
				Required:    true,
				Description: "アカウント更新日時",
			},
		},
		StatusCode:  200,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Description: "認証済みユーザーの情報取得API",
	}
}

// GetUsersContract ユーザー一覧取得API契約
func GetUsersContract() Contract {
	return Contract{
		Name:          "Get Users API",
		Method:        "GET",
		Path:          "/users",
		RequestSchema: map[string]interface{}{
			// リクエストボディなし
		},
		ResponseSchema: map[string]interface{}{
			"users": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"id": FieldDefinition{
						Type:        "integer",
						Required:    true,
						Description: "ユーザーID",
					},
					"name": FieldDefinition{
						Type:        "string",
						Required:    true,
						Description: "ユーザー名",
					},
					"account_id": FieldDefinition{
						Type:        "string",
						Required:    true,
						Description: "アカウントID",
					},
					"created_at": FieldDefinition{
						Type:        "datetime",
						Required:    true,
						Description: "アカウント作成日時",
					},
					"updated_at": FieldDefinition{
						Type:        "datetime",
						Required:    true,
						Description: "アカウント更新日時",
					},
				},
			},
		},
		StatusCode:  200,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Description: "システム内全ユーザー一覧取得API",
	}
}

// GetCreateBillContract 家計簿作成API契約
func GetCreateBillContract() Contract {
	return Contract{
		Name:   "Create Bill API",
		Method: "POST",
		Path:   "/bills",
		RequestSchema: map[string]interface{}{
			"year": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "対象年",
			},
			"month": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "対象月（1-12）",
			},
			"payer_id": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "支払者のユーザーID",
			},
		},
		ResponseSchema: map[string]interface{}{
			"id": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "作成された家計簿ID",
			},
			"year": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "対象年",
			},
			"month": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "対象月",
			},
			"status": FieldDefinition{
				Type:        "string",
				Required:    true,
				Enum:        []string{"pending", "requested", "paid"},
				Description: "家計簿の状態",
			},
			"created_at": FieldDefinition{
				Type:        "datetime",
				Required:    true,
				Description: "作成日時",
			},
		},
		StatusCode:  201,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Description: "新規家計簿作成API",
	}
}

// GetBillContract 家計簿取得API契約
func GetBillContract() Contract {
	return Contract{
		Name:          "Get Bill API",
		Method:        "GET",
		Path:          "/bills/:id",
		RequestSchema: map[string]interface{}{
			// パスパラメータとして bill_id
		},
		ResponseSchema: map[string]interface{}{
			"id": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "家計簿ID",
			},
			"year": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "対象年",
			},
			"month": FieldDefinition{
				Type:        "integer",
				Required:    true,
				Description: "対象月",
			},
			"status": FieldDefinition{
				Type:        "string",
				Required:    true,
				Enum:        []string{"pending", "requested", "paid"},
				Description: "家計簿の状態",
			},
			"total_amount": FieldDefinition{
				Type:        "number",
				Required:    true,
				Description: "総金額",
			},
			"created_at": FieldDefinition{
				Type:        "datetime",
				Required:    true,
				Description: "作成日時",
			},
		},
		StatusCode:  200,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Description: "指定家計簿詳細取得API",
	}
}

// GetBillsContract 家計簿一覧取得API契約
func GetBillsContract() Contract {
	return Contract{
		Name:          "Get Bills API",
		Method:        "GET",
		Path:          "/bills",
		RequestSchema: map[string]interface{}{
			// クエリパラメータでフィルタリング可能
		},
		ResponseSchema: map[string]interface{}{
			"bills": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"id": FieldDefinition{
						Type:        "integer",
						Required:    true,
						Description: "家計簿ID",
					},
					"year": FieldDefinition{
						Type:        "integer",
						Required:    true,
						Description: "対象年",
					},
					"month": FieldDefinition{
						Type:        "integer",
						Required:    true,
						Description: "対象月",
					},
					"status": FieldDefinition{
						Type:        "string",
						Required:    true,
						Enum:        []string{"pending", "requested", "paid"},
						Description: "家計簿の状態",
					},
					"total_amount": FieldDefinition{
						Type:        "number",
						Required:    true,
						Description: "総金額",
					},
				},
			},
		},
		StatusCode:  200,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Description: "家計簿一覧取得API",
	}
}
