# ========================================
# Money Management API - Makefile
# テスト実行・ドキュメント生成・開発ツール
# ========================================

# デフォルトターゲット
.DEFAULT_GOAL := help

help:
	@echo ""
	@echo "Money Management API - 開発ツール"
	@echo ""
	@echo "📚 ドキュメント生成:"
	@echo "  make docs              - API仕様書をすべての形式で生成"
	@echo "  make docs-markdown     - Markdown形式のAPI仕様書を生成"
	@echo "  make docs-openapi      - OpenAPI JSON仕様書を生成"
	@echo "  make docs-yaml         - OpenAPI YAML仕様書を生成"
	@echo ""
	@echo "🧪 テスト実行:"
	@echo "  make test              - 全テスト実行（MySQL起動込み）"
	@echo "  make test-only         - テストのみ実行（MySQL手動管理）"
	@echo "  make test-coverage     - カバレッジ測定付きテスト"
	@echo "  make test-contract     - 契約テストのみ実行"
	@echo "  make test-short        - 短時間テスト実行"
	@echo "  make test-e2e          - E2Eテスト実行（フルスタック）"
	@echo "  make test-e2e-ui       - E2EテストをUIモードで実行"
	@echo "  make test-e2e-fast     - 高速E2Eテスト（スモーク+回帰）"
	@echo "  make test-e2e-smoke    - スモークテストのみ実行"
	@echo "  make test-e2e-visual   - ビジュアルリグレッションテスト"
	@echo ""
	@echo "🗄️  データベース管理:"
	@echo "  make test-db-up        - テスト用MySQLコンテナ起動"
	@echo "  make test-db-down      - テスト用MySQLコンテナ停止"
	@echo ""
	@echo "🏗️  ビルド・クリーンアップ:"
	@echo "  make build             - APIサーバーをビルド"
	@echo "  make clean             - 生成ファイルを削除"
	@echo ""

# ========================================
# ドキュメント生成
# ========================================

# すべての形式でAPI仕様書を生成
docs: docs-markdown docs-openapi docs-yaml
	@echo "✅ 全てのAPI仕様書を生成しました"
	@echo "📁 生成されたファイル:"
	@ls -la API_SPECIFICATION.md openapi.json openapi.yaml 2>/dev/null || true

# Markdown形式のAPI仕様書を生成（ルートディレクトリ）
docs-markdown:
	@echo "📝 Markdown API仕様書を生成中..."
	@cd backend && go run cmd/generate-docs/main.go -format markdown -output ../API_SPECIFICATION.md
	@echo "✅ API_SPECIFICATION.md を生成しました"

# OpenAPI JSON仕様書を生成
docs-openapi:
	@echo "📊 OpenAPI JSON仕様書を生成中..."
	@cd backend && go run cmd/generate-docs/main.go -format openapi-json -output ../openapi.json
	@echo "✅ openapi.json を生成しました"

# OpenAPI YAML仕様書を生成
docs-yaml:
	@echo "📋 OpenAPI YAML仕様書を生成中..."
	@cd backend && go run cmd/generate-docs/main.go -format openapi-yaml -output ../openapi.yaml
	@echo "✅ openapi.yaml を生成しました"

# ========================================
# テスト実行
# ========================================

# テスト用MySQLコンテナを起動してテストを実行
test:
	@echo "🚀 テスト用MySQLコンテナを起動中..."
	docker-compose -f docker-compose.test.yml up -d test-database
	@echo "⏰ MySQLの起動を待機中..."
	sleep 10
	@echo "🧪 自動テストを実行中..."
	cd backend && go test ./... -v
	@echo "🧹 テスト用コンテナをクリーンアップ中..."
	docker-compose -f docker-compose.test.yml down -v

# テスト用MySQLコンテナのみ起動
test-db-up:
	@echo "🚀 テスト用MySQLコンテナを起動中..."
	docker-compose -f docker-compose.test.yml up -d test-database

# テスト用MySQLコンテナのみ停止・削除
test-db-down:
	@echo "🧹 テスト用MySQLコンテナを停止・削除中..."
	docker-compose -f docker-compose.test.yml down -v

# テスト環境でのテスト実行のみ（コンテナは手動管理）
test-only:
	@echo "🧪 自動テストを実行中..."
	cd backend && go test ./... -v

# テストカバレッジを計測して実行
test-coverage:
	@echo "🚀 テスト用MySQLコンテナを起動中..."
	docker-compose -f docker-compose.test.yml up -d test-database
	@echo "⏰ MySQLの起動を待機中..."
	sleep 10
	@echo "📊 カバレッジ測定付きテストを実行中..."
	cd backend && go test ./... -v -coverprofile=coverage.out
	cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "📈 カバレッジレポートが coverage.html に生成されました"
	@echo "🧹 テスト用コンテナをクリーンアップ中..."
	docker-compose -f docker-compose.test.yml down -v

# 契約テストのみ実行
test-contract:
	@echo "📋 契約テストを実行中..."
	cd backend && go test -v ./internal/testing -run "Contract"
	cd backend && go test -v ./internal/handlers -run "ContractTestSuite"
	@echo "✅ 契約テストが完了しました"

# 短時間テスト実行（並列化なし、安定性重視）
test-short:
	@echo "⚡ 短時間テストを実行中..."
	cd backend && go test -short ./internal/...
	@echo "✅ 短時間テストが完了しました"

# E2Eテスト実行（フルスタック環境）
test-e2e:
	@echo "🌐 E2Eテスト環境を起動中..."
	docker-compose -f docker-compose.test.yml up -d --wait
	@echo "⏳ サービスの準備完了を待機中..."
	sleep 10
	@echo "🎭 Playwright E2Eテストを実行中..."
	cd frontend && npm run test:e2e
	@echo "🧹 E2Eテスト環境をクリーンアップ中..."
	docker-compose -f docker-compose.test.yml down -v
	@echo "✅ E2Eテストが完了しました"

# E2EテストをUIモードで実行
test-e2e-ui:
	@echo "🌐 E2Eテスト環境を起動中..."
	docker-compose -f docker-compose.test.yml up -d --wait
	@echo "⏳ サービスの準備完了を待機中..."
	sleep 10
	@echo "🎭 Playwright E2EテストをUIモードで実行中..."
	cd frontend && npm run test:e2e:ui
	@echo "🧹 手動でコンテナを停止してください: make test-db-down"

# 高速E2Eテスト（スモーク+回帰テストのみ）
test-e2e-fast:
	@echo "🌐 E2Eテスト環境を起動中..."
	docker-compose -f docker-compose.test.yml up -d --wait
	@echo "⏳ サービスの準備完了を待機中..."
	sleep 10
	@echo "⚡ 高速E2Eテストを実行中..."
	cd frontend && npm run test:e2e:fast
	@echo "🧹 E2Eテスト環境をクリーンアップ中..."
	docker-compose -f docker-compose.test.yml down -v
	@echo "✅ 高速E2Eテストが完了しました"

# スモークテストのみ実行
test-e2e-smoke:
	@echo "🌐 E2Eテスト環境を起動中..."
	docker-compose -f docker-compose.test.yml up -d --wait
	@echo "⏳ サービスの準備完了を待機中..."
	sleep 10
	@echo "🔥 スモークテストを実行中..."
	cd frontend && npm run test:e2e:smoke
	@echo "🧹 E2Eテスト環境をクリーンアップ中..."
	docker-compose -f docker-compose.test.yml down -v
	@echo "✅ スモークテストが完了しました"

# ビジュアルリグレッションテスト
test-e2e-visual:
	@echo "🌐 E2Eテスト環境を起動中..."
	docker-compose -f docker-compose.test.yml up -d --wait
	@echo "⏳ サービスの準備完了を待機中..."
	sleep 10
	@echo "👁️ ビジュアルリグレッションテストを実行中..."
	cd frontend && npm run test:e2e:visual
	@echo "🧹 E2Eテスト環境をクリーンアップ中..."
	docker-compose -f docker-compose.test.yml down -v
	@echo "✅ ビジュアルリグレッションテストが完了しました"

# ========================================
# ビルド・クリーンアップ
# ========================================

# APIサーバーをビルド
build:
	@echo "🏗️  APIサーバーをビルド中..."
	cd backend && go build -o ../bin/money-management-api ./main.go
	@echo "✅ ビルドが完了しました: bin/money-management-api"

# 生成ファイルを削除
clean:
	@echo "🧹 生成ファイルを削除中..."
	rm -f API_SPECIFICATION.md
	rm -f openapi.json
	rm -f openapi.yaml
	rm -rf bin/
	rm -f backend/coverage.out
	rm -f backend/coverage.html
	@echo "✅ クリーンアップが完了しました"

# ========================================
# 開発者向けショートカット
# ========================================

# よく使うコマンドのエイリアス
md: docs-markdown
json: docs-openapi
yaml: docs-yaml
tc: test-contract
ts: test-short
e2e: test-e2e
e2e-ui: test-e2e-ui
e2e-fast: test-e2e-fast
e2e-smoke: test-e2e-smoke
e2e-visual: test-e2e-visual

.PHONY: help docs docs-markdown docs-openapi docs-yaml test test-db-up test-db-down test-only test-coverage test-contract test-short test-e2e test-e2e-ui test-e2e-fast test-e2e-smoke test-e2e-visual build clean md json yaml tc ts e2e e2e-ui e2e-fast e2e-smoke e2e-visual
