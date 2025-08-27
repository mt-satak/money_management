#!/bin/bash

# ========================================
# GitHub Branch Protection Setup Script
# mainブランチのプロテクション設定を自動化
# ========================================

set -e  # エラー時に即座に終了

# カラー出力用
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ログ出力関数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# GitHub CLI の確認
check_gh_cli() {
    if ! command -v gh &> /dev/null; then
        log_error "GitHub CLI (gh) がインストールされていません"
        log_info "インストール: https://cli.github.com/"
        exit 1
    fi

    # GitHub認証の確認
    if ! gh auth status &> /dev/null; then
        log_error "GitHub認証が必要です"
        log_info "実行: gh auth login"
        exit 1
    fi

    log_success "GitHub CLI認証確認済み"
}

# リポジトリ情報の取得
get_repo_info() {
    # git remoteからリポジトリ名を取得
    REPO_URL=$(git remote get-url origin 2>/dev/null || echo "")

    if [[ -z "$REPO_URL" ]]; then
        log_error "Gitリポジトリではないか、originリモートが設定されていません"
        exit 1
    fi

    # GitHub URLからowner/repoを抽出
    if [[ "$REPO_URL" =~ github\.com[:/]([^/]+)/([^/.]+) ]]; then
        REPO_OWNER="${BASH_REMATCH[1]}"
        REPO_NAME="${BASH_REMATCH[2]}"
        REPO_FULL_NAME="${REPO_OWNER}/${REPO_NAME}"
    else
        log_error "GitHubリポジトリのURLを解析できません: $REPO_URL"
        exit 1
    fi

    log_info "対象リポジトリ: $REPO_FULL_NAME"
}

# ブランチプロテクション設定
setup_branch_protection() {
    log_info "mainブランチのプロテクション設定を適用中..."

    # ブランチプロテクション設定のJSON
    PROTECTION_JSON='{
  "required_status_checks": {
    "strict": true,
    "contexts": ["pr-tests"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}'

    # API実行
    if gh api "repos/$REPO_FULL_NAME/branches/main/protection" \
        -X PUT \
        -H "Accept: application/vnd.github.v3+json" \
        --input - <<< "$PROTECTION_JSON" > /dev/null; then
        log_success "ブランチプロテクション設定完了"
    else
        log_error "ブランチプロテクション設定に失敗しました"
        exit 1
    fi
}

# 自動ブランチ削除設定
setup_auto_branch_deletion() {
    log_info "PRマージ後の自動ブランチ削除を設定中..."

    # リポジトリ設定のJSON
    REPO_SETTINGS_JSON='{
  "delete_branch_on_merge": true
}'

    # API実行
    if gh api "repos/$REPO_FULL_NAME" \
        -X PATCH \
        -H "Accept: application/vnd.github.v3+json" \
        --input - <<< "$REPO_SETTINGS_JSON" > /dev/null; then
        log_success "自動ブランチ削除設定完了"
    else
        log_error "自動ブランチ削除設定に失敗しました"
        exit 1
    fi
}

# 設定確認
verify_settings() {
    log_info "設定内容を確認中..."

    # ブランチプロテクション確認
    PROTECTION_STATUS=$(gh api "repos/$REPO_FULL_NAME/branches/main/protection" 2>/dev/null || echo "null")

    if [[ "$PROTECTION_STATUS" != "null" ]]; then
        log_success "✅ ブランチプロテクション: 有効"
        log_info "   └─ PR必須: 有効"
        log_info "   └─ ステータスチェック: pr-tests"
        log_info "   └─ 管理者例外: 無効"
        log_info "   └─ フォースプッシュ: 無効"
    else
        log_warning "ブランチプロテクション設定の確認に失敗"
    fi

    # リポジトリ設定確認
    DELETE_BRANCH_ON_MERGE=$(gh api "repos/$REPO_FULL_NAME" --jq '.delete_branch_on_merge' 2>/dev/null || echo "null")

    if [[ "$DELETE_BRANCH_ON_MERGE" == "true" ]]; then
        log_success "✅ 自動ブランチ削除: 有効"
    else
        log_warning "自動ブランチ削除設定の確認に失敗"
    fi
}

# メイン処理
main() {
    echo "========================================="
    echo "GitHub Branch Protection Setup Script"
    echo "========================================="
    echo ""

    check_gh_cli
    get_repo_info

    echo ""
    log_info "以下の設定を適用します:"
    log_info "• PR必須（直接pushを禁止）"
    log_info "• ステータスチェック「pr-tests」必須"
    log_info "• 管理者例外なし"
    log_info "• フォースプッシュ・削除禁止"
    log_info "• PRマージ後の自動ブランチ削除"
    echo ""

    read -p "続行しますか？ (y/N): " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "処理をキャンセルしました"
        exit 0
    fi

    echo ""
    setup_branch_protection
    setup_auto_branch_deletion

    echo ""
    verify_settings

    echo ""
    log_success "🎉 ブランチプロテクション設定が完了しました！"
    log_info "mainブランチへの直接pushは今後禁止され、すべてPR経由でのマージが必要になります。"
}

# スクリプト実行
main "$@"
