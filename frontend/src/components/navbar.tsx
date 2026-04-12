"use client";

import Link from "next/link";
import { useAuthStore } from "@/stores/auth-store";
import { useBalanceSync } from "@/hooks/use-websocket";
import { useTranslation, type Locale } from "@/i18n";
import { formatCurrency } from "@/lib/format";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Gavel, Plus, Wallet, User, LogOut, ChevronDown, Globe } from "lucide-react";

const LANG_LABELS: Record<Locale, string> = { en: "EN", vi: "VI" };

export function Navbar() {
  const { user, isAuthenticated, logout } = useAuthStore();
  const { t, locale, setLocale } = useTranslation();
  useBalanceSync();

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/90 backdrop-blur-md">
      <div className="w-full max-w-[1200px] mx-auto px-5 sm:px-8 flex h-14 items-center justify-between">
        <div className="flex items-center gap-8">
          <Link href="/" className="flex items-center gap-2.5 group">
            <div className="flex items-center justify-center h-7 w-7 rounded-md bg-primary text-primary-foreground transition-transform group-hover:scale-105">
              <Gavel className="h-3.5 w-3.5" />
            </div>
            <span className="font-bold text-lg tracking-tight">Auction</span>
          </Link>
          <nav className="hidden md:flex items-center gap-1 text-sm">
            <Link href="/" className="px-3 py-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors">
              {t("nav.browse")}
            </Link>
            {isAuthenticated && (
              <>
                <Link href="/auctions/create" className="px-3 py-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors flex items-center gap-1.5">
                  <Plus className="h-3.5 w-3.5" />
                  {t("nav.create")}
                </Link>
                <Link href="/my/auctions" className="px-3 py-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors">
                  {t("nav.myAuctions")}
                </Link>
              </>
            )}
          </nav>
        </div>

        <div className="flex items-center gap-2.5">
          {/* Language Switcher */}
          <DropdownMenu>
            <DropdownMenuTrigger>
              <Button variant="ghost" size="sm" className="gap-1 h-8 px-2 text-xs text-muted-foreground">
                <Globe className="h-3.5 w-3.5" />
                {LANG_LABELS[locale]}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-28">
              <DropdownMenuItem onClick={() => setLocale("en")} className={`text-sm cursor-pointer ${locale === "en" ? "font-semibold" : ""}`}>
                🇺🇸 English
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setLocale("vi")} className={`text-sm cursor-pointer ${locale === "vi" ? "font-semibold" : ""}`}>
                🇻🇳 Tiếng Việt
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {isAuthenticated && user ? (
            <>
              <Link href="/wallet">
                <Button variant="outline" size="sm" className="gap-1.5 font-semibold text-xs h-8 border-border/60 hover:border-primary/30 hover:bg-primary/5 transition-all">
                  <Wallet className="h-3.5 w-3.5 text-primary" />
                  {formatCurrency(user.balance)}
                </Button>
              </Link>
              <DropdownMenu>
                <DropdownMenuTrigger>
                  <Button variant="ghost" size="sm" className="gap-1.5 h-8 px-2">
                    <span className="inline-flex items-center justify-center h-6 w-6 rounded-full bg-primary text-primary-foreground text-[11px] font-bold">
                      {user.username[0].toUpperCase()}
                    </span>
                    <span className="hidden sm:inline text-sm font-medium">{user.username}</span>
                    <ChevronDown className="h-3 w-3 text-muted-foreground" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-44">
                  <DropdownMenuItem>
                    <Link href="/profile" className="flex items-center gap-2 w-full text-sm">
                      <User className="h-3.5 w-3.5" />
                      {t("nav.profile")}
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <Link href="/wallet" className="flex items-center gap-2 w-full text-sm">
                      <Wallet className="h-3.5 w-3.5" />
                      {t("nav.wallet")}
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={logout} className="flex items-center gap-2 text-destructive text-sm cursor-pointer">
                    <LogOut className="h-3.5 w-3.5" />
                    {t("nav.signOut")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ) : (
            <div className="flex items-center gap-2">
              <Link href="/login">
                <Button variant="ghost" size="sm" className="text-sm h-8">{t("nav.signIn")}</Button>
              </Link>
              <Link href="/register">
                <Button size="sm" className="text-sm h-8">{t("nav.getStarted")}</Button>
              </Link>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
