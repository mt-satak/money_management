# セキュリティ強化されたマルチステージDockerfile
FROM golang:1.23-alpine AS builder

# セキュリティ更新とビルドツールのインストール
RUN apk update && apk upgrade && \
    apk add --no-cache git ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

# 非rootユーザーでビルド実行
RUN adduser -D -g '' appbuilder
USER appbuilder

WORKDIR /app

# 依存関係ファイルをコピー（キャッシュ効率化）
COPY --chown=appbuilder:appbuilder go.mod go.sum ./

# 依存関係をダウンロード
RUN go mod download && go mod verify

# ソースコードをコピー
COPY --chown=appbuilder:appbuilder . .

# セキュアなビルド設定でコンパイル
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main .

# 最小限の実行イメージ
FROM scratch

# 必要な証明書とタイムゾーンデータをコピー
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 非root実行用ユーザー作成（scratch用）
COPY --from=builder /etc/passwd /etc/passwd

# アプリケーションバイナリをコピー
COPY --from=builder /app/main /app/main

# 非rootユーザーで実行
USER appbuilder

# ヘルスチェック用エンドポイントを公開
EXPOSE 8080

# セキュリティを考慮したエントリーポイント
ENTRYPOINT ["/app/main"]
