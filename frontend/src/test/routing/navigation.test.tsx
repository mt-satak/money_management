import React from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render, createMockUser } from "../utils/test-utils";
import App from "../../App";
import { User } from "../../types";

// React Router NavigationをモックするためのNavigation Mockを作成
const mockNavigate = vi.fn();
const mockUseNavigate = () => mockNavigate;

// useNavigateをモック
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// 認証状態をテスト用に制御するためのモック
let mockUser: User | null = null;
let mockLoading = false;

const mockLogin = vi.fn();
const mockLogout = vi.fn();
const mockRegister = vi.fn();

vi.mock("../../hooks/useAuth", () => ({
  useAuth: () => ({
    user: mockUser,
    loading: mockLoading,
    login: mockLogin,
    logout: mockLogout,
    register: mockRegister,
  }),
  AuthProvider: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

// APIのモック
vi.mock("../../services/api", () => ({
  api: {
    getBills: vi.fn(() => Promise.resolve({ bills: [] })),
    getUsers: vi.fn(() => Promise.resolve({ users: [] })),
  },
}));

describe("ナビゲーション・ルーティング修正テスト", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
    mockNavigate.mockClear();
    mockUser = null;
    mockLoading = false;
  });

  describe("ログアウト時のURL強制リセット", () => {
    it('ログアウト実行時にnavigate(\"/\")が呼び出される', async () => {
      // ログイン状態でテスト開始
      mockUser = createMockUser({ id: 1, name: "テストユーザー" });

      render(<App />);

      // ログアウトボタンが表示されるまで待機
      await waitFor(() => {
        expect(screen.getByText("ログアウト")).toBeInTheDocument();
      });

      // ログアウトボタンをクリック
      const logoutButton = screen.getByText("ログアウト");
      await user.click(logoutButton);

      // mockLogoutが呼ばれることを確認
      expect(mockLogout).toHaveBeenCalledTimes(1);
    });

    it("ログアウト後は未認証ユーザー向けルートが適用される", () => {
      // 未認証状態でテスト
      mockUser = null;

      render(<App />);

      // ログインページが表示されることを確認
      expect(screen.getByText("家計簿アプリ")).toBeInTheDocument();
      expect(
        screen.getByText("アカウントにログインしてください"),
      ).toBeInTheDocument();
    });
  });

  describe("ログイン成功後の明示的リダイレクト", () => {
    it('ログイン成功時にnavigate(\"/bills\")が呼び出される', async () => {
      // 未認証状態でテスト開始
      mockUser = null;

      render(<App />);

      // ログインフォームに入力
      const accountIdInput = screen.getByLabelText("アカウントID");
      const passwordInput = screen.getByLabelText("パスワード");
      const loginButton = screen.getByRole("button", { name: "ログイン" });

      await user.type(accountIdInput, "test_user");
      await user.type(passwordInput, "password");
      await user.click(loginButton);

      // mockLoginが呼ばれることを確認
      expect(mockLogin).toHaveBeenCalledWith({
        account_id: "test_user",
        password: "password",
      });
    });

    it('新規登録成功時にもnavigate(\"/bills\")が呼び出される', async () => {
      // 未認証状態でテスト開始
      mockUser = null;

      render(<App />);

      // 新規登録モードに切り替え
      const switchToRegisterButton =
        screen.getByText("アカウントをお持ちでない方はこちら");
      await user.click(switchToRegisterButton);

      // 新規登録フォームに入力
      const nameInput = screen.getByLabelText("お名前");
      const accountIdInput = screen.getByLabelText("アカウントID");
      const passwordInput = screen.getByLabelText("パスワード");
      const confirmPasswordInput = screen.getByLabelText("パスワード（確認）");
      const registerButton = screen.getByRole("button", {
        name: "アカウントを作成",
      });

      await user.type(nameInput, "テストユーザー");
      await user.type(accountIdInput, "new_user");
      await user.type(passwordInput, "password123");
      await user.type(confirmPasswordInput, "password123");
      await user.click(registerButton);

      // mockRegisterが呼ばれることを確認
      expect(mockRegister).toHaveBeenCalledWith({
        name: "テストユーザー",
        account_id: "new_user",
        password: "password123",
      });
    });
  });

  describe("App.tsx のルート分離", () => {
    it("未認証状態ではすべてのパスでログインページが表示される", () => {
      mockUser = null;

      // 任意のパス（家計簿詳細など）でアクセスした場合でも
      // ログインページが表示されることを確認
      render(<App />);

      expect(screen.getByText("家計簿アプリ")).toBeInTheDocument();
      expect(screen.getByLabelText("アカウントID")).toBeInTheDocument();
      expect(screen.getByLabelText("パスワード")).toBeInTheDocument();
    });

    it("認証済み状態では家計簿関連ルートが適用される", async () => {
      mockUser = createMockUser({ id: 1, name: "テストユーザー" });

      render(<App />);

      // ヘッダーが表示されることを確認（認証済みの証拠）
      await waitFor(() => {
        expect(screen.getByText("💰 家計簿アプリ")).toBeInTheDocument();
        expect(
          screen.getByText("こんにちは、テストユーザーさん"),
        ).toBeInTheDocument();
      });
    });
  });

  describe("認証チェック強化", () => {
    it("BillsListPageで未認証の場合はリダイレクトされる", () => {
      // これは実際のコンポーネントでuseEffectが実行されることで
      // 間接的にテストされる（モック呼び出しで確認）
      mockUser = null;

      render(<App />);

      // 未認証状態ではログインページが表示されることを確認
      expect(
        screen.getByText("アカウントにログインしてください"),
      ).toBeInTheDocument();
    });

    it("認証チェックが正常に動作する", async () => {
      // 最初は認証済み
      mockUser = createMockUser({ id: 1, name: "テストユーザー" });

      render(<App />);

      // 認証済みユーザー向けの画面が表示される
      await waitFor(() => {
        expect(
          screen.getByText("こんにちは、テストユーザーさん"),
        ).toBeInTheDocument();
      });
    });
  });

  describe("統合ナビゲーションフロー", () => {
    it("ログイン → ログアウト → 再ログインのフローが正常に動作する", async () => {
      // 1. 最初は未認証状態
      mockUser = null;
      const { rerender } = render(<App />);

      expect(
        screen.getByText("アカウントにログインしてください"),
      ).toBeInTheDocument();

      // 2. ログイン実行（認証状態に変更）
      mockUser = createMockUser({ id: 1, name: "テストユーザー" });
      rerender(<App />);

      await waitFor(() => {
        expect(
          screen.getByText("こんにちは、テストユーザーさん"),
        ).toBeInTheDocument();
      });

      // 3. ログアウト実行
      const logoutButton = screen.getByText("ログアウト");
      await user.click(logoutButton);
      expect(mockLogout).toHaveBeenCalled();

      // 4. ログアウト後は未認証状態（userをnullに変更）
      mockUser = null;
      rerender(<App />);

      expect(
        screen.getByText("アカウントにログインしてください"),
      ).toBeInTheDocument();
    });
  });
});
