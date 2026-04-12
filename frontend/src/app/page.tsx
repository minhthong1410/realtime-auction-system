"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import Link from "next/link";
import api from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { useWebSocket } from "@/hooks/use-websocket";
import { useTranslation } from "@/i18n";
import { AuctionCard } from "@/components/auction-card";
import { Button } from "@/components/ui/button";
import { ArrowRight, Zap, Shield, Radio } from "lucide-react";
import type { Auction, ApiResponse, WSMessage, WSNewBid, WSAuctionEnded } from "@/lib/types";

export default function HomePage() {
  const { isAuthenticated } = useAuthStore();
  const { t } = useTranslation();
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  const wsRooms = useMemo(
    () => auctions.filter((a) => a.status === 1).map((a) => `auction:${a.id}`),
    [auctions]
  );

  useWebSocket(
    wsRooms,
    useCallback((msg: WSMessage) => {
      if (msg.type === "new_bid") {
        const data = msg.data as WSNewBid;
        setAuctions((prev) =>
          prev.map((a) =>
            a.id === data.auction_id
              ? { ...a, current_price: data.amount, bid_count: data.bid_count, winner_name: data.username }
              : a
          )
        );
      }
      if (msg.type === "auction_ended") {
        const data = msg.data as WSAuctionEnded;
        setAuctions((prev) =>
          prev.map((a) =>
            a.id === data.auction_id
              ? { ...a, status: 2, winner_name: data.winner, current_price: data.final_price }
              : a
          )
        );
      }
    }, [])
  );

  useEffect(() => {
    fetchAuctions();
  }, [page]); // eslint-disable-line react-hooks/exhaustive-deps

  const fetchAuctions = async () => {
    setLoading(true);
    try {
      const { data } = await api.get<ApiResponse<Auction[]>>("/api/auctions", {
        params: { page, size: 12 },
      });
      setAuctions(data.data || []);
      setTotalPages(data.pagination?.totalPages || 1);
    } catch {
      setAuctions([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-10">
      {page === 1 && (
        <section className="relative overflow-hidden rounded-2xl border border-border/40 bg-gradient-to-br from-primary/[0.06] via-background to-accent/40">
          <div className="relative z-10 px-8 py-12 md:px-12 md:py-16">
            <p className="text-xs font-semibold tracking-widest uppercase text-primary mb-4">{t("home.heroLabel")}</p>
            <h1 className="text-3xl md:text-5xl font-extrabold tracking-tight leading-[1.1] max-w-xl">
              {t("home.heroTitle")}<br />
              <span className="text-primary">{t("home.heroTitleHighlight")}</span>
            </h1>
            <p className="mt-4 text-muted-foreground max-w-md leading-relaxed">{t("home.heroDescription")}</p>
            <div className="mt-8 flex flex-wrap gap-3">
              {isAuthenticated ? (
                <Link href="/auctions/create">
                  <Button className="gap-2 h-10 px-5">
                    {t("home.createAuction")}
                    <ArrowRight className="h-4 w-4" />
                  </Button>
                </Link>
              ) : (
                <Link href="/register">
                  <Button className="gap-2 h-10 px-5">
                    {t("nav.getStarted")}
                    <ArrowRight className="h-4 w-4" />
                  </Button>
                </Link>
              )}
              <Link href="/">
                <Button variant="outline" className="h-10 px-5">{t("home.browseAuctions")}</Button>
              </Link>
            </div>
            <div className="mt-10 flex flex-wrap gap-6 text-[13px] text-muted-foreground">
              <div className="flex items-center gap-2">
                <div className="flex items-center justify-center h-7 w-7 rounded-md bg-primary/10">
                  <Radio className="h-3.5 w-3.5 text-primary" />
                </div>
                {t("home.featureRealtime")}
              </div>
              <div className="flex items-center gap-2">
                <div className="flex items-center justify-center h-7 w-7 rounded-md bg-primary/10">
                  <Zap className="h-3.5 w-3.5 text-primary" />
                </div>
                {t("home.featureInstant")}
              </div>
              <div className="flex items-center gap-2">
                <div className="flex items-center justify-center h-7 w-7 rounded-md bg-primary/10">
                  <Shield className="h-3.5 w-3.5 text-primary" />
                </div>
                {t("home.featureSecure")}
              </div>
            </div>
          </div>
          <div className="absolute -right-20 -top-20 h-72 w-72 rounded-full bg-primary/[0.04] blur-3xl pointer-events-none" />
          <div className="absolute -bottom-12 right-16 h-48 w-48 rounded-full bg-accent/60 blur-2xl pointer-events-none" />
        </section>
      )}

      <section>
        <div className="flex items-baseline justify-between mb-6">
          <h2 className="text-lg font-semibold tracking-tight">{t("home.activeAuctions")}</h2>
          {!loading && auctions.length > 0 && (
            <span className="text-xs text-muted-foreground">{auctions.length} {t("home.listings")}</span>
          )}
        </div>

        {loading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="rounded-xl border border-border/40 overflow-hidden">
                <div className="aspect-[4/3] bg-muted animate-pulse" />
                <div className="p-4 space-y-3">
                  <div className="h-4 bg-muted animate-pulse rounded w-3/4" />
                  <div className="h-5 bg-muted animate-pulse rounded w-1/3" />
                </div>
              </div>
            ))}
          </div>
        ) : auctions.length === 0 ? (
          <div className="text-center py-20 border border-dashed rounded-xl">
            <div className="text-4xl mb-3 opacity-30">🔨</div>
            <p className="font-medium text-muted-foreground">{t("home.noAuctions")}</p>
            <p className="text-sm text-muted-foreground/70 mt-1">{t("home.noAuctionsHint")}</p>
            {isAuthenticated && (
              <Link href="/auctions/create">
                <Button className="mt-5 gap-2" size="sm">
                  {t("home.createAuction")}
                  <ArrowRight className="h-3.5 w-3.5" />
                </Button>
              </Link>
            )}
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
              {auctions.map((auction) => (
                <AuctionCard key={auction.id} auction={auction} />
              ))}
            </div>
            {totalPages > 1 && (
              <div className="flex justify-center items-center gap-3 mt-10">
                <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)} className="h-8">
                  {t("common.previous")}
                </Button>
                <span className="text-xs text-muted-foreground tabular-nums">{page} / {totalPages}</span>
                <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)} className="h-8">
                  {t("common.next")}
                </Button>
              </div>
            )}
          </>
        )}
      </section>
    </div>
  );
}
