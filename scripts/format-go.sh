#!/usr/bin/env bash
# Go code formatting script for pre-commit hook

# エラーハンドリングの有効化
set -e

# gofmtが利用可能かチェック
if ! command -v gofmt &> /dev/null; then
    echo "❌ エラー: gofmtが見つかりません。Goがインストールされているか確認してください。"
    exit 1
fi

# ファイルが指定されているかチェック
if [ $# -eq 0 ]; then
    echo "⚠️  警告: フォーマット対象のファイルが指定されていません。"
    exit 0
fi

echo "🔧 Goコードフォーマットを実行中..."

# 各ファイルをフォーマット
for file in "$@"; do
    if [ -f "$file" ]; then
        echo "   📝 フォーマット中: $file"
        gofmt -w "$file"
    else
        echo "⚠️  警告: ファイルが存在しません: $file"
    fi
done

echo "✅ Goコードフォーマット完了"
