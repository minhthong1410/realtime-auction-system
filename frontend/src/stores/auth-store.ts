import { create } from "zustand";
import api from "@/lib/api";
import type { User, ApiResponse, LoginResponse, TokenResponse, TotpSetupResponse, TotpConfirmResponse } from "@/lib/types";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;

  // Temp token for TOTP flow
  tempToken: string | null;
  requireTotpSetup: boolean;

  // Actions
  register: (username: string, email: string, password: string) => Promise<LoginResponse>;
  login: (email: string, password: string) => Promise<LoginResponse>;
  verifyOtp: (code: string) => Promise<void>;
  totpSetup: () => Promise<TotpSetupResponse>;
  totpConfirm: (code: string) => Promise<string[]>; // returns backup codes
  totpDisable: () => Promise<void>;
  fetchProfile: () => Promise<void>;
  logout: () => void;
  setBalance: (balance: number) => void;
  setTokens: (accessToken: string, refreshToken: string, user: User) => void;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  tempToken: null,
  requireTotpSetup: false,

  register: async (username, email, password) => {
    const { data } = await api.post<ApiResponse<LoginResponse>>("/api/auth/register", {
      username,
      email,
      password,
    });
    const result = data.data;
    set({ tempToken: result.temp_token, requireTotpSetup: true });
    return result;
  },

  login: async (email, password) => {
    const { data } = await api.post<ApiResponse<LoginResponse>>("/api/auth/login", {
      email,
      password,
    });
    const result = data.data;
    set({ tempToken: result.temp_token, requireTotpSetup: result.require_totp_setup });
    return result;
  },

  verifyOtp: async (code) => {
    const { tempToken } = get();
    const { data } = await api.post<ApiResponse<TokenResponse>>("/api/auth/verify-otp", {
      temp_token: tempToken,
      code,
    });
    const tokens = data.data;
    get().setTokens(tokens.access_token, tokens.refresh_token, tokens.user);
    set({ tempToken: null });
  },

  totpSetup: async () => {
    const { tempToken } = get();
    const { data } = await api.post<ApiResponse<TotpSetupResponse>>("/api/auth/totp/setup", {
      temp_token: tempToken,
    });
    return data.data;
  },

  totpConfirm: async (code) => {
    const { tempToken } = get();
    const { data } = await api.post<ApiResponse<TotpConfirmResponse>>("/api/auth/totp/confirm", {
      temp_token: tempToken,
      code,
    });
    const result = data.data;
    get().setTokens(result.access_token, result.refresh_token, result.user);
    set({ tempToken: null, requireTotpSetup: false });
    return result.backup_codes;
  },

  totpDisable: async () => {
    await api.post("/api/totp/disable");
  },

  fetchProfile: async () => {
    try {
      const { data } = await api.get<ApiResponse<User>>("/api/me");
      set({ user: data.data, isAuthenticated: true, isLoading: false });
    } catch {
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },

  logout: () => {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    set({ user: null, isAuthenticated: false, tempToken: null });
  },

  setBalance: (balance) => {
    set((state) => ({
      user: state.user ? { ...state.user, balance } : null,
    }));
  },

  setTokens: (accessToken, refreshToken, user) => {
    localStorage.setItem("access_token", accessToken);
    localStorage.setItem("refresh_token", refreshToken);
    set({ user, isAuthenticated: true, isLoading: false });
  },
}));
