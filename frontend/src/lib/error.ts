import { AxiosError } from "axios";
import type { ApiResponse } from "./types";

export function getErrorMessage(error: unknown, fallback = "Something went wrong"): string {
  if (error instanceof AxiosError) {
    const data = error.response?.data as ApiResponse<unknown> | undefined;
    return data?.message || fallback;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return fallback;
}
