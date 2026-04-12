export interface User {
  id: string;
  username: string;
  email: string;
  balance: number;
  created_at: string;
}

export interface Auction {
  id: string;
  seller_id: string;
  seller_name: string;
  title: string;
  description: string;
  image_url: string;
  starting_price: number;
  current_price: number;
  winner_id: string | null;
  winner_name: string;
  status: number; // 1=active, 2=ended, 3=cancelled
  start_time: string;
  end_time: string;
  created_at: string;
  bid_count: number;
}

export interface Bid {
  id: string;
  auction_id: string;
  user_id: string;
  username: string;
  amount: number;
  created_at: string;
}

export interface Deposit {
  id: string;
  user_id: string;
  amount: number;
  status: string;
  stripe_payment_id: string;
  created_at: string;
  updated_at: string;
}

export interface ApiResponse<T> {
  code: number;
  success: boolean;
  message: string;
  data: T;
  timestamp: string;
  pagination?: {
    total: number;
    page: number;
    size: number;
    totalPages: number;
  };
}

export interface LoginResponse {
  require_totp_setup: boolean;
  totp_enabled: boolean;
  temp_token: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface TotpSetupResponse {
  qr_code: string;
  secret: string;
}

export interface TotpConfirmResponse {
  access_token: string;
  refresh_token: string;
  user: User;
  backup_codes: string[];
}

export interface DepositResponse {
  checkout_url: string;
  stripe_payment_id: string;
  amount: number;
}

export interface Withdrawal {
  id: string;
  user_id: string;
  amount: number;
  status: string;
  bank_name: string;
  account_number: string;
  account_holder: string;
  note: string;
  reviewed_at: string | null;
  created_at: string;
  updated_at: string;
}

// WebSocket messages
export interface WSMessage {
  type: string;
  data: unknown;
}

export interface WSNewBid {
  auction_id: string;
  amount: number;
  username: string;
  bid_count: number;
  time_left: number;
  created_at: string;
}

export interface WSAuctionEnded {
  auction_id: string;
  winner: string;
  final_price: number;
}

export interface WSBalanceUpdate {
  balance: number;
  reason: string;
}
