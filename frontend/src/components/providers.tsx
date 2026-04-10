"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth-store";
import { Toaster } from "@/components/ui/sonner";

export function Providers({ children }: { children: React.ReactNode }) {
  const { fetchProfile } = useAuthStore();

  useEffect(() => {
    const token = localStorage.getItem("access_token");
    if (token) {
      fetchProfile();
    } else {
      useAuthStore.setState({ isLoading: false });
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <>
      {children}
      <Toaster />
    </>
  );
}
