import { test, expect } from "@playwright/test";

test.describe("基本ユーザーフロー E2Eテスト", () => {
  test.describe("ログインと基本操作", () => {
    test("既存ユーザーのログインとページナビゲーション", async ({ page }) => {
      // ログイン
      await page.goto("/login");
      await page.fill("#accountId", "e2etest");
      await page.fill("#password", "password");
      await page.click('button[type="submit"]');

      // ログイン成功を確認
      await page.waitForTimeout(3000);

      // 家計簿一覧ページに移動
      await page.goto("/bills");
      await page.waitForLoadState("networkidle");

      // ページが表示されることを確認
      await expect(page.locator("h1")).toBeVisible();
    });

    test("新規登録フォームのバリデーション", async ({ page }) => {
      await page.goto("/login");

      // 新規登録モードに切り替え
      await page.click("text=アカウントをお持ちでない方はこちら");

      // パスワード不一致エラー
      await page.fill("#name", "テストユーザー");
      await page.fill("#accountId", "test_user_mismatch");
      await page.fill("#password", "password123");
      await page.fill("#confirmPassword", "password456"); // 異なるパスワード

      await page.click('button[type="submit"]');

      // エラーメッセージが表示されることを確認
      await expect(page.locator(".bg-red-50")).toBeVisible();
      await expect(page.locator("text=パスワードが一致しません")).toBeVisible();

      // パスワード短すぎるエラー
      await page.fill("#password", "123"); // 3文字（短すぎる）
      await page.fill("#confirmPassword", "123");

      await page.click('button[type="submit"]');

      // エラーメッセージが表示されることを確認
      await expect(
        page.locator("text=パスワードは6文字以上で入力してください"),
      ).toBeVisible();
    });

    test("存在しないアカウントでのログインエラー", async ({ page }) => {
      await page.goto("/login");

      // 存在しないアカウントでログイン試行
      await page.fill("#accountId", "nonexistent_user");
      await page.fill("#password", "wrongpassword");

      await page.click('button[type="submit"]');

      // ログインエラーが表示されることを確認
      await page.waitForSelector(".bg-red-50", { timeout: 10000 });
      await expect(page.locator(".bg-red-50")).toBeVisible();
    });
  });
});
