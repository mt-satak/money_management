# 開発環境セットアップ

このファイルでは、ホットリロード機能を有効にした開発環境の使用方法を説明します。

## 開発環境の起動

```bash
# 開発環境を起動
docker-compose -f docker-compose.dev.yml up --build -d

# ログを確認（オプション）
docker-compose -f docker-compose.dev.yml logs -f frontend
```

## 本番環境の起動

```bash
# セキュア版（Docker Secrets使用）
docker-compose -f docker-compose.secure.yml up --build -d

# 標準版（開発用、secretsディレクトリ使用）
docker-compose up --build -d
```

**⚠️ 重要**:
- `secrets/` ディレクトリは開発・テスト専用
- 本番環境では外部シークレット管理システム（AWS Secrets Manager等）を使用
- 詳細は `SECURITY_SETUP.md` および `docker-compose.production.example.yml` を参照

## 環境の切り替え

```bash
# 現在の環境を停止
docker-compose down

# または開発環境を停止
docker-compose -f docker-compose.dev.yml down

# 新しい環境を起動
docker-compose -f docker-compose.dev.yml up --build -d  # 開発環境
# または
docker-compose up --build -d                           # 本番環境
```

## ホットリロード機能

開発環境では以下の機能が有効です：

- **ファイル変更の自動検出**: `frontend/src`フォルダ内のファイルを変更すると自動的にブラウザに反映
- **ボリュームマウント**: ローカルのソースコードがコンテナ内にマウントされます
- **ポーリングベースの監視**: Dockerコンテナ環境でも確実にファイル変更を検出

## アクセス先

- **フロントエンド**: http://localhost:3000
- **バックエンドAPI**: http://localhost:8080

## トラブルシューティング

### ホットリロードが動作しない場合

1. コンテナを再起動
```bash
docker-compose -f docker-compose.dev.yml restart frontend
```

2. ログを確認
```bash
docker-compose -f docker-compose.dev.yml logs frontend
```

3. ブラウザのキャッシュをクリア

### ポートが使用中の場合

他のプロセスがポート3000や8080を使用している場合は、そのプロセスを停止してから再起動してください。
