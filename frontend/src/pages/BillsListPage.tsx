import React, { useState, useEffect } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../services/api";
import { BillResponse, User } from "../types";
import { useAuth } from "../hooks/useAuth";
import {
  isDuplicateBill,
  generateDuplicateWarningMessage,
  generateCreateButtonText,
} from "../utils/billUtils";

export default function BillsListPage() {
  const { user } = useAuth();
  const navigate = useNavigate();

  // 認証チェック - 未認証の場合はリダイレクト
  useEffect(() => {
    if (!user) {
      navigate("/", { replace: true });
      return;
    }
  }, [user, navigate]);
  const [bills, setBills] = useState<BillResponse[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createForm, setCreateForm] = useState({
    year: new Date().getFullYear(),
    month: new Date().getMonth() + 1,
    payer_id: 0,
  });
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<number | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [billsResponse, usersResponse] = await Promise.all([
        api.getBills(),
        api.getUsers(),
      ]);
      setBills(billsResponse.bills || []);
      setUsers(usersResponse.users || []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "家計簿一覧の取得に失敗しました",
      );
    } finally {
      setLoading(false);
    }
  };

  // 現在選択中の年月が重複しているかチェック
  const currentIsDuplicate = isDuplicateBill(
    bills,
    createForm.year,
    createForm.month,
  );

  const handleCreateBill = async (e: React.FormEvent) => {
    e.preventDefault();
    if (createForm.payer_id === 0) {
      setError("支払者を選択してください");
      return;
    }

    // 重複チェック（念のための二重チェック）
    if (currentIsDuplicate) {
      setError(
        `${createForm.year}年${createForm.month}月の家計簿は既に存在します`,
      );
      return;
    }

    try {
      setCreating(true);
      setError("");
      const newBill = await api.createBill(createForm);
      setBills([newBill, ...bills]);
      setShowCreateModal(false);
      setCreateForm({
        year: new Date().getFullYear(),
        month: new Date().getMonth() + 1,
        payer_id: 0,
      });
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "家計簿の作成に失敗しました",
      );
    } finally {
      setCreating(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const styles = {
      pending: "bg-yellow-100 text-yellow-800 border-yellow-200",
      requested: "bg-blue-100 text-blue-800 border-blue-200",
      paid: "bg-green-100 text-green-800 border-green-200",
    };
    const labels = {
      pending: "作成中",
      requested: "請求済み",
      paid: "支払済み",
    };
    return (
      <span
        className={`px-2 py-1 text-xs font-medium rounded-full border ${styles[status as keyof typeof styles]}`}
      >
        {labels[status as keyof typeof labels]}
      </span>
    );
  };

  const getRoleBadge = (bill: BillResponse, currentUserId: number) => {
    if (bill.requester_id === currentUserId) {
      return <span className="text-xs text-blue-600 font-medium">請求者</span>;
    } else if (bill.payer_id === currentUserId) {
      return <span className="text-xs text-green-600 font-medium">支払者</span>;
    }
    return null;
  };

  const handleDeleteBill = async (billId: number) => {
    if (
      !window.confirm("この家計簿を削除しますか？この操作は取り消せません。")
    ) {
      return;
    }

    try {
      setDeleting(billId);
      setError("");
      await api.deleteBill(billId);
      setBills(bills.filter((bill) => bill.id !== billId));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "家計簿の削除に失敗しました",
      );
    } finally {
      setDeleting(null);
    }
  };

  const canDeleteBill = (bill: BillResponse) => {
    return user && bill.requester_id === user.id && bill.status === "pending";
  };

  // 支払者の家計簿かどうかを判定し、ステータスに応じたカード背景色を返す
  const getCardStyle = (bill: BillResponse, currentUserId: number) => {
    const isPayerBill = bill.payer_id === currentUserId;

    if (!isPayerBill) {
      // 請求者の家計簿は通常スタイル
      return "card hover:shadow-lg transition-shadow";
    }

    // 支払者の家計簿の場合、ステータスに応じて背景色を変更
    switch (bill.status) {
      case "requested":
        // 請求中は黄色背景で目立つ
        return "card hover:shadow-lg transition-shadow bg-yellow-50 border-yellow-200 border-2";
      case "paid":
        // 支払済みは緑色背景
        return "card hover:shadow-lg transition-shadow bg-green-50 border-green-200 border-2";
      default:
        // その他（pending等）は通常スタイル
        return "card hover:shadow-lg transition-shadow";
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">月次家計簿一覧</h1>
          <p className="text-gray-600 mt-1">家計簿の作成・管理ができます</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn-primary"
        >
          + 新規作成
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md">
          {error}
        </div>
      )}

      {bills.length === 0 ? (
        <div className="card text-center py-12">
          <div className="text-6xl mb-4">📊</div>
          <h3 className="text-lg font-semibold text-gray-900 mb-2">
            家計簿がまだありません
          </h3>
          <p className="text-gray-600 mb-4">
            「新規作成」ボタンから最初の家計簿を作成しましょう
          </p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="btn-primary"
          >
            家計簿を作成する
          </button>
        </div>
      ) : (
        <div className="grid gap-4">
          {bills.map((bill) => (
            <div
              key={bill.id}
              className={
                user
                  ? getCardStyle(bill, user.id)
                  : "card hover:shadow-lg transition-shadow"
              }
            >
              <div className="flex justify-between items-start">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <h3 className="text-lg font-semibold text-gray-900">
                      {bill.year}年{bill.month}月の家計簿
                    </h3>
                    {getStatusBadge(bill.status)}
                    {user && getRoleBadge(bill, user.id)}
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm text-gray-600">
                    <div>
                      <span className="font-medium">請求者:</span>{" "}
                      {bill.requester.name}
                    </div>
                    <div>
                      <span className="font-medium">支払者:</span>{" "}
                      {bill.payer.name}
                    </div>
                    <div>
                      <span className="font-medium">項目数:</span>{" "}
                      {bill.items.length}個
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-2xl font-bold text-gray-900">
                    ¥{bill.total_amount.toLocaleString()}
                  </div>
                  <div className="text-sm text-gray-600">請求金額</div>
                </div>
              </div>

              {bill.items.length > 0 && (
                <div className="mt-4 pt-4 border-t border-gray-100">
                  <h4 className="text-sm font-medium text-gray-700 mb-3">
                    品目詳細
                  </h4>
                  <div className="space-y-2">
                    {bill.items.slice(0, 5).map((item) => (
                      <div
                        key={item.id}
                        className="py-1 px-2 bg-gray-50 rounded text-sm"
                      >
                        <div className="text-gray-700">{item.item_name}</div>
                        <div className="font-medium text-gray-900">
                          ¥{item.amount.toLocaleString()}
                        </div>
                      </div>
                    ))}
                  </div>
                  {bill.items.length > 5 && (
                    <div className="text-center text-xs text-gray-500 mt-2">
                      ...他{bill.items.length - 5}項目
                    </div>
                  )}
                </div>
              )}

              <div className="flex justify-between items-center mt-4 pt-4 border-t border-gray-100">
                <div className="text-xs text-gray-500">
                  作成: {new Date(bill.created_at).toLocaleDateString()}
                  {bill.request_date && (
                    <>
                      , 請求: {new Date(bill.request_date).toLocaleDateString()}
                    </>
                  )}
                  {bill.payment_date && (
                    <>
                      , 支払: {new Date(bill.payment_date).toLocaleDateString()}
                    </>
                  )}
                </div>
                <div className="flex items-center gap-3">
                  {canDeleteBill(bill) && (
                    <button
                      onClick={() => handleDeleteBill(bill.id)}
                      disabled={deleting === bill.id}
                      className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
                    >
                      {deleting === bill.id ? "削除中..." : "削除"}
                    </button>
                  )}
                  <Link
                    to={`/bills/${bill.year}/${bill.month}`}
                    className="text-blue-600 hover:text-blue-800 text-sm font-medium"
                  >
                    詳細・編集 →
                  </Link>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 新規作成モーダル */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md mx-4">
            <h2 className="text-xl font-bold text-gray-900 mb-4">
              新しい家計簿を作成
            </h2>

            <form onSubmit={handleCreateBill} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  年
                </label>
                <input
                  type="number"
                  value={createForm.year}
                  onChange={(e) =>
                    setCreateForm({
                      ...createForm,
                      year: parseInt(e.target.value),
                    })
                  }
                  className="form-input"
                  min="2020"
                  max="2030"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  月
                </label>
                <select
                  value={createForm.month}
                  onChange={(e) =>
                    setCreateForm({
                      ...createForm,
                      month: parseInt(e.target.value),
                    })
                  }
                  className="form-input"
                  required
                >
                  {Array.from({ length: 12 }, (_, i) => i + 1).map((month) => (
                    <option key={month} value={month}>
                      {month}月
                    </option>
                  ))}
                </select>
              </div>

              {/* 重複警告表示 */}
              {currentIsDuplicate && (
                <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 px-3 py-2 rounded-md text-sm">
                  {generateDuplicateWarningMessage(
                    createForm.year,
                    createForm.month,
                  )}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  支払者
                </label>
                <select
                  value={createForm.payer_id}
                  onChange={(e) =>
                    setCreateForm({
                      ...createForm,
                      payer_id: parseInt(e.target.value),
                    })
                  }
                  className="form-input"
                  required
                >
                  <option value={0}>支払者を選択してください</option>
                  {users
                    .filter((u) => u.id !== user?.id)
                    .map((u) => (
                      <option key={u.id} value={u.id}>
                        {u.name}
                      </option>
                    ))}
                </select>
              </div>

              <div className="flex justify-end gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
                >
                  キャンセル
                </button>
                <button
                  type="submit"
                  disabled={creating || currentIsDuplicate}
                  className="btn-primary disabled:opacity-50"
                >
                  {generateCreateButtonText(creating, currentIsDuplicate)}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
