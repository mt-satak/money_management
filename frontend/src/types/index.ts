export interface User {
  id: number;
  name: string;
  account_id: string;
  created_at: string;
  updated_at: string;
}

export interface BillItem {
  id: number;
  bill_id: number;
  item_name: string;
  amount: number;
  created_at: string;
  updated_at: string;
}

export interface MonthlyBill {
  id: number;
  year: number;
  month: number;
  requester_id: number;
  payer_id: number;
  status: "pending" | "requested" | "paid";
  request_date?: string;
  payment_date?: string;
  requester: User;
  payer: User;
  items: BillItem[];
  created_at: string;
  updated_at: string;
}

export interface BillResponse extends MonthlyBill {
  total_amount: number;
}

export interface LoginRequest {
  account_id: string;
  password: string;
}

export interface RegisterRequest {
  name: string;
  account_id: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}
