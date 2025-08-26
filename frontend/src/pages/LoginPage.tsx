import React, { useState } from "react";
import { useAuth } from "../hooks/useAuth";

export default function LoginPage() {
  const { login, register } = useAuth();
  const [isRegistering, setIsRegistering] = useState(false);
  const [name, setName] = useState("");
  const [accountId, setAccountId] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      if (isRegistering) {
        if (password !== confirmPassword) {
          throw new Error("パスワードが一致しません");
        }
        if (password.length < 6) {
          throw new Error("パスワードは6文字以上で入力してください");
        }
        await register({ name, account_id: accountId, password });
      } else {
        await login({ account_id: accountId, password });
      }
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : isRegistering
            ? "アカウント登録に失敗しました"
            : "ログインに失敗しました",
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
      <div className="max-w-md w-full space-y-8 bg-white p-8 rounded-xl shadow-lg">
        <div className="text-center">
          <div className="text-4xl mb-4">💰</div>
          <h2 className="text-3xl font-extrabold text-gray-900">
            家計簿アプリ
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {isRegistering
              ? "アカウントを新規作成"
              : "アカウントにログインしてください"}
          </p>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md">
              {error}
            </div>
          )}

          <div className="space-y-4">
            {isRegistering && (
              <div>
                <label
                  htmlFor="name"
                  className="block text-sm font-medium text-gray-700"
                >
                  お名前
                </label>
                <input
                  id="name"
                  name="name"
                  type="text"
                  required
                  className="form-input mt-1"
                  placeholder="山田太郎"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
            )}

            <div>
              <label
                htmlFor="accountId"
                className="block text-sm font-medium text-gray-700"
              >
                アカウントID
              </label>
              <input
                id="accountId"
                name="accountId"
                type="text"
                required
                className="form-input mt-1"
                placeholder="英数字とアンダースコア、3-20文字"
                value={accountId}
                onChange={(e) => setAccountId(e.target.value)}
                minLength={3}
                maxLength={20}
                pattern="[a-zA-Z0-9_]+"
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium text-gray-700"
              >
                パスワード
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                className="form-input mt-1"
                placeholder="6文字以上"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>

            {isRegistering && (
              <div>
                <label
                  htmlFor="confirmPassword"
                  className="block text-sm font-medium text-gray-700"
                >
                  パスワード（確認）
                </label>
                <input
                  id="confirmPassword"
                  name="confirmPassword"
                  type="password"
                  required
                  className="form-input mt-1"
                  placeholder="パスワードを再入力"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                />
              </div>
            )}
          </div>

          <button
            type="submit"
            disabled={loading}
            className="btn-primary w-full disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <div className="flex items-center justify-center">
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                {isRegistering ? "登録中..." : "ログイン中..."}
              </div>
            ) : isRegistering ? (
              "アカウントを作成"
            ) : (
              "ログイン"
            )}
          </button>

          <div className="text-center">
            <button
              type="button"
              onClick={() => {
                setIsRegistering(!isRegistering);
                setError("");
                setName("");
                setAccountId("");
                setPassword("");
                setConfirmPassword("");
              }}
              className="text-sm text-blue-600 hover:text-blue-800 underline"
            >
              {isRegistering
                ? "すでにアカウントをお持ちですか？ログインはこちら"
                : "アカウントをお持ちでない方はこちら"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
