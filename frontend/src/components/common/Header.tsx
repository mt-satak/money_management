import React from "react";
import { Link, useLocation } from "react-router-dom";
import { useAuth } from "../../hooks/useAuth";

export default function Header() {
  const { user, logout } = useAuth();
  const location = useLocation();

  const isActiveLink = (path: string) => {
    if (path === "/" && location.pathname === "/") return true;
    if (path !== "/" && location.pathname.startsWith(path)) return true;
    return false;
  };

  return (
    <header className="bg-white shadow-sm border-b border-gray-200">
      <div className="container mx-auto px-4 py-4">
        <div className="flex justify-between items-center">
          <div className="flex items-center space-x-8">
            <Link to="/" className="text-2xl font-bold text-gray-900">
              💰 家計簿アプリ
            </Link>
            <nav className="flex space-x-6">
              <Link
                to="/bills"
                className={`px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                  isActiveLink("/bills")
                    ? "bg-blue-100 text-blue-700"
                    : "text-gray-600 hover:text-gray-900 hover:bg-gray-100"
                }`}
              >
                家計簿管理
              </Link>
            </nav>
          </div>
          <div className="flex items-center space-x-4">
            <span className="text-gray-600">
              こんにちは、<span className="font-medium">{user?.name}</span>さん
            </span>
            <button
              onClick={logout}
              className="text-red-600 hover:text-red-700 font-medium transition-colors"
            >
              ログアウト
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
