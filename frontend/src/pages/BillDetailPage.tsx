import React, { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { api } from "../services/api";
import { BillResponse } from "../types";
import { useAuth } from "../hooks/useAuth";

export default function BillDetailPage() {
  const { year, month } = useParams<{ year: string; month: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  // 認証チェック - 未認証の場合はリダイレクト
  useEffect(() => {
    if (!user) {
      navigate("/", { replace: true });
      return;
    }
  }, [user, navigate]);
  const [bill, setBill] = useState<BillResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [items, setItems] = useState<
    Array<{ id?: number; item_name: string; amount: number }>
  >([{ item_name: "", amount: 0 }]);
  const [saving, setSaving] = useState(false);
  const [requesting, setRequesting] = useState(false);
  const [paying, setPaying] = useState(false);

  useEffect(() => {
    if (year && month) {
      loadBill();
    }
  }, [year, month]);

  const loadBill = async () => {
    if (!year || !month) return;

    try {
      setLoading(true);
      const response = await api.getBill(parseInt(year), parseInt(month));

      if ("bill" in response && response.bill === null) {
        setBill(null);
        setItems([{ item_name: "", amount: 0 }]);
      } else {
        const billData = response as BillResponse;
        setBill(billData);
        if (billData.items.length > 0) {
          setItems(
            billData.items.map((item) => ({
              id: item.id,
              item_name: item.item_name,
              amount: item.amount,
            })),
          );
        } else {
          setItems([{ item_name: "", amount: 0 }]);
        }
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "家計簿の取得に失敗しました",
      );
    } finally {
      setLoading(false);
    }
  };

  const addItem = () => {
    setItems([...items, { item_name: "", amount: 0 }]);
  };

  const removeItem = (index: number) => {
    if (items.length > 1) {
      setItems(items.filter((_, i) => i !== index));
    }
  };

  const updateItem = (
    index: number,
    field: "item_name" | "amount",
    value: string | number,
  ) => {
    const newItems = [...items];
    newItems[index] = { ...newItems[index], [field]: value };
    setItems(newItems);
  };

  const handleSave = async () => {
    if (!bill) return;

    try {
      setSaving(true);
      setError("");
      const validItems = items.filter(
        (item) => item.item_name.trim() !== "" && item.amount > 0,
      );
      const updatedBill = await api.updateItems(bill.id, validItems);
      setBill(updatedBill);
      setItems(
        updatedBill.items.map((item) => ({
          id: item.id,
          item_name: item.item_name,
          amount: item.amount,
        })),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "項目の更新に失敗しました");
    } finally {
      setSaving(false);
    }
  };

  const handleRequest = async () => {
    if (!bill) return;

    try {
      setRequesting(true);
      setError("");
      await api.requestBill(bill.id);
      await loadBill(); // 状態を更新するために再読み込み
    } catch (err) {
      setError(err instanceof Error ? err.message : "請求の確定に失敗しました");
    } finally {
      setRequesting(false);
    }
  };

  const handlePayment = async () => {
    if (!bill) return;

    try {
      setPaying(true);
      setError("");
      await api.paymentBill(bill.id);
      await loadBill(); // 状態を更新するために再読み込み
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "支払いの確定に失敗しました",
      );
    } finally {
      setPaying(false);
    }
  };

  const getStatusInfo = (status: string) => {
    const statusConfig = {
      pending: {
        badge: "bg-yellow-100 text-yellow-800 border-yellow-200",
        label: "作成中",
        description: "項目を編集して請求を確定してください",
      },
      requested: {
        badge: "bg-blue-100 text-blue-800 border-blue-200",
        label: "請求済み",
        description: "支払者による支払い確定を待っています",
      },
      paid: {
        badge: "bg-green-100 text-green-800 border-green-200",
        label: "支払済み",
        description: "支払いが完了しました",
      },
    };
    return statusConfig[status as keyof typeof statusConfig];
  };

  const canEdit =
    bill && bill.status === "pending" && user && bill.requester_id === user.id;
  const canRequest =
    bill && bill.status === "pending" && user && bill.requester_id === user.id;
  const canPay =
    bill && bill.status === "requested" && user && bill.payer_id === user.id;

  const totalAmount = items.reduce((sum, item) => sum + (item.amount || 0), 0);

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!bill) {
    return (
      <div className="card text-center py-12">
        <div className="text-6xl mb-4">📝</div>
        <h3 className="text-lg font-semibold text-gray-900 mb-2">
          {year}年{month}月の家計簿が見つかりません
        </h3>
        <p className="text-gray-600 mb-4">
          この月の家計簿はまだ作成されていません
        </p>
        <Link to="/bills" className="btn-primary">
          家計簿一覧に戻る
        </Link>
      </div>
    );
  }

  const statusInfo = getStatusInfo(bill.status);

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <Link
            to="/bills"
            className="text-blue-600 hover:text-blue-800 text-sm mb-2 inline-block"
          >
            ← 家計簿一覧に戻る
          </Link>
          <h1 className="text-3xl font-bold text-gray-900">
            {bill.year}年{bill.month}月の家計簿
          </h1>
          <div className="flex items-center gap-3 mt-2">
            <span
              className={`px-3 py-1 text-sm font-medium rounded-full border ${statusInfo.badge}`}
            >
              {statusInfo.label}
            </span>
            <p className="text-gray-600 text-sm">{statusInfo.description}</p>
          </div>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md">
          {error}
        </div>
      )}

      {/* 基本情報 */}
      <div className="card">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">基本情報</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              請求者
            </label>
            <p className="text-gray-900">{bill.requester.name}</p>
            <p className="text-sm text-gray-600">
              ID: {bill.requester.account_id}
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              支払者
            </label>
            <p className="text-gray-900">{bill.payer.name}</p>
            <p className="text-sm text-gray-600">ID: {bill.payer.account_id}</p>
          </div>
        </div>
      </div>

      {/* 金額サマリー */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="card bg-blue-50 border-blue-200">
          <h3 className="text-lg font-semibold text-blue-900 mb-2">請求金額</h3>
          <p className="text-3xl font-bold text-blue-900">
            ¥{bill.total_amount.toLocaleString()}
          </p>
        </div>
        <div className="card bg-gray-50 border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900 mb-2">項目数</h3>
          <p className="text-3xl font-bold text-gray-900">
            {bill.items.length}個
          </p>
        </div>
      </div>

      {/* 項目編集 */}
      <div className="card">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold text-gray-900">支出項目</h2>
          {canEdit && (
            <button
              onClick={addItem}
              className="text-blue-600 hover:text-blue-800 text-sm font-medium"
            >
              + 項目を追加
            </button>
          )}
        </div>

        <div className="space-y-3">
          {items.map((item, index) => (
            <div key={index} className="flex gap-3 items-center">
              <div className="flex-1">
                <input
                  type="text"
                  value={item.item_name}
                  onChange={(e) =>
                    updateItem(index, "item_name", e.target.value)
                  }
                  placeholder="項目名（例: 食費、交通費）"
                  className="form-input"
                  disabled={!canEdit}
                />
              </div>
              <div className="w-32">
                <input
                  type="number"
                  value={item.amount || ""}
                  onChange={(e) =>
                    updateItem(index, "amount", parseFloat(e.target.value) || 0)
                  }
                  placeholder="金額"
                  className="form-input"
                  disabled={!canEdit}
                  min="0"
                  step="0.01"
                />
              </div>
              {canEdit && items.length > 1 && (
                <button
                  onClick={() => removeItem(index)}
                  className="text-red-600 hover:text-red-800 p-2"
                  title="削除"
                >
                  ×
                </button>
              )}
            </div>
          ))}
        </div>

        {canEdit && (
          <div className="flex justify-between items-center mt-6 pt-4 border-t border-gray-200">
            <div className="text-sm text-gray-600">
              現在の合計: ¥{totalAmount.toLocaleString()}
            </div>
            <button
              onClick={handleSave}
              disabled={saving}
              className="btn-primary disabled:opacity-50"
            >
              {saving ? "保存中..." : "項目を保存"}
            </button>
          </div>
        )}
      </div>

      {/* アクション */}
      <div className="flex justify-center gap-4">
        {canRequest && (
          <button
            onClick={handleRequest}
            disabled={requesting}
            className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {requesting ? "確定中..." : "請求を確定する"}
          </button>
        )}
        {canPay && (
          <button
            onClick={handlePayment}
            disabled={paying}
            className="bg-green-600 text-white px-6 py-2 rounded-md hover:bg-green-700 disabled:opacity-50"
          >
            {paying ? "処理中..." : "支払いを確定する"}
          </button>
        )}
      </div>

      {/* 履歴 */}
      <div className="card">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">履歴</h2>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-gray-600">作成日時:</span>
            <span className="text-gray-900">
              {new Date(bill.created_at).toLocaleString()}
            </span>
          </div>
          {bill.request_date && (
            <div className="flex justify-between">
              <span className="text-gray-600">請求日時:</span>
              <span className="text-gray-900">
                {new Date(bill.request_date).toLocaleString()}
              </span>
            </div>
          )}
          {bill.payment_date && (
            <div className="flex justify-between">
              <span className="text-gray-600">支払日時:</span>
              <span className="text-gray-900">
                {new Date(bill.payment_date).toLocaleString()}
              </span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
