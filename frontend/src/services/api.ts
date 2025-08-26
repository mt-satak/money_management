import {
  LoginRequest,
  RegisterRequest,
  LoginResponse,
  BillResponse,
  User,
} from "../types";

const API_BASE = "/api";

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<T> {
  const token = localStorage.getItem("token");

  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      "Content-Type": "application/json",
      ...(token && { Authorization: `Bearer ${token}` }),
      ...options.headers,
    },
    ...options,
  });

  if (!response.ok) {
    const errorData = await response
      .json()
      .catch(() => ({ error: "ネットワークエラー" }));
    throw new ApiError(
      response.status,
      errorData.error || "リクエストが失敗しました",
    );
  }

  return response.json();
}

export const api = {
  // 認証
  login: (data: LoginRequest): Promise<LoginResponse> =>
    apiRequest("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  register: (data: RegisterRequest): Promise<LoginResponse> =>
    apiRequest("/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getMe: (): Promise<User> => apiRequest("/auth/me"),

  // ユーザー
  getUsers: (): Promise<{ users: User[] }> => apiRequest("/users"),

  // 家計簿
  getBills: (): Promise<{ bills: BillResponse[] }> => apiRequest("/bills"),

  getBill: (
    year: number,
    month: number,
  ): Promise<BillResponse | { bill: null }> =>
    apiRequest(`/bills/${year}/${month}`),

  createBill: (data: {
    year: number;
    month: number;
    payer_id: number;
  }): Promise<BillResponse> =>
    apiRequest("/bills", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  updateItems: (
    billId: number,
    items: Array<{ id?: number; item_name: string; amount: number }>,
  ): Promise<BillResponse> =>
    apiRequest(`/bills/${billId}/items`, {
      method: "PUT",
      body: JSON.stringify({ items }),
    }),

  requestBill: (billId: number): Promise<{ message: string }> =>
    apiRequest(`/bills/${billId}/request`, {
      method: "PUT",
    }),

  paymentBill: (billId: number): Promise<{ message: string }> =>
    apiRequest(`/bills/${billId}/payment`, {
      method: "PUT",
    }),

  deleteBill: (billId: number): Promise<{ message: string }> =>
    apiRequest(`/bills/${billId}`, {
      method: "DELETE",
    }),
};
