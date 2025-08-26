import { test, expect } from '@playwright/test';

test.describe('基本UI テスト', () => {
  test('ログインページの基本要素が表示される', async ({ page }) => {
    await page.goto('/login');

    // ページのタイトル確認
    await expect(page).toHaveTitle(/家計簿/);

    // ログインフォームの要素確認
    await expect(page.locator('h2:has-text("家計簿アプリ")')).toBeVisible();
    await expect(page.locator('#accountId')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();

    // プレースホルダーテキストの確認
    await expect(page.locator('#accountId')).toHaveAttribute('placeholder', '英数字とアンダースコア、3-20文字');
    await expect(page.locator('#password')).toHaveAttribute('placeholder', '6文字以上');
  });

  test('新規登録モードへの切り替え', async ({ page }) => {
    await page.goto('/login');

    // 新規登録モードに切り替え
    await page.click('text=アカウントをお持ちでない方はこちら');

    // 新規登録フォームが表示される
    await expect(page.locator('text=アカウントを新規作成')).toBeVisible();
    await expect(page.locator('#name')).toBeVisible();
    await expect(page.locator('#confirmPassword')).toBeVisible();

    // ログインモードに戻る
    await page.click('text=すでにアカウントをお持ちですか？ログインはこちら');

    // ログインフォームに戻る
    await expect(page.locator('text=アカウントにログインしてください')).toBeVisible();
    await expect(page.locator('#name')).not.toBeVisible();
  });

  test('フォームバリデーション表示', async ({ page }) => {
    await page.goto('/login');

    // 空でSubmit
    await page.click('button[type="submit"]');

    // HTML5バリデーションが機能する
    // （実際のエラーメッセージは実装に依存）

    // アカウントIDの最小長チェック
    await page.fill('#accountId', 'ab'); // 2文字（最小3文字未満）
    await page.fill('#password', '123456');

    // HTML5バリデーションでminlength制約が機能する
    const accountIdInput = page.locator('#accountId');
    await expect(accountIdInput).toHaveAttribute('minLength', '3');
    await expect(accountIdInput).toHaveAttribute('maxLength', '20');
    await expect(accountIdInput).toHaveAttribute('pattern', '[a-zA-Z0-9_]+');
  });
});
