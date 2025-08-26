import { describe, it, expect } from "vitest";
import {
  isDuplicateBill,
  generateDuplicateWarningMessage,
  generateCreateButtonText,
} from "../../utils/billUtils";
import { createMockBill, createBillsForDuplicateTest } from "./test-utils";
import { BillResponse } from "../../types";

describe("isDuplicateBill 関数", () => {
  describe("基本的な重複チェック", () => {
    it("同一年月の家計簿が存在する場合はtrueを返す", () => {
      const bills = createBillsForDuplicateTest();
      const result = isDuplicateBill(bills, 2025, 8);
      expect(result).toBe(true);
    });

    it("存在しない年月の場合はfalseを返す", () => {
      const bills = createBillsForDuplicateTest();
      const result = isDuplicateBill(bills, 2025, 7);
      expect(result).toBe(false);
    });

    it("bills配列が空の場合はfalseを返す", () => {
      const bills: BillResponse[] = [];
      const result = isDuplicateBill(bills, 2025, 8);
      expect(result).toBe(false);
    });
  });

  describe("エッジケース", () => {
    it("年が同じで月が異なる場合はfalseを返す", () => {
      const bills = createBillsForDuplicateTest();
      const result = isDuplicateBill(bills, 2025, 7); // 8月,9月は存在するが7月はない
      expect(result).toBe(false);
    });

    it("月が同じで年が異なる場合はfalseを返す", () => {
      const bills = createBillsForDuplicateTest();
      const result = isDuplicateBill(bills, 2026, 8); // 2025年8月は存在するが2026年8月はない
      expect(result).toBe(false);
    });

    it("複数の家計簿がある中で正確に重複を検出する", () => {
      const bills = [
        createMockBill({ id: 1, year: 2024, month: 1 }),
        createMockBill({ id: 2, year: 2024, month: 2 }),
        createMockBill({ id: 3, year: 2025, month: 1 }),
        createMockBill({ id: 4, year: 2025, month: 8 }),
      ];

      expect(isDuplicateBill(bills, 2024, 1)).toBe(true);
      expect(isDuplicateBill(bills, 2025, 8)).toBe(true);
      expect(isDuplicateBill(bills, 2024, 3)).toBe(false);
      expect(isDuplicateBill(bills, 2025, 9)).toBe(false);
    });
  });

  describe("境界値テスト", () => {
    it("年の境界値で正しく動作する", () => {
      const bills = [
        createMockBill({ year: 2024, month: 12 }),
        createMockBill({ year: 2025, month: 1 }),
      ];

      expect(isDuplicateBill(bills, 2024, 12)).toBe(true);
      expect(isDuplicateBill(bills, 2025, 1)).toBe(true);
      expect(isDuplicateBill(bills, 2024, 11)).toBe(false);
      expect(isDuplicateBill(bills, 2025, 2)).toBe(false);
    });

    it("月の境界値で正しく動作する", () => {
      const bills = [
        createMockBill({ year: 2025, month: 1 }),
        createMockBill({ year: 2025, month: 12 }),
      ];

      expect(isDuplicateBill(bills, 2025, 1)).toBe(true);
      expect(isDuplicateBill(bills, 2025, 12)).toBe(true);
      expect(isDuplicateBill(bills, 2025, 0)).toBe(false); // 無効な月
      expect(isDuplicateBill(bills, 2025, 13)).toBe(false); // 無効な月
    });
  });

  describe("パフォーマンステスト", () => {
    it("大量のデータでも効率的に動作する", () => {
      // 1000件の家計簿データを生成
      const bills: BillResponse[] = [];
      for (let year = 2020; year < 2025; year++) {
        for (let month = 1; month <= 12; month++) {
          bills.push(
            createMockBill({
              id: bills.length + 1,
              year,
              month,
            }),
          );
        }
      }

      const startTime = performance.now();
      const result = isDuplicateBill(bills, 2023, 6);
      const endTime = performance.now();

      expect(result).toBe(true);
      expect(endTime - startTime).toBeLessThan(10); // 10ms以内で完了することを確認
    });
  });
});

describe("generateDuplicateWarningMessage 関数", () => {
  it("正しい警告メッセージを生成する", () => {
    const message = generateDuplicateWarningMessage(2025, 8);
    expect(message).toBe("⚠️ 2025年8月の家計簿は既に存在します");
  });

  it("異なる年月でも正しくメッセージを生成する", () => {
    expect(generateDuplicateWarningMessage(2024, 12)).toBe(
      "⚠️ 2024年12月の家計簿は既に存在します",
    );
    expect(generateDuplicateWarningMessage(2026, 1)).toBe(
      "⚠️ 2026年1月の家計簿は既に存在します",
    );
  });

  it("境界値でも正しくメッセージを生成する", () => {
    expect(generateDuplicateWarningMessage(2025, 1)).toBe(
      "⚠️ 2025年1月の家計簿は既に存在します",
    );
    expect(generateDuplicateWarningMessage(2025, 12)).toBe(
      "⚠️ 2025年12月の家計簿は既に存在します",
    );
  });
});

describe("generateCreateButtonText 関数", () => {
  it("作成中の場合は作成中メッセージを返す", () => {
    const text = generateCreateButtonText(true, false);
    expect(text).toBe("作成中...");
  });

  it("重複している場合は重複エラーメッセージを返す", () => {
    const text = generateCreateButtonText(false, true);
    expect(text).toBe("重複のため作成不可");
  });

  it("通常状態の場合は作成メッセージを返す", () => {
    const text = generateCreateButtonText(false, false);
    expect(text).toBe("作成");
  });

  it("作成中かつ重複の場合は作成中を優先する", () => {
    const text = generateCreateButtonText(true, true);
    expect(text).toBe("作成中...");
  });

  describe("状態の組み合わせパターン", () => {
    const testCases = [
      { creating: false, isDuplicate: false, expected: "作成" },
      { creating: true, isDuplicate: false, expected: "作成中..." },
      { creating: false, isDuplicate: true, expected: "重複のため作成不可" },
      { creating: true, isDuplicate: true, expected: "作成中..." },
    ];

    testCases.forEach(({ creating, isDuplicate, expected }) => {
      it(`creating: ${creating}, isDuplicate: ${isDuplicate} の場合は "${expected}" を返す`, () => {
        const result = generateCreateButtonText(creating, isDuplicate);
        expect(result).toBe(expected);
      });
    });
  });
});
