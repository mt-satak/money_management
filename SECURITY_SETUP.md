# セキュリティ構成ガイド

## 🔐 セキュリティ強化済み構成

このドキュメントでは、セキュリティ監査の結果に基づいて実装された強化策について説明します。

## 📋 実装済み対策

### 1. JWTシークレット環境変数化 ✅ 完了

**問題**: JWTシークレットのハードコーディング (CVSS 9.8)
**対策**: 環境変数 + Docker Secrets対応

```go
// before: ハードコーディング
var jwtSecret = []byte("your-super-secret-jwt-key-change-in-production")

// after: 環境変数 + Docker Secrets
func GetJWTSecret() []byte {
    // 1. Docker Secrets優先 (/run/secrets/jwt_secret)
    // 2. JWT_SECRET環境変数フォールバック
}
```

### 2. データベースパスワード保護 ✅ 完了

**問題**: データベースパスワードの平文保存 (CVSS 9.1)
**対策**: Docker Secrets + 環境変数対応

```go
func getSecurePassword() string {
    // 1. Docker Secrets優先 (/run/secrets/db_password)
    // 2. DB_PASSWORD環境変数フォールバック
    // 3. 開発用デフォルト（警告付き）
}
```

### 3. CSRF保護実装 ✅ 完了

**問題**: CSRF攻撃によるデータ改ざん (CVSS 8.8)
**対策**: gin-csrf ミドルウェア + 環境変数対応

```go
// CSRF設定の安全な取得
func GetCSRFSecret() string {
    // 1. Docker Secrets優先 (/run/secrets/csrf_secret)
    // 2. CSRF_SECRET環境変数フォールバック
    // 3. 開発用デフォルト（警告付き）
}

// セッション設定の安全な取得
func GetSessionSecret() []byte {
    // 1. Docker Secrets優先 (/run/secrets/session_secret)
    // 2. SESSION_SECRET環境変数フォールバック
    // 3. 開発用デフォルト（警告付き）
}
```

### 4. HTTPS強制実装 ✅ 完了

**問題**: 通信の盗聴・改ざん (CVSS 7.5)
**対策**: nginx SSL設定 + セキュリティヘッダー

```nginx
# HTTPS設定
server {
    listen 443 ssl http2;
    ssl_protocols TLSv1.2 TLSv1.3;

    # セキュリティヘッダー
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Content-Security-Policy "default-src 'self'" always;
}
```

### 5. APIレート制限実装 ✅ 完了

**問題**: DDoS攻撃・ブルートフォース (CVSS 6.5)
**対策**: golang.org/x/time/rate ベースの制限

```go
// 認証エンドポイント: 1分間10リクエスト（ブルートフォース対策）
authLimiter := NewRateLimiter(10, 3)

// 一般API: 1分間100リクエスト
generalLimiter := NewRateLimiter(100, 20)

// 作成系API: 1分間30リクエスト
createLimiter := NewRateLimiter(30, 10)
```

### 6. セッション管理強化 ✅ 完了

**問題**: セッション無効化機能の欠如 (CVSS 6.8)
**対策**: JWTトークンブラックリスト + ログアウト機能

```go
// トークンブラックリスト管理
func AddTokenToBlacklist(token string, expiresAt time.Time, reason string)
func IsTokenBlacklisted(token string) bool

// ログアウトエンドポイント
POST /api/auth/logout       // 単一セッションログアウト
POST /api/auth/logout-all   // 全デバイスログアウト
GET  /api/auth/token-status // トークンステータス確認
```

## 🚀 使用方法

### 開発環境（従来通り）

```bash
# 従来のdocker-compose.yml使用
docker-compose up --build
```

### セキュア環境（推奨）

```bash
# セキュア版compose使用
docker-compose -f docker-compose.secure.yml up --build
```

## 📁 セキュリティファイル構成

```
secrets/
├── jwt_secret.txt          # JWTシークレットキー（64文字）
├── db_password.txt         # データベースパスワード（64文字）
├── csrf_secret.txt         # CSRFトークン生成用シークレット（64文字）
├── session_secret.txt      # セッション暗号化用シークレット（64文字）
└── mysql_root_password.txt # MySQLルートパスワード（64文字）

ssl/
├── localhost.crt           # 開発用SSL証明書
├── localhost.key           # 開発用SSL秘密鍵
└── localhost.conf          # 証明書生成設定
```

## 🔧 セキュリティ機能

### Docker Secrets対応

1. **優先順位**: Docker Secrets > 環境変数 > デフォルト値
2. **ファイル読み込み**: `/run/secrets/` から自動読み込み
3. **ログ出力**: 読み込み元の明確な表示

### 検証機能

- **JWT最小長チェック**: 32文字以上を強制
- **空文字チェック**: 空の設定値を検出
- **改行文字除去**: Docker Secretsファイルの改行処理

## ⚠️ セキュリティ注意事項

### 🚨 重要：本番環境での機微情報管理

**現在のsecretsディレクトリは開発・テスト専用です。本番環境では絶対に使用しないでください。**

### 本番環境での正しい対応

#### 1. 機微情報の管理方法

**❌ 避けるべき方法**:
- secretsディレクトリをそのまま本番環境にデプロイ
- ソースコードに機微情報を含める
- 環境変数にプレーンテキストで設定

**✅ 推奨する方法**:

##### クラウド環境での対応
```bash
# AWS Secrets Manager
aws secretsmanager create-secret --name "household-budget/jwt-secret" --secret-string "$(openssl rand -base64 48)"

# Azure Key Vault
az keyvault secret set --vault-name "household-budget-kv" --name "jwt-secret" --value "$(openssl rand -base64 48)"

# Google Secret Manager
echo -n "$(openssl rand -base64 48)" | gcloud secrets create jwt-secret --data-file=-
```

##### オンプレミス環境での対応
```bash
# 本番環境でのみ生成（サーバー上で直接実行）
openssl rand -base64 48 > /run/secrets/jwt_secret
openssl rand -base64 48 > /run/secrets/db_password
openssl rand -base64 48 > /run/secrets/csrf_secret
openssl rand -base64 48 > /run/secrets/session_secret
openssl rand -base64 48 > /run/secrets/mysql_root_password

# 厳格な権限設定
chmod 400 /run/secrets/*
chown root:root /run/secrets/*
```

#### 2. デプロイ時の注意点

**開発環境**:
- secretsディレクトリを使用（.gitignoreで除外済み）
- 環境変数でのオーバーライドも可能

**ステージング/本番環境**:
- Docker Secrets、Kubernetes Secrets、またはクラウドシークレット管理サービスを使用
- 環境変数は外部シークレット管理システムから注入
- secretsディレクトリは存在させない

#### 3. CI/CDでの対応

```yaml
# 例: GitHub Actions
- name: 本番デプロイ
  env:
    JWT_SECRET: ${{ secrets.JWT_SECRET }}
    DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
    CSRF_SECRET: ${{ secrets.CSRF_SECRET }}
    SESSION_SECRET: ${{ secrets.SESSION_SECRET }}
  run: docker-compose -f docker-compose.production.yml up -d
```

#### 4. 本番環境チェックリスト

- [ ] secretsディレクトリが本番環境に存在しない
- [ ] 全てのシークレットが外部管理システムから取得
- [ ] シークレットローテーション計画の策定
- [ ] アクセスログの監視設定
- [ ] 定期的なセキュリティ監査の実施

### Docker Swarmでの使用

```yaml
# docker-compose.secure.yml をベースに
secrets:
  jwt_secret:
    external: true
  db_password:
    external: true
```

## 🧪 テスト環境

### 環境変数によるテスト実行

```bash
JWT_SECRET="test-jwt-secret-for-testing-purposes-32chars" \
DB_PASSWORD="test-password" \
CSRF_SECRET="test-csrf-secret-for-testing-purposes-32chars" \
SESSION_SECRET="test-session-secret-for-testing-purposes-32chars" \
go test ./...
```

## 📊 セキュリティレベル比較

| 項目 | 従来 | セキュア版 | 改善度 |
|------|------|-----------|--------|
| JWTシークレット | ハードコード | Docker Secrets | 🔴→🟢 |
| DBパスワード | 平文 | Docker Secrets | 🔴→🟢 |
| CSRF保護 | なし | 実装済み | 🔴→🟢 |
| HTTPS強制 | なし | SSL対応+セキュリティヘッダー | 🔴→🟢 |
| レート制限 | なし | 多層レート制限 | 🔴→🟢 |
| セッション管理 | トークン永続 | ブラックリスト+ログアウト | 🔴→🟢 |
| セッション暗号化 | デフォルト | 強力なシークレット | 🟠→🟢 |
| 外部DB公開 | 公開(3306) | 非公開 | 🟠→🟢 |
| パスワード長 | チェックなし | 32文字以上 | ➕ |

## 🔄 次のステップ

### 未実装の対策（今後対応予定）

**Phase 3: セキュリティ強化（中リスク対応）**
1. **入力値検証の強化** (CVSS 5.1)
2. **エラーハンドリング改善** (CVSS 4.8)
3. **コンテナセキュリティ強化** (CVSS 4.5)

**Phase 4: 運用セキュリティ（継続的）**
1. **監査ログの実装** (LOW)
2. **バックアップ戦略** (LOW)
3. **定期的な脆弱性スキャン** (継続)

### 監査・モニタリング

- 定期的なSecretsローテーション
- ログ監視（不正アクセス検知）
- 依存関係の脆弱性スキャン
