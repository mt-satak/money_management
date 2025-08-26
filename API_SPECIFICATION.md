# Money Management API 仕様書（セキュリティ強化版）

**バージョン**: 2.0.0
**生成日時**: 2025-08-26 11:20:00
**生成方法**: 実装に基づく正確な仕様書作成
**セキュリティレベル**: Phase 3 完了済み（本番環境対応）

## 概要

Money Management APIは家計簿管理システムのセキュリティ強化されたRESTful APIです。ユーザー認証、家計簿の作成・編集・請求・支払い・削除機能を提供します。

**主要機能**:
- JWT認証システム（セキュリティ強化済み）
- 月次家計簿管理（CRUD操作）
- 重複作成防止
- 権限制御（請求者・支払者別）
- ワークフロー管理（pending→requested→paid）
- セキュリティ監視・管理機能

## 認証

JWT (JSON Web Token) ベースの認証を使用します。認証が必要なエンドポイントでは、リクエストヘッダーに以下を含めてください：

```
Authorization: Bearer <JWT_TOKEN>
```

## セキュリティ実装状況（Phase 3 完了）

**✅ 本番環境対応済み** - 以下のセキュリティ対策が実装されています：

### Phase 1 (Critical) - 実装済み
- ✅ JWT認証（環境変数・Docker Secrets対応）
- ✅ データベースパスワード保護
- ✅ CSRF保護実装（utrack/gin-csrf）

### Phase 2 (High Risk) - 実装済み
- ✅ HTTPS強制対応
- ✅ 3段階APIレート制限
- ✅ セッション管理強化（ログアウト・トークンブラックリスト）

### Phase 3 (Medium Risk) - 実装済み
- ✅ セキュリティヘッダー実装（CSP, HSTS, X-Frame-Options等）
- ✅ 入力値検証・サニタイゼーション（SQLインジェクション・XSS防止）
- ✅ セキュアエラーハンドリング
- ✅ コンテナセキュリティ強化

## エンドポイント一覧


### 認証

#### Login API

**POST** `/auth/login`

ユーザーログインAPI

**リクエスト例:**

```json
{
  "account_id": "example_ログイン用アカウントid",
  "password": "example_ログインパスワード"
}
```

**レスポンス例** (HTTP 200):

```json
{
  "token": "example_jwtアクセストークン",
  "user": "example_value"
}
```

#### Register API

**POST** `/auth/register`

新規ユーザー登録API

**リクエスト例:**

```json
{
  "account_id": "example_一意のアカウントid",
  "name": "example_ユーザー名",
  "password": "example_パスワード"
}
```

**レスポンス例** (HTTP 201):

```json
{
  "token": "example_jwtアクセストークン",
  "user": "example_value"
}
```

#### Get Me API

**GET** `/auth/me`

認証済みユーザーの情報取得API

**レスポンス例** (HTTP 200):

```json
{
  "account_id": "example_アカウントid",
  "created_at": "2024-01-01T00:00:00Z",
  "id": 1,
  "name": "example_ユーザー名",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Logout API

**POST** `/auth/logout`

ログアウトAPI（現在のJWTトークンを無効化）

**権限**: 認証必須

**レスポンス例** (HTTP 200):

```json
{
  "message": "ログアウトしました"
}
```

#### Logout All API

**POST** `/auth/logout-all`

全デバイスログアウトAPI（ユーザーの全JWTトークンを無効化）

**権限**: 認証必須

**レスポンス例** (HTTP 200):

```json
{
  "message": "すべてのデバイスからログアウトしました"
}
```

#### Get Token Status API

**GET** `/auth/token-status`

JWTトークンのステータス確認API

**権限**: 認証必須

**レスポンス例** (HTTP 200):

```json
{
  "valid": true,
  "user_id": 1,
  "expires_at": "2024-01-08T00:00:00Z"
}
```


### ユーザー管理

#### Get Users API

**GET** `/users`

システム内全ユーザー一覧取得API

**レスポンス例** (HTTP 200):

```json
{
  "users": "example_value"
}
```


### 家計簿

#### Create Bill API

**POST** `/bills`

新規家計簿作成API（重複チェック付き）

**権限**: 認証必須
**制限**: 請求者と支払者は異なるユーザーである必要がある

**リクエスト例:**

```json
{
  "year": 2024,
  "month": 3,
  "payer_id": 2
}
```

**レスポンス例** (HTTP 201):

```json
{
  "id": 1,
  "year": 2024,
  "month": 3,
  "requester_id": 1,
  "payer_id": 2,
  "status": "pending",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "requester": {...},
  "payer": {...},
  "items": [],
  "total_amount": 0.0
}
```

**エラーレスポンス** (HTTP 409):

```json
{
  "error": "指定された年月の家計簿は既に存在します"
}
```

#### Get Bill API

**GET** `/bills/:year/:month`

指定年月の家計簿詳細取得API

**権限**: 認証必須（請求者または支払者のみ）

**レスポンス例** (HTTP 200):

```json
{
  "id": 1,
  "year": 2024,
  "month": 3,
  "requester_id": 1,
  "payer_id": 2,
  "status": "pending",
  "requester": {
    "id": 1,
    "name": "請求者名",
    "account_id": "requester_id"
  },
  "payer": {
    "id": 2,
    "name": "支払者名",
    "account_id": "payer_id"
  },
  "items": [
    {
      "id": 1,
      "item_name": "食費",
      "amount": 5000.0
    }
  ],
  "total_amount": 5000.0
}
```

**家計簿がない場合** (HTTP 200):

```json
{
  "bill": null
}
```

#### Get Bills List API

**GET** `/bills`

家計簿一覧取得API（請求者・支払者両方の家計簿）

**権限**: 認証必須

**レスポンス例** (HTTP 200):

```json
{
  "bills": [
    {
      "id": 1,
      "year": 2024,
      "month": 3,
      "requester_id": 1,
      "payer_id": 2,
      "status": "requested",
      "total_amount": 15000.0,
      "requester": {...},
      "payer": {...}
    }
  ]
}
```

#### Update Bill Items API

**PUT** `/bills/:id/items`

家計簿項目更新API

**権限**: 認証必須（請求者のみ、pending状態のみ）

**リクエスト例:**

```json
{
  "items": [
    {
      "item_name": "食費",
      "amount": 5000.0
    },
    {
      "item_name": "交通費",
      "amount": 2000.0
    }
  ]
}
```

#### Request Bill API

**PUT** `/bills/:id/request`

請求確定API（pending → requested）

**権限**: 認証必須（請求者のみ）

**レスポンス例** (HTTP 200):

```json
{
  "message": "家計簿の請求が確定しました"
}
```

#### Payment Bill API

**PUT** `/bills/:id/payment`

支払確定API（requested → paid）

**権限**: 認証必須（支払者のみ）

**レスポンス例** (HTTP 200):

```json
{
  "message": "支払いが確定しました"
}
```

#### Delete Bill API

**DELETE** `/bills/:id`

家計簿削除API

**権限**: 認証必須（請求者のみ、pending状態のみ）

**レスポンス例** (HTTP 200):

```json
{
  "message": "家計簿を削除しました"
}
```


### セキュリティ・監視

#### Health Check API

**GET** `/health`

ヘルスチェックAPI（認証不要）

**レスポンス例** (HTTP 200):

```json
{
  "status": "ok",
  "message": "家計簿API稼働中"
}
```

#### CSRF Token API

**GET** `/api/csrf-token`

CSRFトークン取得API（認証不要）

**レスポンス例** (HTTP 200):

```json
{
  "csrf_token": "MjQyODM0NzQ2NzQ4NzY4NzQ2NzQ2NzQ2NzQ2NzQ2"
}
```

#### Security Status API

**GET** `/api/security-status`

セキュリティ監視情報取得API（認証不要・管理用）

**レスポンス例** (HTTP 200):

```json
{
  "rate_limit": {
    "auth_clients": 2,
    "create_clients": 1,
    "general_clients": 5,
    "limits": {
      "auth": {
        "burst": 3,
        "requests_per_minute": 10
      },
      "create": {
        "burst": 10,
        "requests_per_minute": 30
      },
      "general": {
        "burst": 20,
        "requests_per_minute": 100
      }
    }
  },
  "token_blacklist": {
    "expired_tokens": 12,
    "total_tokens": 25,
    "valid_tokens": 13
  }
}
```

## セキュリティ仕様

### レート制限

本APIでは3段階のレート制限を実装しています：

| 種別 | エンドポイント | 制限 | バースト |
|------|----------------|------|----------|
| 一般API | `/api/*` | 100req/min | 20 |
| 認証API | `/api/auth/*` | 10req/min | 3 |
| 作成系API | 作成・更新系 | 30req/min | 10 |

### CSRFプロテクション

すべてのPOST/PUT/DELETEリクエストにはCSRFトークンが必要です：
1. `/api/csrf-token` でトークンを取得
2. リクエストヘッダーに `X-CSRF-Token` を含める

### セキュリティヘッダー

レスポンスに以下のセキュリティヘッダーが自動設定されます：
- `Content-Security-Policy`
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`


## エラーレスポンス

すべてのエラーは以下の形式で返されます：

```json
{
  "error": "エラーメッセージ"
}
```

### HTTPステータスコード

| ステータスコード | 説明 |
|-----------------|------|
| 200 | 成功 |
| 201 | 作成成功 |
| 400 | リクエストエラー |
| 401 | 認証エラー |
| 403 | 権限エラー・CSRF・レート制限 |
| 404 | リソースが見つからない |
| 409 | 競合エラー |
| 429 | レート制限超過 |
| 500 | サーバーエラー |

## 品質保証

### テスト体系
- **バックエンド**: 73テストケース（カバレッジ87.3%）
  - ユニットテスト: 15テスト（モック使用）
  - 統合テスト: 41テスト（実DB使用）
  - 契約テスト: 17テスト（API仕様検証）

- **フロントエンド**: 268 E2Eテストケース
  - 4ブラウザ対応（Chrome/Safari Desktop・Mobile）
  - ビジュアルリグレッション: 32テスト
  - パフォーマンス監視: Core Web Vitals測定

### 実行コマンド
```bash
# バックエンドテスト
cd backend && go test ./...

# フロントエンドE2Eテスト
cd frontend && npm run test:e2e

# 契約テスト・API仕様書生成
cd backend && make docs
```

本API仕様書は実装コードに基づいて作成されており、継続的な品質保証システムによって整合性が維持されています。

- **API契約カバレッジ**: 100%（13エンドポイント）
- **セキュリティレベル**: Phase 3 完了済み（本番環境対応）
- **最終更新**: 2025-08-26 11:20:00
- **セキュリティ監査**: Phase 1-3 完了（`SECURITY_AUDIT_REPORT.md`参照）
- **レート制限実装**: 3段階（一般・認証・作成系）
- **CSRF保護**: 実装済み
- **セキュリティヘッダー**: 全面実装済み
