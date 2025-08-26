// ========================================
// 認証サービステスト - モック/スタブパターンの実証
// 外部依存性を完全に分離した高速・確実なユニットテスト
// ========================================

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testmocks "money_management/internal/testing"
)

// TestAuthService_Login_Success ログイン成功のテスト（モック使用）
func TestAuthService_Login_Success(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// テストデータをモックDBに設定
	testUser, err := deps.SetupTestUserWithMock("test_user", "password123")
	assert.NoError(t, err)

	// ログイン実行
	result, err := authService.Login("test_user", "password123")

	// 結果検証
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testUser.ID, result.User.ID)
	assert.Equal(t, "test_user", result.User.AccountID)
	assert.NotEmpty(t, result.Token)

	// モックが生成したトークンの形式をチェック
	expectedToken := "mock_token_" + string(rune(testUser.ID+'0'))
	assert.Equal(t, expectedToken, result.Token)
}

// TestAuthService_Login_InvalidAccount 不正なアカウントIDでのログインテスト
func TestAuthService_Login_InvalidAccount(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// 存在しないアカウントでログイン試行
	result, err := authService.Login("nonexistent", "password123")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "認証情報が無効です", err.Error())
}

// TestAuthService_Login_WrongPassword 間違ったパスワードでのログインテスト
func TestAuthService_Login_WrongPassword(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// テストユーザーをセットアップ
	_, err := deps.SetupTestUserWithMock("test_user", "correct_password")
	assert.NoError(t, err)

	// 間違ったパスワードでログイン試行
	result, err := authService.Login("test_user", "wrong_password")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "認証情報が無効です", err.Error())
}

// TestAuthService_Login_DatabaseError データベースエラーのテスト
func TestAuthService_Login_DatabaseError(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// データベースエラーを設定
	deps.DB.SetError("database connection failed")

	// ログイン試行
	result, err := authService.Login("test_user", "password123")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestAuthService_Login_TokenGenerationError トークン生成エラーのテスト
func TestAuthService_Login_TokenGenerationError(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// テストユーザーをセットアップ
	_, err := deps.SetupTestUserWithMock("test_user", "password123")
	assert.NoError(t, err)

	// JWTサービスでエラーを設定
	deps.JWTService.SetError(true)

	// ログイン試行
	result, err := authService.Login("test_user", "password123")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "トークンを生成できませんでした", err.Error())
}

// TestAuthService_Register_Success ユーザー登録成功のテスト
func TestAuthService_Register_Success(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// ユーザー登録実行
	user, err := authService.Register("新規ユーザー", "new_user", "password123")

	// 結果検証
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "新規ユーザー", user.Name)
	assert.Equal(t, "new_user", user.AccountID)
	assert.Equal(t, "mock_hashed_password123", user.PasswordHash)
	assert.NotZero(t, user.ID) // モックで自動採番されたID
}

// TestAuthService_Register_ShortPassword 短いパスワードでの登録テスト
func TestAuthService_Register_ShortPassword(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// 短いパスワードで登録試行
	user, err := authService.Register("テストユーザー", "test_user", "1234567")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "パスワードは8文字以上で入力してください", err.Error())
}

// TestAuthService_Register_InvalidAccountID 不正なアカウントIDでの登録テスト
func TestAuthService_Register_InvalidAccountID(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// 短いアカウントIDで登録試行
	user, err := authService.Register("テストユーザー", "ab", "password123")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "アカウントIDは3文字以上20文字以下で入力してください", err.Error())

	// 長いアカウントIDで登録試行
	user, err = authService.Register("テストユーザー", "this_is_a_very_long_account_id", "password123")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "アカウントIDは3文字以上20文字以下で入力してください", err.Error())
}

// TestAuthService_Register_DuplicateAccountID 重複アカウントIDでの登録テスト
func TestAuthService_Register_DuplicateAccountID(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// 最初のユーザーを登録
	_, err := authService.Register("ユーザー1", "test_user", "password123")
	assert.NoError(t, err)

	// 同じアカウントIDで再登録試行
	user, err := authService.Register("ユーザー2", "test_user", "password456")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "このアカウントIDは既に使用されています", err.Error())
}

// TestAuthService_Register_PasswordHashingError パスワードハッシュ化エラーのテスト
func TestAuthService_Register_PasswordHashingError(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// パスワードハッシャーでエラーを設定
	deps.PasswordHasher.SetError(true)

	// ユーザー登録試行
	user, err := authService.Register("テストユーザー", "test_user", "password123")

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "パスワードの処理に失敗しました", err.Error())
}

// TestAuthService_GetUserByID_Success ユーザーID取得成功のテスト
func TestAuthService_GetUserByID_Success(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// テストユーザーをセットアップ
	testUser, err := deps.SetupTestUserWithMock("test_user", "password123")
	assert.NoError(t, err)

	// ユーザー取得実行
	user, err := authService.GetUserByID(testUser.ID)

	// 結果検証
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, "test_user", user.AccountID)
}

// TestAuthService_GetUserByID_NotFound 存在しないユーザーID取得のテスト
func TestAuthService_GetUserByID_NotFound(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// 存在しないユーザーID取得試行
	user, err := authService.GetUserByID(999)

	// 結果検証
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "ユーザーが見つかりません", err.Error())
}

// TestAuthService_GetAllUsers_Success 全ユーザー取得成功のテスト
func TestAuthService_GetAllUsers_Success(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// 複数のテストユーザーをセットアップ
	user1, err := deps.SetupTestUserWithMock("user1", "password123")
	assert.NoError(t, err)
	user2, err := deps.SetupTestUserWithMock("user2", "password456")
	assert.NoError(t, err)

	// 全ユーザー取得実行
	users, err := authService.GetAllUsers()

	// 結果検証
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// ユーザーIDでソート（モックでは順序が保証されないため）
	if users[0].ID > users[1].ID {
		users[0], users[1] = users[1], users[0]
	}

	assert.Equal(t, user1.ID, users[0].ID)
	assert.Equal(t, user2.ID, users[1].ID)
}

// TestAuthService_GetAllUsers_Empty 空のユーザーリスト取得のテスト
func TestAuthService_GetAllUsers_Empty(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// モック依存性をセットアップ
	deps := testmocks.NewMockTestDependencies()
	authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

	// 全ユーザー取得実行（ユーザーが存在しない状態）
	users, err := authService.GetAllUsers()

	// 結果検証
	assert.NoError(t, err)
	assert.Empty(t, users)
}

// ========================================
// モック vs 実際のサービスの比較テスト
// ========================================

// TestAuthService_MockVsReal_Performance モックと実際のサービスのパフォーマンス比較
func TestAuthService_MockVsReal_Performance(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	// この比較テストは実際のDBが利用できない環境では実行しない
	t.Skip("Performance comparison test - requires actual database setup")

	// モックバージョンのテスト実行時間測定などを実装可能
}

// ========================================
// エラー処理の包括的テスト
// ========================================

// TestAuthService_ErrorHandling_Comprehensive 包括的エラーハンドリングテスト
func TestAuthService_ErrorHandling_Comprehensive(t *testing.T) {
	// ユニットテストは並列化を無効にして安定性を重視

	tests := []struct {
		name        string
		setupMock   func(*testmocks.MockTestDependencies)
		operation   func(*AuthService) error
		expectedErr string
	}{
		{
			name: "データベース接続エラー",
			setupMock: func(deps *testmocks.MockTestDependencies) {
				deps.DB.SetError("connection failed")
			},
			operation: func(service *AuthService) error {
				_, err := service.Login("test", "password")
				return err
			},
			expectedErr: "connection failed",
		},
		{
			name: "パスワードハッシュ化エラー",
			setupMock: func(deps *testmocks.MockTestDependencies) {
				deps.PasswordHasher.SetError(true)
			},
			operation: func(service *AuthService) error {
				_, err := service.Register("test", "testuser", "password123")
				return err
			},
			expectedErr: "パスワードの処理に失敗しました",
		},
		{
			name: "JWT生成エラー",
			setupMock: func(deps *testmocks.MockTestDependencies) {
				deps.SetupTestUserWithMock("test", "password123")
				deps.JWTService.SetError(true)
			},
			operation: func(service *AuthService) error {
				_, err := service.Login("test", "password123")
				return err
			},
			expectedErr: "トークンを生成できませんでした",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モック依存性をセットアップ
			deps := testmocks.NewMockTestDependencies()
			authService := NewAuthService(deps.DB, deps.PasswordHasher, deps.JWTService)

			// テストケース固有のモック設定
			tt.setupMock(deps)

			// オペレーション実行
			err := tt.operation(authService)

			// 結果検証
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}
