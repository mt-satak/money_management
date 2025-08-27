import React from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render, createMockUser, createMockBill } from "../utils/test-utils";
import BillsListPage from "../../pages/BillsListPage";
import { BillResponse, User } from "../../types";
import { api } from "../../services/api";

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

// APIのモック（削除ボタンが表示されないように請求者を別のユーザーに設定）
const mockBills: BillResponse[] = [
  createMockBill({ id: 1, year: 2025, month: 8, requester_id: 2, payer_id: 1 }),
  createMockBill({ id: 2, year: 2025, month: 9, requester_id: 2, payer_id: 1 }),
];

const mockUsers: User[] = [
  createMockUser({ id: 1, name: "テストユーザー1" }),
  createMockUser({ id: 2, name: "テストユーザー2" }),
];

vi.mock("../../services/api", () => ({
  api: {
    getBills: vi.fn(() => Promise.resolve({ bills: mockBills })),
    getUsers: vi.fn(() => Promise.resolve({ users: mockUsers })),
    createBill: vi.fn((data) =>
      Promise.resolve(
        createMockBill({
          year: data.year,
          month: data.month,
          payer_id: data.payer_id,
        }),
      ),
    ),
    deleteBill: vi.fn(() =>
      Promise.resolve({ message: "家計簿を削除しました" }),
    ),
  },
}));

describe("BillsListPage - 支払者家計簿表示機能", () => {
  const user = userEvent.setup();

  describe("支払者家計簿のスタイリング", () => {
    it("請求中ステータスの支払者家計簿は黄色背景で表示される", async () => {
      // 請求中ステータスの支払者家計簿をモック
      const requestedBills = [
        createMockBill({
          id: 1,
          year: 2025,
          month: 8,
          status: "requested",
          requester_id: 2,
          payer_id: 1, // mockUser.idと同じ
        }),
      ];

      vi.mocked(api.getBills).mockResolvedValueOnce({ bills: requestedBills });

      render(<BillsListPage />);

      await waitFor(() => {
        const billCard = screen
          .getByText("2025年8月の家計簿")
          .closest(".bg-yellow-50");
        expect(billCard).toBeInTheDocument();
        expect(billCard).toHaveClass(
          "bg-yellow-50",
          "border-yellow-200",
          "border-2",
        );
      });

      // 支払者バッジも確認
      expect(screen.getByText("支払者")).toBeInTheDocument();
    });

    it("支払済みステータスの支払者家計簿は緑色背景で表示される", async () => {
      // 支払済みステータスの支払者家計簿をモック
      const paidBills = [
        createMockBill({
          id: 1,
          year: 2025,
          month: 8,
          status: "paid",
          requester_id: 2,
          payer_id: 1, // mockUser.idと同じ
        }),
      ];

      vi.mocked(api.getBills).mockResolvedValueOnce({ bills: paidBills });

      render(<BillsListPage />);

      await waitFor(() => {
        const billCard = screen
          .getByText("2025年8月の家計簿")
          .closest(".bg-green-50");
        expect(billCard).toBeInTheDocument();
        expect(billCard).toHaveClass(
          "bg-green-50",
          "border-green-200",
          "border-2",
        );
      });

      // 支払者バッジも確認
      expect(screen.getByText("支払者")).toBeInTheDocument();
    });

    it("請求者の家計簿は通常スタイルで表示される", async () => {
      // 請求者の家計簿をモック
      const requesterBills = [
        createMockBill({
          id: 1,
          year: 2025,
          month: 8,
          status: "pending",
          requester_id: 1, // mockUser.idと同じ
          payer_id: 2,
        }),
      ];

      vi.mocked(api.getBills).mockResolvedValueOnce({ bills: requesterBills });

      render(<BillsListPage />);

      await waitFor(() => {
        const billCard = screen.getByText("2025年8月の家計簿").closest(".card");
        expect(billCard).toBeInTheDocument();
        expect(billCard).not.toHaveClass("bg-yellow-50", "bg-green-50");
      });

      // 請求者バッジも確認
      expect(screen.getByText("請求者")).toBeInTheDocument();
    });

    it("支払者は削除ボタンが表示されない", async () => {
      // 支払者の家計簿をモック（作成中ステータス）
      const payerBills = [
        createMockBill({
          id: 1,
          year: 2025,
          month: 8,
          status: "pending", // 作成中でも
          requester_id: 2,
          payer_id: 1, // 支払者が現在のユーザー
        }),
      ];

      vi.mocked(api.getBills).mockResolvedValueOnce({ bills: payerBills });

      render(<BillsListPage />);

      await waitFor(() => {
        expect(screen.getByText("2025年8月の家計簿")).toBeInTheDocument();
      });

      // 削除ボタンが表示されないことを確認
      expect(screen.queryByText("削除")).not.toBeInTheDocument();
    });
  });

  describe("混合表示テスト", () => {
    it("請求者と支払者の家計簿が混在して正しく表示される", async () => {
      const mixedBills = [
        createMockBill({
          id: 1,
          year: 2025,
          month: 8,
          status: "requested",
          requester_id: 2,
          payer_id: 1, // 支払者として
          requester: {
            id: 2,
            name: "請求者ユーザー",
            account_id: "requester",
            created_at: "2025-01-01T00:00:00Z",
            updated_at: "2025-01-01T00:00:00Z",
          },
          payer: {
            id: 1,
            name: "テストユーザー",
            account_id: "payer",
            created_at: "2025-01-01T00:00:00Z",
            updated_at: "2025-01-01T00:00:00Z",
          },
        }),
        createMockBill({
          id: 2,
          year: 2025,
          month: 9,
          status: "pending",
          requester_id: 1, // 請求者として
          payer_id: 2,
          requester: {
            id: 1,
            name: "テストユーザー",
            account_id: "requester",
            created_at: "2025-01-01T00:00:00Z",
            updated_at: "2025-01-01T00:00:00Z",
          },
          payer: {
            id: 2,
            name: "支払者ユーザー",
            account_id: "payer",
            created_at: "2025-01-01T00:00:00Z",
            updated_at: "2025-01-01T00:00:00Z",
          },
        }),
      ];

      vi.mocked(api.getBills).mockResolvedValueOnce({ bills: mixedBills });

      render(<BillsListPage />);

      await waitFor(() => {
        // 支払者の家計簿（請求中）は黄色背景
        const payerCard = screen
          .getByText("2025年8月の家計簿")
          .closest(".bg-yellow-50");
        expect(payerCard).toBeInTheDocument();
        expect(screen.getByText("支払者")).toBeInTheDocument();

        // 請求者の家計簿（作成中）は通常スタイル
        const requesterCard = screen
          .getByText("2025年9月の家計簿")
          .closest(".card");
        expect(requesterCard).toBeInTheDocument();
        expect(requesterCard).not.toHaveClass("bg-yellow-50", "bg-green-50");
        expect(screen.getByText("請求者")).toBeInTheDocument();
      });
    });
  });
});

describe("BillsListPage - 重複チェック機能", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  const openCreateModal = async () => {
    render(<BillsListPage />);

    // データの読み込み完了を待つ
    await waitFor(() => {
      expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });

    // 新規作成ボタンをクリック（より堅牢なセレクター使用）
    const createButton = screen.getByRole("button", { name: /新規作成/ });
    await user.click(createButton);

    // モーダルが開かれることを確認
    expect(screen.getByText("新しい家計簿を作成")).toBeInTheDocument();

    const comboboxes = screen.getAllByRole("combobox");

    return {
      yearInput: screen.getByRole("spinbutton"),
      monthSelect: comboboxes[0], // 最初のselectが月のselect
      payerSelect: comboboxes[1], // 2番目のselectが支払者のselect
      // submitButtonは状態によってテキストが変わるため、各テストで個別に取得する
      get submitButton() {
        return document.querySelector(
          'form button[type="submit"]',
        ) as HTMLButtonElement;
      },
    };
  };

  describe("重複警告表示", () => {
    it("既存の家計簿と同じ年月を選択すると警告メッセージを表示", async () => {
      const { yearInput, monthSelect } = await openCreateModal();

      // 既存の家計簿と同じ年月（2025年8月）を選択
      await user.clear(yearInput);
      await user.type(yearInput, "2025");
      await user.selectOptions(monthSelect, "8");

      // 警告メッセージが表示されることを確認
      expect(
        screen.getByText("⚠️ 2025年8月の家計簿は既に存在します"),
      ).toBeInTheDocument();
    });

    it("存在しない年月を選択すると警告メッセージを非表示", async () => {
      const { yearInput, monthSelect } = await openCreateModal();

      // 存在しない年月（2025年7月）を選択
      await user.clear(yearInput);
      await user.type(yearInput, "2025");
      await user.selectOptions(monthSelect, "7");

      // 警告メッセージが表示されないことを確認
      expect(
        screen.queryByText(/⚠️.*の家計簿は既に存在します/),
      ).not.toBeInTheDocument();
    });

    it("年月を変更すると警告メッセージが動的に更新される", async () => {
      const { yearInput, monthSelect } = await openCreateModal();

      // まず重複する年月を選択
      await user.clear(yearInput);
      await user.type(yearInput, "2025");
      await user.selectOptions(monthSelect, "8");
      expect(
        screen.getByText("⚠️ 2025年8月の家計簿は既に存在します"),
      ).toBeInTheDocument();

      // 別の重複する年月に変更
      await user.selectOptions(monthSelect, "9");
      expect(
        screen.getByText("⚠️ 2025年9月の家計簿は既に存在します"),
      ).toBeInTheDocument();
      expect(
        screen.queryByText("⚠️ 2025年8月の家計簿は既に存在します"),
      ).not.toBeInTheDocument();

      // 重複しない年月に変更
      await user.selectOptions(monthSelect, "7");
      expect(
        screen.queryByText(/⚠️.*の家計簿は既に存在します/),
      ).not.toBeInTheDocument();
    });
  });

  describe("作成ボタンの状態制御", () => {
    it("重複する年月の場合は作成ボタンが無効化される", async () => {
      const { yearInput, monthSelect, payerSelect, submitButton } =
        await openCreateModal();

      // 支払者を選択
      await user.selectOptions(payerSelect, "2");

      // 重複する年月を選択
      await user.clear(yearInput);
      await user.type(yearInput, "2025");
      await user.selectOptions(monthSelect, "8");

      // ボタンが無効化されることを確認
      expect(submitButton).toBeDisabled();
      expect(submitButton).toHaveTextContent("重複のため作成不可");
    });

    it("重複しない年月の場合は作成ボタンが有効化される", async () => {
      const { yearInput, monthSelect, payerSelect, submitButton } =
        await openCreateModal();

      // 支払者を選択
      await user.selectOptions(payerSelect, "2");

      // 重複しない年月を選択
      await user.clear(yearInput);
      await user.type(yearInput, "2025");
      await user.selectOptions(monthSelect, "7");

      // ボタンが有効化されることを確認
      expect(submitButton).not.toBeDisabled();
      expect(submitButton).toHaveTextContent("作成");
    });

    it("重複から非重複に変更するとボタン状態が更新される", async () => {
      const { yearInput, monthSelect, payerSelect, submitButton } =
        await openCreateModal();

      // 支払者を選択
      await user.selectOptions(payerSelect, "2");

      // 最初に重複する年月を選択
      await user.clear(yearInput);
      await user.type(yearInput, "2025");
      await user.selectOptions(monthSelect, "8");
      expect(submitButton).toBeDisabled();
      expect(submitButton).toHaveTextContent("重複のため作成不可");

      // 重複しない年月に変更
      await user.selectOptions(monthSelect, "7");
      expect(submitButton).not.toBeDisabled();
      expect(submitButton).toHaveTextContent("作成");
    });
  });

  describe("支払者選択の制御", () => {
    it("ログインユーザーが支払者選択肢から除外される", async () => {
      const { payerSelect } = await openCreateModal();

      // 支払者選択肢を取得
      const options = Array.from(payerSelect.querySelectorAll("option"));
      const optionValues = options.map((opt) => opt.value);
      const optionTexts = options.map((opt) => opt.textContent);

      // ログインユーザー(ID: 1)が選択肢に含まれないことを確認
      expect(optionValues).not.toContain("1");
      expect(optionTexts).toContain("支払者を選択してください"); // デフォルトオプション
      expect(optionTexts).toContain("テストユーザー2"); // ID:2のユーザー
      expect(optionTexts).not.toContain("テストユーザー1"); // ID:1のログインユーザーは除外
    });

    it("ログインユーザー以外のユーザーは支払者として選択可能", async () => {
      const { payerSelect } = await openCreateModal();

      // ID:2のユーザーを選択可能であることを確認
      await user.selectOptions(payerSelect, "2");
      expect((payerSelect as HTMLSelectElement).value).toBe("2");
    });
  });

  describe("フォーム送信の制御", () => {
    it("重複しない場合は正常に送信される", async () => {
      const mockCreateBill = vi.fn(() =>
        Promise.resolve(createMockBill({ year: 2025, month: 7 })),
      );
      const { api } = await import("../../services/api");
      vi.mocked(api.createBill).mockImplementation(mockCreateBill);

      const { yearInput, monthSelect, payerSelect, submitButton } =
        await openCreateModal();

      // 必要な値を入力（重複しない年月）
      await user.clear(yearInput);
      await user.type(yearInput, "2025");
      await user.selectOptions(monthSelect, "7");
      await user.selectOptions(payerSelect, "2");

      // フォームを送信
      await user.click(submitButton);

      // API が正しいパラメータで呼び出されることを確認
      await waitFor(() => {
        expect(mockCreateBill).toHaveBeenCalledWith({
          year: 2025,
          month: 7,
          payer_id: 2,
        });
      });
    });
  });
});

describe("BillsListPage - 削除機能", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("請求者は作成中ステータスの家計簿を削除できる", async () => {
    // 削除可能な家計簿（請求者=ログインユーザー、ステータス=pending）を作成
    const deletableBill = createMockBill({
      id: 3,
      year: 2025,
      month: 10,
      requester_id: 1, // ログインユーザーのID
      payer_id: 2,
      status: "pending",
    });

    const billsWithDeletable = [...mockBills, deletableBill];
    const { api } = await import("../../services/api");
    vi.mocked(api.getBills).mockResolvedValue({ bills: billsWithDeletable });

    render(<BillsListPage />);

    // データの読み込み完了を待つ
    await waitFor(() => {
      expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });

    // 削除ボタンが表示されることを確認（getByRole を使って特定）
    const deleteButtons = screen.getAllByText("削除");
    expect(deleteButtons).toHaveLength(1);
    const deleteButton = deleteButtons[0];

    // 確認ダイアログをモック
    const confirmSpy = vi.spyOn(window, "confirm");
    confirmSpy.mockReturnValue(true);

    // 削除ボタンをクリック
    await user.click(deleteButton);

    // 確認ダイアログが表示されることを確認
    expect(confirmSpy).toHaveBeenCalledWith(
      "この家計簿を削除しますか？この操作は取り消せません。",
    );

    // API が正しいパラメータで呼び出されることを確認
    await waitFor(() => {
      expect(api.deleteBill).toHaveBeenCalledWith(3);
    });

    confirmSpy.mockRestore();
  });

  it("支払者は削除ボタンが表示されない", async () => {
    // 支払者のみの家計簿（請求者≠ログインユーザー）を作成
    const payerOnlyBill = createMockBill({
      id: 4,
      year: 2025,
      month: 11,
      requester_id: 2, // 他のユーザーのID
      payer_id: 1, // ログインユーザーのID
      status: "pending",
    });

    const billsWithPayerOnly = [...mockBills, payerOnlyBill];
    const { api } = await import("../../services/api");
    vi.mocked(api.getBills).mockResolvedValue({ bills: billsWithPayerOnly });

    render(<BillsListPage />);

    // データの読み込み完了を待つ
    await waitFor(() => {
      expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });

    // 削除ボタンが表示されないことを確認
    expect(screen.queryByText("削除")).not.toBeInTheDocument();
  });

  it("確定済みステータスの家計簿は削除ボタンが表示されない", async () => {
    // 確定済みの家計簿（請求者=ログインユーザー、ステータス=requested）を作成
    const confirmedBill = createMockBill({
      id: 5,
      year: 2025,
      month: 12,
      requester_id: 1, // ログインユーザーのID
      payer_id: 2,
      status: "requested", // 確定済みステータス
    });

    const billsWithConfirmed = [...mockBills, confirmedBill];
    const { api } = await import("../../services/api");
    vi.mocked(api.getBills).mockResolvedValue({ bills: billsWithConfirmed });

    render(<BillsListPage />);

    // データの読み込み完了を待つ
    await waitFor(() => {
      expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });

    // 削除ボタンが表示されないことを確認（確定済みのため）
    expect(screen.queryByText("削除")).not.toBeInTheDocument();
  });

  it("削除キャンセル時はAPIが呼び出されない", async () => {
    // 削除可能な家計簿を作成
    const deletableBill = createMockBill({
      id: 6,
      year: 2025,
      month: 11,
      requester_id: 1,
      payer_id: 2,
      status: "pending",
    });

    const billsWithDeletable = [...mockBills, deletableBill];
    const { api } = await import("../../services/api");
    vi.mocked(api.getBills).mockResolvedValue({ bills: billsWithDeletable });

    render(<BillsListPage />);

    // データの読み込み完了を待つ
    await waitFor(() => {
      expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
    });

    // 削除ボタンをクリック
    const deleteButtons = screen.getAllByText("削除");
    const deleteButton = deleteButtons[0];

    // 確認ダイアログをモック（キャンセルを選択）
    const confirmSpy = vi.spyOn(window, "confirm");
    confirmSpy.mockReturnValue(false);

    await user.click(deleteButton);

    // APIが呼び出されないことを確認
    expect(api.deleteBill).not.toHaveBeenCalled();

    confirmSpy.mockRestore();
  });
});
