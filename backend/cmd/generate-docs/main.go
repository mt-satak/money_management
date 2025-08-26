// ========================================
// API仕様書生成コマンドラインツール
// 契約テストからMarkdown/OpenAPI仕様書を自動生成
// ========================================

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	testmocks "money_management/internal/testing"
)

func main() {
	// コマンドライン引数の定義
	var (
		format = flag.String("format", "markdown", "出力形式 (markdown, openapi-json, openapi-yaml)")
		output = flag.String("output", "", "出力ファイルパス (空の場合は標準出力)")
		help   = flag.Bool("help", false, "ヘルプを表示")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// 仕様書を生成
	var content string
	var err error
	var defaultFilename string

	switch *format {
	case "markdown", "md":
		content, err = testmocks.GenerateMarkdownDoc()
		defaultFilename = "API_SPECIFICATION.md"
	case "openapi-json", "json":
		content, err = testmocks.GenerateOpenAPIJSON()
		defaultFilename = "openapi.json"
	case "openapi-yaml", "yaml":
		content, err = testmocks.GenerateOpenAPIYAML()
		defaultFilename = "openapi.yaml"
	default:
		fmt.Fprintf(os.Stderr, "❌ エラー: 未対応の形式 '%s'\n", *format)
		fmt.Fprintf(os.Stderr, "対応形式: markdown, openapi-json, openapi-yaml\n")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 仕様書生成エラー: %v\n", err)
		os.Exit(1)
	}

	// 出力先を決定
	outputPath := *output
	if outputPath == "" {
		// 標準出力に出力
		fmt.Print(content)
		return
	}

	// ファイルに出力
	if outputPath == "auto" {
		outputPath = defaultFilename
	}

	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(outputPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ ディレクトリ作成エラー: %v\n", err)
			os.Exit(1)
		}
	}

	// ファイルに書き込み
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ ファイル書き込みエラー: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ API仕様書を生成しました: %s\n", outputPath)
	fmt.Printf("📊 形式: %s\n", *format)
	fmt.Printf("📄 サイズ: %d bytes\n", len(content))
}

func showHelp() {
	fmt.Print(`
API仕様書生成ツール - Money Management API

使用方法:
  go run cmd/generate-docs/main.go [オプション]

オプション:
  -format string    出力形式 (markdown, openapi-json, openapi-yaml) (default: markdown)
  -output string    出力ファイルパス (空の場合は標準出力, "auto"で自動ファイル名)
  -help            このヘルプを表示

使用例:
  # Markdownをコンソールに出力
  go run cmd/generate-docs/main.go

  # Markdownをファイルに出力
  go run cmd/generate-docs/main.go -format markdown -output API_SPECIFICATION.md

  # OpenAPI JSONを生成
  go run cmd/generate-docs/main.go -format openapi-json -output openapi.json

  # 自動ファイル名で出力
  go run cmd/generate-docs/main.go -format markdown -output auto

契約テスト基盤:
  本ツールは契約テスト（Contract Testing）により自動生成される仕様書を出力します。
  実装との整合性が保証されており、APIの変更は即座に仕様書に反映されます。

`)
}
