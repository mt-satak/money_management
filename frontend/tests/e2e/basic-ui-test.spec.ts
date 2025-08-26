import { test, expect } from "@playwright/test";

test.describe("基本UIスモークテスト", () => {
  test("アプリケーション基本動作確認", async ({ page }) => {
    console.log("🔍 アプリケーションのスモークテスト開始...");

    // まずはトップページに移動
    await page.goto("/");
    console.log("✅ トップページに移動完了");

    // 基本的なページが読み込まれることを確認
    const response = await page.goto("/");
    expect(response?.status()).toBe(200);
    console.log("✅ レスポンスステータス200確認");

    // ログインページに移動
    await page.goto("/login");
    console.log("✅ ログインページに移動完了");

    // 基本的なDOM要素の存在確認（テキストベースの確認）
    await expect(page.locator("body")).toBeVisible();
    console.log("✅ bodyタグが表示されています");

    // ページタイトルの確認
    const title = await page.title();
    expect(title.length).toBeGreaterThan(0);
    console.log(`✅ ページタイトル: "${title}"`);

    console.log("🎉 スモークテスト完了");
  });
});
