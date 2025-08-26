import { http, HttpResponse } from "msw";
import { BillResponse, User } from "../../types";

export const handlers = [
  // 家計簿一覧取得
  http.get("/api/bills", () => {
    const mockBills: BillResponse[] = [
      {
        id: 1,
        year: 2025,
        month: 8,
        requester_id: 1,
        payer_id: 2,
        status: "pending",
        request_date: undefined,
        payment_date: undefined,
        requester: {
          id: 1,
          name: "テストユーザー1",
          account_id: "test1",
          created_at: "2025-01-01",
          updated_at: "2025-01-01",
        },
        payer: {
          id: 2,
          name: "テストユーザー2",
          account_id: "test2",
          created_at: "2025-01-01",
          updated_at: "2025-01-01",
        },
        items: [],
        created_at: "2025-08-01T00:00:00Z",
        updated_at: "2025-08-01T00:00:00Z",
        total_amount: 0,
      },
    ];
    return HttpResponse.json({ bills: mockBills });
  }),

  // ユーザー一覧取得
  http.get("/api/users", () => {
    const mockUsers: User[] = [
      {
        id: 1,
        name: "テストユーザー1",
        account_id: "test1",
        created_at: "2025-01-01",
        updated_at: "2025-01-01",
      },
      {
        id: 2,
        name: "テストユーザー2",
        account_id: "test2",
        created_at: "2025-01-01",
        updated_at: "2025-01-01",
      },
    ];
    return HttpResponse.json({ users: mockUsers });
  }),

  // 家計簿作成成功
  http.post("/api/bills", () => {
    const mockBill: BillResponse = {
      id: 2,
      year: 2025,
      month: 9,
      requester_id: 1,
      payer_id: 2,
      status: "pending",
      request_date: undefined,
      payment_date: undefined,
      requester: {
        id: 1,
        name: "テストユーザー1",
        account_id: "test1",
        created_at: "2025-01-01",
        updated_at: "2025-01-01",
      },
      payer: {
        id: 2,
        name: "テストユーザー2",
        account_id: "test2",
        created_at: "2025-01-01",
        updated_at: "2025-01-01",
      },
      items: [],
      created_at: "2025-09-01T00:00:00Z",
      updated_at: "2025-09-01T00:00:00Z",
      total_amount: 0,
    };
    return HttpResponse.json(mockBill, { status: 201 });
  }),

  // 重複エラーレスポンス
  http.post("/api/bills/duplicate", () => {
    return HttpResponse.json(
      { error: "指定された年月の家計簿は既に存在します" },
      { status: 409 },
    );
  }),

  // サーバーエラーレスポンス
  http.post("/api/bills/server-error", () => {
    return HttpResponse.json(
      { error: "家計簿の作成に失敗しました" },
      { status: 500 },
    );
  }),
];
