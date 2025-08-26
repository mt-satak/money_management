#!/bin/bash
# Frontend code formatting script for pre-commit hook using Prettier

# フロントエンドディレクトリに移動してから処理
cd frontend || exit 1

# 引数として渡されたファイルパスから frontend/ プレフィックスを除去
files=()
for file in "$@"; do
    # frontend/ プレフィックスを除去して相対パスに変換
    files+=("${file#frontend/}")
done

# ファイルがある場合のみPrettierを実行
if [ ${#files[@]} -gt 0 ]; then
    npx prettier --write "${files[@]}"
fi
