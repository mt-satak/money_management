import { test, expect } from "@playwright/test";

test.describe("支払者家計簿表示機能", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("http://localhost:3000");
  });

  test("支払者として指定された家計簿が正しく表示される", async ({ page }) => {
    // ログイン（e2e_test_userとして）
    await page.fill("#accountId", "e2e_test_user");
    await page.fill("#password", "password123");
    await page.click('button[type="submit"]');

    // 家計簿一覧に遷移することを確認
    await expect(page).toHaveURL("http://localhost:3000/bills");
    await expect(page.getByText("月次家計簿一覧")).toBeVisible();

    // 新しい家計簿を作成（e2e_test_userが請求者、e2e_payerが支払者）
    await page.click("text=+ 新規作成");

    // フォームに入力（支払者を選択）
    const payerSelect = page.locator("select").nth(1); // 2番目のselectが支払者
    await payerSelect.selectOption({ label: "E2Eテスト支払者" });
    await page.click('button[type="submit"]');

    // 家計簿が作成されることを確認
    await expect(page.getByText("月の家計簿")).toBeVisible();
    await expect(page.getByText("請求者")).toBeVisible();

    // ログアウト
    await page.getByRole("button", { name: "ログアウト" }).click();
    await expect(page).toHaveURL("http://localhost:3000/");

    // 支払者としてログイン（e2e_payer）
    await page.fill("#accountId", "e2e_payer");
    await page.fill("#password", "password123");
    await page.click('button[type="submit"]');

    // 家計簿一覧で支払者として指定された家計簿が表示されることを確認
    await expect(page).toHaveURL("http://localhost:3000/bills");
    await expect(page.getByText("月次家計簿一覧")).toBeVisible();
    await expect(page.getByText("月の家計簿")).toBeVisible();
    await expect(page.getByText("支払者")).toBeVisible();

    // 支払者は削除ボタンが表示されないことを確認
    await expect(page.locator("text=削除")).not.toBeVisible();
  });

  test("請求中ステータスの支払者家計簿のスタイリング", async ({ page }) => {
    // 請求者としてログイン
    await page.fill("#accountId", "e2e_test_user");
    await page.fill("#password", "password123");
    await page.click('button[type="submit"]');

    // 家計簿詳細に遷移（既存の家計簿があると仮定）
    const billLink = page.locator('a[href*="/bills/"]').first();
    if ((await billLink.count()) > 0) {
      await billLink.click();

      // 項目を追加して請求を確定
      await page.fill(
        'input[placeholder="項目名（例: 食費、交通費）"]',
        "テスト項目",
      );
      await page.fill('input[placeholder="金額"]', "1000");
      await page.click("text=項目を保存");

      // 少し待機
      await page.waitForTimeout(1000);

      // 請求を確定
      const requestButton = page.locator("text=請求を確定する");
      if (await requestButton.isVisible()) {
        await requestButton.click();
        await expect(page.getByText("請求済み")).toBeVisible();
      }

      // 家計簿一覧に戻る
      await page.click("text=← 家計簿一覧に戻る");

      // ログアウト
      await page.getByRole("button", { name: "ログアウト" }).click();

      // 支払者としてログイン
      await page.fill("#accountId", "e2e_payer");
      await page.fill("#password", "password123");
      await page.click('button[type="submit"]');

      // 請求中の家計簿が黄色背景で表示されることを確認
      await expect(page.locator(".bg-yellow-50").first()).toBeVisible();
      await expect(page.getByText("支払者（要支払い）")).toBeVisible();
    }
  });

  test("支払済みステータスの支払者家計簿のスタイリング", async ({ page }) => {
    // 支払者としてログイン
    await page.fill("#accountId", "e2e_payer");
    await page.fill("#password", "password123");
    await page.click('button[type="submit"]');

    // 請求中の家計簿詳細に遷移
    const requestedBillLink = page
      .locator('a[href*="/bills/"]:has-text("要支払い")')
      .first();
    if ((await requestedBillLink.count()) > 0) {
      await requestedBillLink.click();

      // 支払いを確定
      const payButton = page.locator("text=支払いを確定する");
      if (await payButton.isVisible()) {
        await payButton.click();
        await expect(page.getByText("支払済み")).toBeVisible();
      }

      // 家計簿一覧に戻る
      await page.click("text=← 家計簿一覧に戻る");

      // 支払済みの家計簿が緑色背景で表示されることを確認
      await expect(page.locator(".bg-green-50").first()).toBeVisible();
      await expect(page.getByText("支払者（支払済み）")).toBeVisible();
    }
  });

  test("支払者の家計簿詳細アクセステスト", async ({ page }) => {
    // 支払者としてログイン
    await page.fill("#accountId", "e2e_payer");
    await page.fill("#password", "password123");
    await page.click('button[type="submit"]');

    // 支払者の家計簿詳細にアクセス
    const billLink = page.locator('a[href*="/bills/"]').first();
    if ((await billLink.count()) > 0) {
      await billLink.click();

      // 詳細ページが表示されることを確認
      await expect(page.getByText("月の家計簿")).toBeVisible();
      await expect(page.getByText("基本情報")).toBeVisible();

      // 支払者は項目編集ができないことを確認
      const addItemButton = page.locator("text=+ 項目を追加");
      await expect(addItemButton).not.toBeVisible();

      // 項目保存ボタンも表示されないことを確認
      const saveButton = page.locator("text=項目を保存");
      await expect(saveButton).not.toBeVisible();
    }
  });
});
