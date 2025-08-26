import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import { useNavigate } from "react-router-dom";
import { User, LoginRequest, RegisterRequest } from "../types";
import { api } from "../services/api";

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (data: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (token) {
      api
        .getMe()
        .then(setUser)
        .catch(() => {
          localStorage.removeItem("token");
        })
        .finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, []);

  const login = async (data: LoginRequest) => {
    const response = await api.login(data);
    localStorage.setItem("token", response.token);
    setUser(response.user);
    // ログイン成功後は必ず家計簿一覧へリダイレクト
    navigate("/bills", { replace: true });
  };

  const register = async (data: RegisterRequest) => {
    const response = await api.register(data);
    localStorage.setItem("token", response.token);
    setUser(response.user);
    // 登録成功後も家計簿一覧へリダイレクト
    navigate("/bills", { replace: true });
  };

  const logout = () => {
    localStorage.removeItem("token");
    setUser(null);
    // ログアウト時にURLをルートにリセット
    navigate("/", { replace: true });
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
