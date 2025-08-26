import { test, expect } from '@playwright/test';

test.describe('基本ビジュアルテスト', () => {

  test.describe('ログインページの基本表示', () => {
    test('ログインページの初期状態', async ({ page }) => {
      await page.goto('/login');
      await page.waitForLoadState('networkidle');

      // 基本要素の表示確認
      await expect(page.locator('h2')).toBeVisible();
      await expect(page.locator('#accountId')).toBeVisible();
      await expect(page.locator('#password')).toBeVisible();
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });

    test('新規登録モードの表示', async ({ page }) => {
      await page.goto('/login');
      await page.waitForLoadState('networkidle');

      // 新規登録モードに切り替え
      await page.click('text=アカウントをお持ちでない方はこちら');
      await page.waitForTimeout(500);

      // 新規登録フィールドが表示されることを確認
      await expect(page.locator('#name')).toBeVisible();
      await expect(page.locator('#confirmPassword')).toBeVisible();
    });

    test('ログインエラー表示', async ({ page }) => {
      await page.goto('/login');
      await page.waitForLoadState('networkidle');

      // 無効なログイン情報で試行
      await page.fill('#accountId', 'invalid_user');
      await page.fill('#password', 'wrong_password');
      await page.click('button[type="submit"]');

      // エラーメッセージ表示を確認
      await page.waitForSelector('.bg-red-50', { timeout: 10000 });
      await expect(page.locator('.bg-red-50')).toBeVisible();
    });
  });

  test.describe('家計簿一覧ページの基本表示', () => {
    test.beforeEach(async ({ page }) => {
      // ログイン
      await page.goto('/login');
      await page.fill('#accountId', 'e2etest');
      await page.fill('#password', 'password');
      await page.click('button[type="submit"]');
      await page.waitForTimeout(3000);
      await page.goto('/bills');
      await page.waitForLoadState('networkidle');
    });

    test('家計簿一覧ページの基本要素', async ({ page }) => {
      // 基本要素の確認
      await expect(page.locator('h1')).toBeVisible();
      await expect(page.locator('button:has-text("+ 新規作成")')).toBeVisible();
    });

    test('新規作成モーダル表示', async ({ page }) => {
      // 新規作成ボタンをクリック
      await page.click('button:has-text("+ 新規作成")');
      await page.waitForSelector('.fixed.inset-0');

      // モーダル内の基本要素確認
      await expect(page.locator('.fixed.inset-0')).toBeVisible();
      await expect(page.locator('input[type="number"]')).toBeVisible();
      await expect(page.locator('select').first()).toBeVisible();
    });
  });

  test.describe('レスポンシブデザインの基本確認', () => {
    test('モバイルビューでのログインページ', async ({ page }) => {
      // モバイルビューポートに設定
      await page.setViewportSize({ width: 375, height: 667 });

      await page.goto('/login');
      await page.waitForLoadState('networkidle');

      // モバイルでの基本要素確認
      await expect(page.locator('.max-w-md')).toBeVisible();
      await expect(page.locator('#accountId')).toBeVisible();
      await expect(page.locator('#password')).toBeVisible();
    });

    test('タブレットビューでのログインページ', async ({ page }) => {
      // タブレットビューポートに設定
      await page.setViewportSize({ width: 768, height: 1024 });

      await page.goto('/login');
      await page.waitForLoadState('networkidle');

      // タブレットでの基本要素確認
      await expect(page.locator('h2')).toBeVisible();
      await expect(page.locator('#accountId')).toBeVisible();
    });
  });
});
