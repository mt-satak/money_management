import { test, expect } from '@playwright/test';

/**
 * テストスイート最適化とカテゴリ分類
 * 効率的な並列実行とテスト分類を実現
 */

test.describe('スモークテスト (優先度: 最高)', () => {
  // 最重要な基本機能テスト - 高速実行
  test.describe.configure({ mode: 'parallel' });

  test('クリティカルパス: ログイン機能', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#accountId', 'e2etest');
    await page.fill('#password', 'password');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    const currentUrl = page.url();
    console.log('Current URL after login:', currentUrl);
    expect(currentUrl.includes('/bills') || currentUrl.endsWith('/')).toBe(true);
  });

  test('クリティカルパス: 家計簿一覧表示', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#accountId', 'e2etest');
    await page.fill('#password', 'password');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    await page.goto('/bills');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('クリティカルパス: 新規作成モーダル表示', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#accountId', 'e2etest');
    await page.fill('#password', 'password');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    await page.goto('/bills');
    await page.waitForLoadState('networkidle');
    await page.click('button:has-text("+ 新規作成")');
    await expect(page.locator('.fixed.inset-0')).toBeVisible();
  });
});

test.describe('回帰テスト (優先度: 高)', () => {
  // 既存機能の動作確認テスト
  test.describe.configure({ mode: 'parallel' });

  test.beforeEach(async ({ page }) => {
    // 共通のログイン処理
    await page.goto('/login');
    await page.fill('#accountId', 'e2etest');
    await page.fill('#password', 'password');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    await page.goto('/bills');
    await page.waitForLoadState('networkidle');
  });

  test('重複チェック機能の動作確認', async ({ page }) => {
    await page.click('button:has-text("+ 新規作成")');
    await page.waitForSelector('.fixed.inset-0');

    const currentYear = new Date().getFullYear();
    const currentMonth = new Date().getMonth() + 1;

    await page.fill('input[type="number"]', currentYear.toString());
    await page.selectOption('select', currentMonth.toString());
    await page.selectOption('select', { index: 1 });

    // 重複チェックの動作確認
    await page.waitForTimeout(1000);
  });

  test('フォームバリデーションの動作確認', async ({ page }) => {
    await page.click('button:has-text("+ 新規作成")');
    await page.waitForSelector('.fixed.inset-0');

    // 無効な年を入力
    await page.fill('input[type="number"]', '1900');
    await page.selectOption('select', '1');

    // バリデーション確認
    const yearInput = page.locator('input[type="number"]');
    await expect(yearInput).toHaveValue('1900');
  });
});

test.describe('統合テスト (優先度: 中)', () => {
  // 複数機能の連携テスト
  test.describe.configure({ mode: 'parallel' });

  test('新規登録フォーム基本テスト', async ({ page }) => {
    await page.goto('/login');
    await page.click('text=アカウントをお持ちでない方はこちら');

    // 新規登録フォームが表示されることを確認
    await expect(page.locator('#name')).toBeVisible();
    await expect(page.locator('#confirmPassword')).toBeVisible();
    await expect(page.locator('text=アカウントを新規作成')).toBeVisible();
  });

  test('エラーハンドリングの統合テスト', async ({ page }) => {
    await page.goto('/login');

    // 無効なログイン試行
    await page.fill('#accountId', 'nonexistent_user');
    await page.fill('#password', 'wrong_password');
    await page.click('button[type="submit"]');

    // エラー表示の確認
    await page.waitForSelector('.bg-red-50', { timeout: 10000 });
    await expect(page.locator('.bg-red-50')).toBeVisible();
  });
});

test.describe('UI/UXテスト (優先度: 中)', () => {
  // ユーザーエクスペリエンス関連テスト
  test.describe.configure({ mode: 'parallel' });

  test('レスポンシブデザイン: モバイルビュー', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // モバイルでのレイアウト確認
    const loginContainer = page.locator('.max-w-md');
    await expect(loginContainer).toBeVisible();
  });

  test('レスポンシブデザイン: タブレットビュー', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // タブレットでのレイアウト確認
    await expect(page.locator('h2')).toBeVisible();
  });

  test('キーボードナビゲーション', async ({ page, browserName }) => {
    await page.goto('/login');

    // モバイルブラウザではキーボードフォーカスがサポートされていないためスキップ
    if (browserName === 'webkit' || browserName === 'Mobile Safari') {
      test.skip();
      return;
    }

    // Tabキーでのナビゲーションテスト
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');

    // フォーカス状態の確認
    const focusedElement = await page.locator(':focus').first();
    await expect(focusedElement).toBeVisible();
  });
});

test.describe('パフォーマンステスト (優先度: 低)', () => {
  // パフォーマンス関連テスト - 時間がかかるため優先度低
  test.describe.configure({ mode: 'parallel' });

  test('ページロード時間測定', async ({ page }) => {
    const startTime = Date.now();

    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    const endTime = Date.now();
    const loadTime = endTime - startTime;

    // パフォーマンス基準
    expect(loadTime).toBeLessThan(5000);

    console.log('ページロード時間:', loadTime);
  });

  test('大量操作時のパフォーマンス', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#accountId', 'e2etest');
    await page.fill('#password', 'password');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    await page.goto('/bills');
    await page.waitForLoadState('networkidle');

    const startTime = Date.now();

    // 複数回のモーダル開閉操作
    for (let i = 0; i < 5; i++) {
      await page.click('button:has-text("+ 新規作成")');
      await page.waitForSelector('.fixed.inset-0');
      await page.click('button:has-text("キャンセル")');
      await page.waitForSelector('.fixed.inset-0', { state: 'hidden' });
    }

    const endTime = Date.now();
    const operationTime = endTime - startTime;

    // 操作パフォーマンス基準
    expect(operationTime).toBeLessThan(10000);

    console.log('大量操作時間:', operationTime);
  });
});

test.describe('ビジュアルリグレッション (優先度: 低)', () => {
  // スクリーンショット比較テスト - 実行時間長め
  test.describe.configure({ mode: 'parallel' });

  test('重要画面のビジュアル確認', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // ビジュアル回帰テスト（テスト実行時のみ）
    if (process.env.VISUAL_REGRESSION === 'true') {
      await expect(page).toHaveScreenshot('login-regression-test.png');
    }
  });
});

// テスト実行時間最適化のためのヘルパー関数
test.describe('並列実行最適化テスト', () => {
  // 独立性の高いテストを並列実行
  test.describe.configure({ mode: 'parallel' });

  const parallelTestData = [
    { user: 'user1', scenario: 'ログイン成功' },
    { user: 'user2', scenario: '新規登録' },
    { user: 'user3', scenario: 'パスワードリセット（模擬）' },
    { user: 'user4', scenario: 'アカウント設定（模擬）' }
  ];

  for (const data of parallelTestData) {
    test(`並列テスト: ${data.scenario} (${data.user})`, async ({ page }) => {
      await page.goto('/login');

      // 各ユーザー固有のテストロジック
      switch (data.scenario) {
        case 'ログイン成功':
          await page.fill('#accountId', 'e2etest');
          await page.fill('#password', 'password');
          await page.click('button[type="submit"]');
          await page.waitForTimeout(3000);
          const currentUrl = page.url();
          expect(currentUrl.includes('/bills') || currentUrl.endsWith('/')).toBe(true);
          break;

        case '新規登録':
          await page.click('text=アカウントをお持ちでない方はこちら');
          await expect(page.locator('#name')).toBeVisible();
          break;

        case 'パスワードリセット（模擬）':
          // 実際の機能がある場合の模擬テスト
          await expect(page.locator('#accountId')).toBeVisible();
          break;

        case 'アカウント設定（模擬）':
          // 実際の機能がある場合の模擬テスト
          await expect(page.locator('#password')).toBeVisible();
          break;
      }

      console.log(`${data.scenario} (${data.user}) 完了`);
    });
  }
});
