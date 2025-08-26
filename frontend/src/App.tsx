import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "./hooks/useAuth";
import LoginPage from "./pages/LoginPage";
import BillsListPage from "./pages/BillsListPage";
import BillDetailPage from "./pages/BillDetailPage";
import Header from "./components/common/Header";

function App() {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">アプリケーションを読み込み中...</p>
        </div>
      </div>
    );
  }

  // 未認証ユーザーは全てのパスでログインページを表示
  if (!user) {
    return (
      <Routes>
        <Route path="*" element={<LoginPage />} />
      </Routes>
    );
  }

  // 認証済みユーザー用ルート
  return (
    <div className="min-h-screen bg-gray-50">
      <Header />
      <main className="container mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<Navigate to="/bills" replace />} />
          <Route path="/bills" element={<BillsListPage />} />
          <Route path="/bills/:year/:month" element={<BillDetailPage />} />
          <Route path="*" element={<Navigate to="/bills" replace />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
