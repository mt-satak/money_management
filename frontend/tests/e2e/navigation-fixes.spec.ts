import { test, expect } from '@playwright/test'

test.describe('ナビゲーション修正の検証', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3000')
  })

  test('ログアウト時にURLがルートにリセットされる', async ({ page }) => {
    // ログイン
    await page.fill('#accountId', 'e2e_test_user')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')

    // ログイン後、家計簿一覧画面に遷移することを確認
    await expect(page).toHaveURL('http://localhost:3000/bills')
    await expect(page.getByText('月次家計簿一覧')).toBeVisible()

    // 家計簿詳細に遷移（URLを変更）
    const billLink = page.locator('a[href*="/bills/"]').first()
    if (await billLink.count() > 0) {
      await billLink.click()
      // 詳細ページのURLを確認
      await expect(page.url()).toMatch(/\/bills\/\d+\/\d+/)
    }

    // ログアウト実行
    await page.getByRole('button', { name: 'ログアウト' }).click()

    // ログアウト後はルート（/）にリダイレクトされることを確認
    await expect(page).toHaveURL('http://localhost:3000/')
    await expect(page.getByText('家計簿アプリ')).toBeVisible()
    await expect(page.getByText('アカウントにログインしてください')).toBeVisible()
  })

  test('ログイン成功時に必ず/billsにリダイレクトされる', async ({ page }) => {
    // 直接詳細ページのURLにアクセスを試みる（未認証なのでログインページが表示される）
    await page.goto('http://localhost:3000/bills/2024/12')

    // ログインページが表示されることを確認
    await expect(page.getByText('アカウントにログインしてください')).toBeVisible()

    // ログイン実行
    await page.fill('#accountId', 'e2e_test_user')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')

    // ログイン後は詳細ページではなく家計簿一覧（/bills）にリダイレクトされることを確認
    await expect(page).toHaveURL('http://localhost:3000/bills')
    await expect(page.getByText('月次家計簿一覧')).toBeVisible()
  })

  test('新規登録成功時にも/billsにリダイレクトされる', async ({ page }) => {
    // 新規登録モードに切り替え
    await page.getByText('アカウントをお持ちでない方はこちら').click()

    // 新規登録フォームが表示されることを確認
    await expect(page.getByText('アカウントを新規作成')).toBeVisible()

    // 新規登録フォームに入力
    const timestamp = Date.now()
    await page.fill('#name', 'テストユーザー2')
    await page.fill('#accountId', `test_user_${timestamp}`)
    await page.fill('#password', 'password123')
    await page.fill('#confirmPassword', 'password123')

    // 新規登録実行
    await page.getByRole('button', { name: 'アカウントを作成' }).click()

    // 登録後は家計簿一覧（/bills）にリダイレクトされることを確認
    await expect(page).toHaveURL('http://localhost:3000/bills')
    await expect(page.getByText('月次家計簿一覧')).toBeVisible()
  })

  test('未認証状態では任意のパスでログインページが表示される', async ({ page }) => {
    const testPaths = [
      '/bills',
      '/bills/2024/12',
      '/nonexistent-path',
      '/some/random/path'
    ]

    for (const path of testPaths) {
      await page.goto(`http://localhost:3000${path}`)

      // どのパスでもログインページが表示されることを確認
      await expect(page.getByText('家計簿アプリ')).toBeVisible()
      await expect(page.getByText('アカウントにログインしてください')).toBeVisible()
      await expect(page.locator('#accountId')).toBeVisible()
    }
  })

  test('認証済み状態では認証が必要なページにアクセス可能', async ({ page }) => {
    // ログイン
    await page.fill('#accountId', 'e2e_test_user')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')

    // 家計簿一覧にリダイレクトされることを確認
    await expect(page).toHaveURL('http://localhost:3000/bills')

    // 認証済みユーザー向けのコンテンツが表示されることを確認
    await expect(page.getByText('💰 家計簿アプリ')).toBeVisible()
    await expect(page.getByText('こんにちは、')).toBeVisible()
    await expect(page.getByText('月次家計簿一覧')).toBeVisible()
  })

  test('ログイン→ログアウト→再ログインの完全フロー', async ({ page }) => {
    // 1. 最初のログイン
    await page.fill('#accountId', 'e2e_test_user')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')

    await expect(page).toHaveURL('http://localhost:3000/bills')
    await expect(page.getByText('月次家計簿一覧')).toBeVisible()

    // 2. 詳細ページに遷移（URLを変更）
    const billLink = page.locator('a[href*="/bills/"]').first()
    if (await billLink.count() > 0) {
      await billLink.click()
      await expect(page.url()).toMatch(/\/bills\/\d+\/\d+/)
    }

    // 3. ログアウト
    await page.getByRole('button', { name: 'ログアウト' }).click()
    await expect(page).toHaveURL('http://localhost:3000/')
    await expect(page.getByText('アカウントにログインしてください')).toBeVisible()

    // 4. 再ログイン
    await page.fill('#accountId', 'e2e_test_user')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')

    // 5. 再ログイン後は家計簿一覧にリダイレクトされる（前回の詳細ページではない）
    await expect(page).toHaveURL('http://localhost:3000/bills')
    await expect(page.getByText('月次家計簿一覧')).toBeVisible()
  })

  test('認証チェックによるリダイレクト確認', async ({ page }) => {
    // 一度ログインして認証状態にする
    await page.fill('#accountId', 'e2e_test_user')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('http://localhost:3000/bills')

    // localStorageからトークンを削除してログアウト状態をシミュレート
    await page.evaluate(() => localStorage.removeItem('token'))

    // ページをリロードして認証状態をクリア
    await page.reload()

    // 認証が必要なページ（家計簿詳細）に直接アクセス
    await page.goto('http://localhost:3000/bills/2024/12')

    // 認証チェックによりログインページが表示されることを確認
    await expect(page.getByText('アカウントにログインしてください')).toBeVisible()
  })
})
