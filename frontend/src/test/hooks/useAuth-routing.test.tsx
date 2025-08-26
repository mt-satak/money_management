import React from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { AuthProvider, useAuth } from "../../hooks/useAuth";
import { LoginRequest, RegisterRequest, User } from "../../types";

// useNavigateをモック
const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// APIのモック
const mockLoginResponse = {
  token: "mock-token",
  user: {
    id: 1,
    name: "テストユーザー",
    account_id: "test_user",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  } as User,
};

const mockRegisterResponse = {
  token: "mock-token",
  user: {
    id: 2,
    name: "新規ユーザー",
    account_id: "new_user",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  } as User,
};

const mockApi = {
  login: vi.fn(),
  register: vi.fn(),
  getMe: vi.fn(),
};

vi.mock("../../services/api", () => ({
  api: mockApi,
}));

// localStorageのモック
const mockLocalStorage = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
};
Object.defineProperty(window, "localStorage", {
  value: mockLocalStorage,
});

// テスト用のWrapper
const wrapper = ({ children }: { children: React.ReactNode }) => (
  <BrowserRouter>
    <AuthProvider>{children}</AuthProvider>
  </BrowserRouter>
);

describe("useAuth - ルーティング修正テスト", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockNavigate.mockClear();
    mockLocalStorage.getItem.mockReturnValue(null);
    mockLocalStorage.setItem.mockImplementation(() => {});
    mockLocalStorage.removeItem.mockImplementation(() => {});
  });

  describe("ログイン機能のナビゲーション", () => {
    it('ログイン成功時にnavigate("/bills", { replace: true })が呼び出される', async () => {
      mockApi.login.mockResolvedValue(mockLoginResponse);

      const { result } = renderHook(() => useAuth(), { wrapper });

      const loginData: LoginRequest = {
        account_id: "test_user",
        password: "password",
      };

      await act(async () => {
        await result.current.login(loginData);
      });

      // API呼び出しの確認
      expect(mockApi.login).toHaveBeenCalledWith(loginData);

      // localStorage設定の確認
      expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
        "token",
        "mock-token",
      );

      // ナビゲーション呼び出しの確認
      expect(mockNavigate).toHaveBeenCalledWith("/bills", { replace: true });

      // ユーザー状態の確認
      expect(result.current.user).toEqual(mockLoginResponse.user);
    });

    it("ログインエラー時はナビゲーションが呼び出されない", async () => {
      const loginError = new Error("ログイン失敗");
      mockApi.login.mockRejectedValue(loginError);

      const { result } = renderHook(() => useAuth(), { wrapper });

      const loginData: LoginRequest = {
        account_id: "invalid_user",
        password: "wrong_password",
      };

      await act(async () => {
        try {
          await result.current.login(loginData);
        } catch (error) {
          expect(error).toBe(loginError);
        }
      });

      // ナビゲーションが呼び出されないことを確認
      expect(mockNavigate).not.toHaveBeenCalled();

      // ユーザー状態がnullのままであることを確認
      expect(result.current.user).toBeNull();
    });
  });

  describe("新規登録機能のナビゲーション", () => {
    it('新規登録成功時にnavigate("/bills", { replace: true })が呼び出される', async () => {
      mockApi.register.mockResolvedValue(mockRegisterResponse);

      const { result } = renderHook(() => useAuth(), { wrapper });

      const registerData: RegisterRequest = {
        name: "新規ユーザー",
        account_id: "new_user",
        password: "password123",
      };

      await act(async () => {
        await result.current.register(registerData);
      });

      // API呼び出しの確認
      expect(mockApi.register).toHaveBeenCalledWith(registerData);

      // localStorage設定の確認
      expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
        "token",
        "mock-token",
      );

      // ナビゲーション呼び出しの確認
      expect(mockNavigate).toHaveBeenCalledWith("/bills", { replace: true });

      // ユーザー状態の確認
      expect(result.current.user).toEqual(mockRegisterResponse.user);
    });

    it("新規登録エラー時はナビゲーションが呼び出されない", async () => {
      const registerError = new Error("新規登録失敗");
      mockApi.register.mockRejectedValue(registerError);

      const { result } = renderHook(() => useAuth(), { wrapper });

      const registerData: RegisterRequest = {
        name: "テストユーザー",
        account_id: "duplicate_user",
        password: "password123",
      };

      await act(async () => {
        try {
          await result.current.register(registerData);
        } catch (error) {
          expect(error).toBe(registerError);
        }
      });

      // ナビゲーションが呼び出されないことを確認
      expect(mockNavigate).not.toHaveBeenCalled();

      // ユーザー状態がnullのままであることを確認
      expect(result.current.user).toBeNull();
    });
  });

  describe("ログアウト機能のナビゲーション", () => {
    it('ログアウト時にnavigate("/", { replace: true })が呼び出される', async () => {
      // 最初にログイン状態にする
      mockApi.login.mockResolvedValue(mockLoginResponse);
      const { result } = renderHook(() => useAuth(), { wrapper });

      await act(async () => {
        await result.current.login({
          account_id: "test_user",
          password: "password",
        });
      });

      // ナビゲーションモックをクリア（ログイン時の呼び出しをクリア）
      mockNavigate.mockClear();

      // ログアウト実行
      act(() => {
        result.current.logout();
      });

      // localStorage削除の確認
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("token");

      // ナビゲーション呼び出しの確認
      expect(mockNavigate).toHaveBeenCalledWith("/", { replace: true });

      // ユーザー状態がnullになることを確認
      expect(result.current.user).toBeNull();
    });

    it("複数回ログアウトしても安全に動作する", () => {
      const { result } = renderHook(() => useAuth(), { wrapper });

      // 最初のログアウト
      act(() => {
        result.current.logout();
      });

      expect(mockNavigate).toHaveBeenCalledWith("/", { replace: true });
      expect(result.current.user).toBeNull();

      // ナビゲーションモックをクリア
      mockNavigate.mockClear();

      // 2回目のログアウト
      act(() => {
        result.current.logout();
      });

      // 2回目も正常にナビゲーションが呼び出される
      expect(mockNavigate).toHaveBeenCalledWith("/", { replace: true });
      expect(result.current.user).toBeNull();
    });
  });

  describe("初期化時の自動認証", () => {
    it("有効なトークンがある場合は自動認証され、ナビゲーションは呼び出されない", async () => {
      mockLocalStorage.getItem.mockReturnValue("valid-token");
      mockApi.getMe.mockResolvedValue(mockLoginResponse.user);

      const { result } = renderHook(() => useAuth(), { wrapper });

      // 初期ロード完了まで待機
      await act(async () => {
        // useEffectの完了を待つ
      });

      // 自動認証が成功することを確認
      expect(mockApi.getMe).toHaveBeenCalled();
      expect(result.current.user).toEqual(mockLoginResponse.user);
      expect(result.current.loading).toBe(false);

      // 初期化時はナビゲーションが呼び出されない
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it("無効なトークンの場合はトークンが削除され、ナビゲーションは呼び出されない", async () => {
      mockLocalStorage.getItem.mockReturnValue("invalid-token");
      mockApi.getMe.mockRejectedValue(new Error("Unauthorized"));

      const { result } = renderHook(() => useAuth(), { wrapper });

      // 初期ロード完了まで待機
      await act(async () => {
        // useEffectの完了を待つ
      });

      // トークンが削除されることを確認
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("token");
      expect(result.current.user).toBeNull();
      expect(result.current.loading).toBe(false);

      // 初期化時の認証失敗ではナビゲーションが呼び出されない
      expect(mockNavigate).not.toHaveBeenCalled();
    });
  });
});
