"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth-store";
import { Toaster } from "@/components/ui/sonner";
import { I18nProvider } from "@/i18n";

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
    <I18nProvider>
      {children}
      <Toaster />
    </I18nProvider>
  );
}
