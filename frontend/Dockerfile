# セキュリティ強化されたマルチステージDockerfile（フロントエンド）
FROM node:20-alpine AS build

# セキュリティ更新とビルドツール
RUN apk update && apk upgrade && \
    apk add --no-cache dumb-init && \
    rm -rf /var/cache/apk/*

# 非rootユーザーでビルド
RUN addgroup -g 1001 -S nodejs && \
    adduser -S appuser -u 1001 -G nodejs

USER appuser
WORKDIR /app

# 依存関係ファイルをコピー
COPY --chown=appuser:nodejs package*.json ./

# ビルドに必要な依存関係をインストール（devDependenciesも含む）
RUN npm ci --audit && \
    npm cache clean --force

# ソースコードをコピーしてビルド
COPY --chown=appuser:nodejs . .
RUN npm run build

# 本番環境用nginx
FROM nginx:1.25-alpine

# セキュリティ更新
RUN apk update && apk upgrade && \
    apk add --no-cache dumb-init && \
    rm -rf /var/cache/apk/*

# 非rootユーザーでnginx実行
RUN addgroup -g 1001 -S webuser && \
    adduser -S nginxuser -u 1001 -G webuser

# nginxの設定とアクセス権限を調整
RUN mkdir -p /var/cache/nginx/client_temp && \
    chown -R nginxuser:webuser /var/cache/nginx && \
    chown -R nginxuser:webuser /var/log/nginx && \
    chown -R nginxuser:webuser /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && \
    chown nginxuser:webuser /var/run/nginx.pid && \
    mkdir -p /etc/ssl/private && \
    chown -R nginxuser:webuser /etc/ssl/private

# セキュリティ強化されたnginx設定
COPY --chown=nginxuser:webuser nginx.conf /etc/nginx/conf.d/default.conf

# SSL証明書をコピー
COPY --chown=nginxuser:webuser ssl/localhost.crt /etc/ssl/certs/localhost.crt
COPY --chown=nginxuser:webuser ssl/localhost.key /etc/ssl/private/localhost.key

# ビルド成果物をコピー
COPY --from=build --chown=nginxuser:webuser /app/dist /usr/share/nginx/html

# 非rootユーザーで実行
USER nginxuser

# 通信ポート
EXPOSE 80 443

# セキュアなエントリーポイント
ENTRYPOINT ["dumb-init", "--"]
CMD ["nginx", "-g", "daemon off;"]
