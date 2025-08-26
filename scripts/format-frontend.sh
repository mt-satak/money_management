#!/usr/bin/env bash
# Frontend code formatting script for pre-commit hook using Prettier

# エラーハンドリングの有効化
set -e

# フロントエンドディレクトリの存在チェック
if [ ! -d "frontend" ]; then
    echo "❌ エラー: frontendディレクトリが見つかりません。"
    exit 1
fi

# package.jsonの存在チェック
if [ ! -f "frontend/package.json" ]; then
    echo "❌ エラー: frontend/package.jsonが見つかりません。"
    exit 1
fi

# ファイルが指定されているかチェック
if [ $# -eq 0 ]; then
    echo "⚠️  警告: フォーマット対象のファイルが指定されていません。"
    exit 0
fi

echo "🎨 フロントエンドコードフォーマットを実行中..."

# フロントエンドディレクトリに移動
cd frontend || {
    echo "❌ エラー: frontendディレクトリに移動できませんでした。"
    exit 1
}

# 引数として渡されたファイルパスから frontend/ プレフィックスを除去
files=()
for file in "$@"; do
    # frontend/ プレフィックスを除去して相対パスに変換
    rel_file="${file#frontend/}"
    if [ -f "$rel_file" ]; then
        files+=("$rel_file")
        echo "   💅 フォーマット対象: $rel_file"
    else
        echo "⚠️  警告: ファイルが存在しません: $file"
    fi
done

# ファイルがある場合のみPrettierを実行
if [ ${#files[@]} -gt 0 ]; then
    echo "   🚀 Prettierを実行中..."
    npx prettier --write "${files[@]}" || {
        echo "❌ エラー: Prettierの実行に失敗しました。"
        echo "   フロントエンドディレクトリで 'npm install' を実行してください。"
        exit 1
    }
    echo "✅ フロントエンドコードフォーマット完了 (${#files[@]}ファイル処理)"
else
    echo "⚠️  処理対象のファイルがありませんでした。"
fi
