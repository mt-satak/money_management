# 家計簿アプリケーション

セキュリティ強化された家計簿管理システム（Phase 3 セキュリティ実装完了）

## 🚀 クイックスタート

```zsh
# セキュアなアプリケーションを起動
docker-compose up --build

# バックグラウンドで起動
docker-compose up -d --build

# HTTPS版（本番環境用）
docker-compose -f docker-compose.https.yml up --build
```

## 📱 アクセス方法

- **フロントエンド**: http://localhost:3000 (HTTPS: https://localhost:443)
- **バックエンドAPI**: http://localhost:8080
- **データベース**: localhost:3306
- **セキュリティ監視**: http://localhost:8080/api/security-status

## 👤 ログイン情報

初期ユーザーは登録されていません。
アプリケーション起動後、Webページから新規ユーザー登録を行ってください。

ログインは **アカウントID + パスワード** の組み合わせで行います。

## 🛠️ 技術スタック

- **Frontend**: React 18 + TypeScript + Tailwind CSS + Vite + nginx
- **Backend**: Go 1.23 + Gin + GORM + JWT
- **Database**: MySQL 8.0 (Docker Secrets対応)
- **Container**: Docker + Docker Compose (セキュリティ強化済み)
- **Testing**: Playwright (E2E) + Jest (Unit)

## 📊 実装状況

### ✅ 完了
- JWT認証システム（JWTトークンによる認証・認可）
- ユーザー管理（登録・ログイン・情報取得）
- 月次家計簿管理（作成・一覧・詳細・削除）
- 請求項目CRUD（支出項目の追加・編集・削除・一括更新）
- ワークフロー管理（作成中→請求済み→支払済みの状態遷移）
- 重複防止機能（同一年月の家計簿重複作成防止）
- 権限制御（請求者・支払者別のアクション制限）
- 支払者向け機能（支払者指定家計簿の表示・カード背景色分け）
- Docker環境構築（開発・本番環境分離）
- 包括的テスト体系（Backend 87.3%・Frontend E2E 268テスト）
- セキュリティ監査実施（15の脆弱性特定・対策提案）

### 🔐 セキュリティ機能（Phase 3 完了）
- ✅ **Phase 1 (Critical)**: JWT認証・データベースパスワード・CSRF保護
- ✅ **Phase 2 (High Risk)**: HTTPS強制・APIレート制限・セッション管理
- ✅ **Phase 3 (Medium Risk)**: セキュリティヘッダー・入力値検証・エラーハンドリング・コンテナセキュリティ

**セキュリティ実装詳細:**
- JWT認証（環境変数・Docker Secrets対応）
- CSRF保護（utrack/gin-csrf）
- 3段階APIレート制限（一般・認証・作成系）
- SSL/HTTPS対応（nginx + OpenSSL証明書）
- セキュリティヘッダー（CSP, HSTS, X-Frame-Options等）
- 入力値検証・サニタイゼーション（SQLインジェクション・XSS防止）
- セキュアエラーハンドリング（情報漏洩防止）
- コンテナセキュリティ（非rootユーザー・最小限イメージ）

### ⚠️ 本番環境での注意事項
**🚨 重要**: `secrets/` ディレクトリは開発・テスト専用です。本番環境では：
- AWS Secrets Manager / Azure Key Vault / Google Secret Manager を使用
- 環境変数は外部シークレット管理システムから注入
- `docker-compose.production.example.yml` を参考に設定
- 詳細は `SECURITY_SETUP.md` を参照

### 📋 残り対応項目（Phase 4: 低リスク）
- 監査ログ機能の強化
- バックアップ戦略の策定
- 脆弱性スキャンの自動化
- セキュリティ監視の拡張

## 🔄 開発コマンド

```zsh
# アプリケーション管理
docker-compose logs -f        # ログ確認
docker-compose down           # 停止
docker-compose down -v        # データベースリセット
docker-compose up --build     # リビルド起動

# テスト実行
cd backend && go test ./...           # バックエンドテスト
cd frontend && npm run test:e2e       # フロントエンドE2Eテスト
cd frontend && npm run test:e2e:fast  # 高速回帰テスト

# ドキュメント生成
cd backend && make docs               # API仕様書生成
```

## 🔐 セキュリティ

**✅ セキュリティレベル**: Phase 3 実装完了により本番環境対応済み
詳細な実装状況は `SECURITY_AUDIT_REPORT.md` をご確認ください。

**実装済み対策（Phase 1-3）:**
1. ✅ JWT_SECRET 環境変数化・Docker Secrets対応
2. ✅ データベースパスワード保護
3. ✅ CSRF保護実装
4. ✅ HTTPS強制設定・SSL証明書
5. ✅ APIレート制限（3段階）
6. ✅ セキュリティヘッダー全体実装
7. ✅ 入力値検証・サニタイゼーション
8. ✅ セキュアエラーハンドリング
9. ✅ コンテナセキュリティ強化

**監視エンドポイント**: http://localhost:8080/api/security-status

## 🧪 品質保証

- **Backend テスト**: 73テストケース（カバレッジ87.3%）
- **Frontend E2E**: 268テストケース（4ブラウザ対応）
- **契約テスト**: 7API×100%カバレッジ
- **ビジュアル回帰**: 32テストケース
- **パフォーマンス監視**: Core Web Vitals測定

## 📞 サポート

問題が発生した場合は、GitHubのIssueまたは開発者にお問い合わせください。
セキュリティ関連の問題は `SECURITY_AUDIT_REPORT.md` の緊急対応項目を優先してください。
