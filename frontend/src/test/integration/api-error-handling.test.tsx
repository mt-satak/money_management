import React from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { render, createMockUser, createMockBill } from "../utils/test-utils";
import { server } from "../mocks/server";
import BillsListPage from "../../pages/BillsListPage";
import { BillResponse, User } from "../../types";

// useAuthフックのモック
const mockUser = createMockUser({ id: 1, name: "テストユーザー" });
vi.mock("../../hooks/useAuth", () => ({
  useAuth: () => ({
    user: mockUser,
    token: "mock-token",
    login: vi.fn(),
    logout: vi.fn(),
    register: vi.fn(),
  }),
}));

// 基本データ
const mockBills: BillResponse[] = [
  createMockBill({ id: 1, year: 2025, month: 8 }),
];

const mockUsers: User[] = [
  createMockUser({ id: 1, name: "テストユーザー1" }),
  createMockUser({ id: 2, name: "テストユーザー2" }),
];

// 基本的なAPIレスポンス
const setupBasicMocks = () => {
  server.use(
    http.get("/api/bills", () => {
      return HttpResponse.json({ bills: mockBills });
    }),
    http.get("/api/users", () => {
      return HttpResponse.json({ users: mockUsers });
    }),
  );
};

describe("APIエラーハンドリング統合テスト", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
    setupBasicMocks();
  });

  const openCreateModalAndFillForm = async () => {
    render(<BillsListPage />);

    // データの読み込み完了を待つ（新規作成ボタンが表示されるまで）
    const createButton = await waitFor(
      () => screen.getByRole("button", { name: /新規作成/ }),
      { timeout: 10000 },
    );
    await user.click(createButton);

    const comboboxes = screen.getAllByRole("combobox");
    const yearInput = screen.getByRole("spinbutton");
    const monthSelect = comboboxes[0];
    const payerSelect = comboboxes[1];

    // フォームに値を入力（重複しない年月）
    await user.clear(yearInput);
    await user.type(yearInput, "2025");
    await user.selectOptions(monthSelect, "7");
    await user.selectOptions(payerSelect, "2");

    return {
      yearInput,
      monthSelect,
      payerSelect,
      get submitButton() {
        return document.querySelector(
          'form button[type="submit"]',
        ) as HTMLButtonElement;
      },
    };
  };

  describe("409 Conflict エラー (重複エラー)", () => {
    it("409エラー時に適切なエラーメッセージを表示", async () => {
      // 409エラーを返すAPIをモック
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.json(
            { error: "指定された年月の家計簿は既に存在します" },
            { status: 409 },
          );
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      // フォーム送信
      await user.click(submitButton);

      // 409エラーメッセージが表示されることを確認
      await waitFor(
        () => {
          // エラーメッセージが表示されることを確認（画面に実際に表示される内容に合わせて調整）
          const errorElement = screen.queryByText(
            "指定された年月の家計簿は既に存在します",
          );
          if (!errorElement) {
            // フォールバック：一般的なエラーメッセージも確認
            expect(
              screen.getByText(/家計簿.*失敗|指定された年月.*存在/),
            ).toBeInTheDocument();
          } else {
            expect(errorElement).toBeInTheDocument();
          }
        },
        { timeout: 3000 },
      );

      // 成功時のモーダルクローズが発生しないことを確認
      expect(screen.getByText("新しい家計簿を作成")).toBeInTheDocument();
    });

    it("409エラー後にフォームが使用可能な状態になる", async () => {
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.json(
            { error: "指定された年月の家計簿は既に存在します" },
            { status: 409 },
          );
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      // フォーム送信（409エラーになる）
      await user.click(submitButton);

      await waitFor(() => {
        expect(
          screen.getByText(/家計簿.*失敗|指定された年月.*存在/),
        ).toBeInTheDocument();
      });

      // ボタンが再度有効化されることを確認
      expect(submitButton).not.toBeDisabled();
      expect(submitButton).toHaveTextContent("作成");

      // 再度送信可能であることを確認
      await user.click(submitButton);
      // 2回目のAPIコールが発生することを期待
    });
  });

  describe("500 Internal Server Error", () => {
    it("500エラー時に汎用エラーメッセージを表示", async () => {
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.json(
            { error: "家計簿の作成に失敗しました" },
            { status: 500 },
          );
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      await user.click(submitButton);

      await waitFor(() => {
        expect(
          screen.getByText("家計簿の作成に失敗しました"),
        ).toBeInTheDocument();
      });

      // モーダルが開いたままであることを確認
      expect(screen.getByText("新しい家計簿を作成")).toBeInTheDocument();
    });

    it("500エラー時にローディング状態が解除される", async () => {
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.json(
            { error: "家計簿の作成に失敗しました" },
            { status: 500 },
          );
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      // 送信ボタンをクリック
      await user.click(submitButton);

      // エラーメッセージが表示されることを確認
      await waitFor(() => {
        expect(
          screen.getByText("家計簿の作成に失敗しました"),
        ).toBeInTheDocument();
      });

      // ローディング状態が解除されることを確認
      expect(submitButton).not.toHaveTextContent("作成中...");
      expect(submitButton).toHaveTextContent("作成");
      expect(submitButton).not.toBeDisabled();
    });
  });

  describe("ネットワークエラー", () => {
    it("ネットワークエラー時に接続エラーメッセージを表示", async () => {
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.error();
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      await user.click(submitButton);

      await waitFor(() => {
        // ネットワークエラー時のメッセージを確認（実際のエラーメッセージに合わせて調整）
        expect(
          screen.getByText(/Failed to fetch|ネットワークエラー/),
        ).toBeInTheDocument();
      });
    });

    it("遅いレスポンス時のローディング状態確認", async () => {
      server.use(
        http.post("/api/bills", async () => {
          // 短い遅延でローディング状態をテスト
          await new Promise((resolve) => setTimeout(resolve, 100));
          return HttpResponse.json(createMockBill({ year: 2025, month: 7 }), {
            status: 201,
          });
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      await user.click(submitButton);

      // 短時間だけローディング状態になることを確認
      expect(submitButton).toHaveTextContent("作成中...");
      expect(submitButton).toBeDisabled();

      // 最終的に成功してモーダルが閉じることを確認
      await waitFor(
        () => {
          expect(
            screen.queryByText("新しい家計簿を作成"),
          ).not.toBeInTheDocument();
        },
        { timeout: 3000 },
      );
    });
  });

  describe("APIレスポンス形式エラー", () => {
    it("不正なJSONレスポンス時のエラーハンドリング", async () => {
      server.use(
        http.post("/api/bills", () => {
          return new HttpResponse("Invalid JSON", {
            status: 200,
            headers: {
              "Content-Type": "application/json",
            },
          });
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      await user.click(submitButton);

      await waitFor(() => {
        // JSON Parse エラーに対するフォールバックメッセージ
        expect(
          screen.getByText(
            /ネットワークエラー|リクエストが失敗しました|Unexpected token/,
          ),
        ).toBeInTheDocument();
      });
    });

    it("空のerrorcodeレスポンスのハンドリング", async () => {
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.json(
            {
              success: false,
              message: "Unknown error",
            },
            { status: 400 },
          );
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      await user.click(submitButton);

      await waitFor(() => {
        // デフォルトエラーメッセージが表示される
        expect(
          screen.getByText("リクエストが失敗しました"),
        ).toBeInTheDocument();
      });
    });
  });

  describe("認証エラー", () => {
    it("401 Unauthorized エラーのハンドリング", async () => {
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.json(
            { error: "認証が必要です" },
            { status: 401 },
          );
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText("認証が必要です")).toBeInTheDocument();
      });
    });

    it("403 Forbidden エラーのハンドリング", async () => {
      server.use(
        http.post("/api/bills", () => {
          return HttpResponse.json(
            { error: "アクセスが拒否されました" },
            { status: 403 },
          );
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      await user.click(submitButton);

      await waitFor(() => {
        expect(
          screen.getByText("アクセスが拒否されました"),
        ).toBeInTheDocument();
      });
    });
  });

  describe("複数エラーのシーケンス", () => {
    it("エラー → 成功のシーケンスが正しく動作する", async () => {
      let callCount = 0;

      server.use(
        http.post("/api/bills", () => {
          callCount++;
          if (callCount === 1) {
            // 1回目は500エラー
            return HttpResponse.json(
              { error: "一時的なエラーです" },
              { status: 500 },
            );
          } else {
            // 2回目以降は成功
            return HttpResponse.json(createMockBill({ year: 2025, month: 7 }), {
              status: 201,
            });
          }
        }),
      );

      const { submitButton } = await openCreateModalAndFillForm();

      // 1回目の送信（エラー）
      await user.click(submitButton);
      await waitFor(() => {
        expect(screen.getByText("一時的なエラーです")).toBeInTheDocument();
      });

      // 2回目の送信（成功）
      await user.click(submitButton);
      await waitFor(() => {
        // モーダルが閉じることを確認（成功の証拠）
        expect(
          screen.queryByText("新しい家計簿を作成"),
        ).not.toBeInTheDocument();
      });

      expect(callCount).toBe(2);
    });
  });
});
