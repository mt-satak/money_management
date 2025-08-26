import React, { ReactElement } from "react";
import { render, RenderOptions } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { BillResponse, User } from "../../types";

// カスタムレンダー関数（Router付き）
const AllTheProviders = ({ children }: { children: React.ReactNode }) => {
  return <BrowserRouter>{children}</BrowserRouter>;
};

const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">,
) => render(ui, { wrapper: AllTheProviders, ...options });

// テストデータファクトリ
export const createMockUser = (overrides?: Partial<User>): User => ({
  id: 1,
  name: "テストユーザー",
  account_id: "test_user",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  ...overrides,
});

export const createMockBill = (
  overrides?: Partial<BillResponse>,
): BillResponse => ({
  id: 1,
  year: 2025,
  month: 8,
  requester_id: 1,
  payer_id: 2,
  status: "pending",
  request_date: undefined,
  payment_date: undefined,
  requester: createMockUser({ id: 1, name: "テスト請求者" }),
  payer: createMockUser({ id: 2, name: "テスト支払者" }),
  items: [],
  created_at: "2025-08-01T00:00:00Z",
  updated_at: "2025-08-01T00:00:00Z",
  total_amount: 0,
  ...overrides,
});

// 重複チェック用のテストデータ
export const createBillsForDuplicateTest = () => [
  createMockBill({ id: 1, year: 2025, month: 8 }),
  createMockBill({ id: 2, year: 2025, month: 9 }),
  createMockBill({ id: 3, year: 2024, month: 12 }),
];

export * from "@testing-library/react";
export { customRender as render };
