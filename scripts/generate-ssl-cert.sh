#!/bin/bash

# 開発用SSL証明書生成スクリプト
# 注意: これは開発・テスト専用です。本番環境では正規の証明書を使用してください。

set -e

# 設定
SSL_DIR="$(dirname "$0")/../ssl"
CERT_FILE="$SSL_DIR/localhost.crt"
KEY_FILE="$SSL_DIR/localhost.key"
CONFIG_FILE="$SSL_DIR/localhost.conf"

echo "🔐 開発用SSL証明書を生成しています..."

# SSL ディレクトリを作成
mkdir -p "$SSL_DIR"

# OpenSSL設定ファイルを作成
cat > "$CONFIG_FILE" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = JP
ST = Tokyo
L = Tokyo
O = Development
OU = IT Department
CN = localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# 秘密鍵を生成
openssl genrsa -out "$KEY_FILE" 2048

# 証明書署名要求（CSR）を生成
openssl req -new -key "$KEY_FILE" -out "$SSL_DIR/localhost.csr" -config "$CONFIG_FILE"

# 自己署名証明書を生成（1年間有効）
openssl x509 -req -in "$SSL_DIR/localhost.csr" -signkey "$KEY_FILE" -out "$CERT_FILE" -days 365 -extensions v3_req -extfile "$CONFIG_FILE"

# CSRファイルを削除
rm "$SSL_DIR/localhost.csr"

# ファイル権限を設定
chmod 600 "$KEY_FILE"
chmod 644 "$CERT_FILE"

echo "✅ SSL証明書が生成されました:"
echo "   証明書: $CERT_FILE"
echo "   秘密鍵: $KEY_FILE"
echo ""
echo "⚠️ これは開発専用の自己署名証明書です。"
echo "   ブラウザで警告が表示されますが、開発環境では安全に無視できます。"
echo ""
echo "🚀 HTTPS対応でアプリケーションを起動:"
echo "   docker-compose -f docker-compose.https.yml up --build"