#!/usr/bin/env bash
# YAML/JSON formatting script for pre-commit hook using local Prettier

# エラーハンドリングの有効化
set -e

# ファイルが指定されているかチェック
if [ $# -eq 0 ]; then
    echo "⚠️  警告: フォーマット対象のファイルが指定されていません。"
    exit 0
fi

echo "🎨 YAML/JSONファイルフォーマットを実行中..."

# フロントエンドディレクトリでローカルPrettierを確認
if [ -x "frontend/node_modules/.bin/prettier" ]; then
    echo "   📦 フロントエンドのローカルPrettierを使用"
    for file in "$@"; do
        echo "   💅 フォーマット対象: \"$file\""
        ./frontend/node_modules/.bin/prettier --write "$file" || {
            echo "❌ エラー: Prettierの実行に失敗しました: $file"
            exit 1
        }
    done
    echo "✅ YAML/JSONフォーマット完了 ($# ファイル処理)"
else
    echo "   🌐 npx Prettierを使用（ローカル版が見つからない）"
    for file in "$@"; do
        echo "   💅 フォーマット対象: \"$file\""
        npx prettier --write "$file" || {
            echo "❌ エラー: Prettierの実行に失敗しました: $file"
            echo "   フロントエンドディレクトリで 'npm install' を実行してください。"
            exit 1
        }
    done
    echo "✅ YAML/JSONフォーマット完了 ($# ファイル処理)"
fi
