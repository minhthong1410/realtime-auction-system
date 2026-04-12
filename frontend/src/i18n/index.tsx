"use client";

import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from "react";
import en from "./locales/en.json";
import vi from "./locales/vi.json";

export type Locale = "en" | "vi";

const dictionaries: Record<Locale, typeof en> = { en, vi };

type Dictionary = typeof en;

// Nested key accessor: t("home.heroTitle") → dictionary.home.heroTitle
type NestedKeys<T, Prefix extends string = ""> = T extends object
  ? { [K in keyof T & string]: NestedKeys<T[K], Prefix extends "" ? K : `${Prefix}.${K}`> }[keyof T & string]
  : Prefix;

type DictKey = NestedKeys<Dictionary>;

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: DictKey, params?: Record<string, string | number>) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

function getNestedValue(obj: Record<string, unknown>, path: string): string {
  const value = path.split(".").reduce<unknown>((acc, part) => {
    if (acc && typeof acc === "object") return (acc as Record<string, unknown>)[part];
    return undefined;
  }, obj);
  return typeof value === "string" ? value : path;
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");

  useEffect(() => {
    const saved = localStorage.getItem("locale") as Locale | null;
    if (saved && dictionaries[saved]) {
      setLocaleState(saved);
    }
  }, []);

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    localStorage.setItem("locale", l);
  }, []);

  const t = useCallback((key: DictKey, params?: Record<string, string | number>): string => {
    let value = getNestedValue(dictionaries[locale] as unknown as Record<string, unknown>, key);
    if (params) {
      Object.entries(params).forEach(([k, v]) => {
        value = value.replace(`{${k}}`, String(v));
      });
    }
    return value;
  }, [locale]);

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useTranslation() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useTranslation must be used within I18nProvider");
  return ctx;
}
