"use client";

import { useState, useEffect, useCallback, use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import api from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { useWebSocket } from "@/hooks/use-websocket";
import { useCountdown, formatTimeLeft } from "@/hooks/use-countdown";
import { formatCurrency, formatRelativeTime } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import Image from "next/image";
import { getErrorMessage } from "@/lib/error";
import { ChevronLeft, ChevronRight, Clock, Pencil, Trophy, TrendingUp, User } from "lucide-react";
import { useTranslation } from "@/i18n";
import type { Auction, Bid, ApiResponse, WSMessage, WSNewBid, WSAuctionEnded } from "@/lib/types";

export default function AuctionDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();
  const { t } = useTranslation();

  const [auction, setAuction] = useState<Auction | null>(null);
  const [bids, setBids] = useState<Bid[]>([]);
  const [bidAmount, setBidAmount] = useState("");
  const [loading, setLoading] = useState(true);
  const [bidding, setBidding] = useState(false);
  const [currentImage, setCurrentImage] = useState(0);

  const timeLeft = useCountdown(auction?.end_time || "");
  const isEnded = (timeLeft.total <= 0 && auction !== null) || auction?.status !== 1;
  const isUrgent = !isEnded && timeLeft.total < 300;

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [auctionRes, bidsRes] = await Promise.all([
          api.get<ApiResponse<Auction>>(`/api/auctions/${id}`),
          api.get<ApiResponse<Bid[]>>(`/api/auctions/${id}/bids`, { params: { size: 50 } }),
        ]);
        setAuction(auctionRes.data.data);
        setBids(bidsRes.data.data || []);
      } catch {
        toast.error(t("auction.notFound"));
        router.push("/");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [id, router, t]);

  useWebSocket(
    [`auction:${id}`],
    useCallback((msg: WSMessage) => {
      if (msg.type === "new_bid") {
        const data = msg.data as WSNewBid;
        setAuction((prev) =>
          prev ? { ...prev, current_price: data.amount, bid_count: data.bid_count, winner_name: data.username } : prev
        );
        setBids((prev) => [{
          id: crypto.randomUUID(), auction_id: data.auction_id,
          user_id: "", username: data.username, amount: data.amount, created_at: data.created_at,
        }, ...prev]);
      }
      if (msg.type === "auction_ended") {
        const data = msg.data as WSAuctionEnded;
        setAuction((prev) =>
          prev ? { ...prev, status: 2, winner_name: data.winner, current_price: data.final_price } : prev
        );
        toast.info(`${t("auction.auctionEnded")} ${data.winner || t("auction.noBidsWinner")}`);
      }
      if (msg.type === "auction_updated") {
        const data = msg.data as Auction;
        setAuction(data);
      }
    }, [t])
  );

  const handleBid = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isAuthenticated) { router.push("/login"); return; }
    const amount = Math.round(parseFloat(bidAmount) * 100);
    if (isNaN(amount) || amount <= 0) { toast.error(t("auction.enterValidAmount")); return; }
    setBidding(true);
    try {
      await api.post(`/api/auctions/${id}/bid`, { amount });
      setBidAmount("");
      toast.success(t("auction.bidPlaced"));
    } catch (err) {
      toast.error(getErrorMessage(err, t("auction.bidFailed")));
    } finally { setBidding(false); }
  };

  if (loading) {
    return (
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
        <div className="lg:col-span-3 space-y-4">
          <div className="aspect-[4/3] bg-muted animate-pulse rounded-xl" />
          <div className="h-7 bg-muted animate-pulse rounded w-2/3" />
        </div>
        <div className="lg:col-span-2 h-72 bg-muted animate-pulse rounded-xl" />
      </div>
    );
  }

  if (!auction) return null;
  const minBid = (auction.current_price + 1) / 100;

  return (
    <div className="grid grid-cols-1 lg:grid-cols-5 gap-8">
      {/* Left */}
      <div className="lg:col-span-3 space-y-5">
        <div className="aspect-[4/3] bg-muted rounded-xl overflow-hidden relative">
          {(auction.images?.length > 0 || auction.image_url) ? (
            <>
              <Image
                src={auction.images?.[currentImage] || auction.image_url}
                alt={auction.title}
                width={800} height={600} unoptimized
                className="w-full h-full object-cover"
              />
              {auction.images?.length > 1 && (
                <>
                  <button
                    onClick={() => setCurrentImage((p) => (p - 1 + auction.images.length) % auction.images.length)}
                    className="absolute left-2 top-1/2 -translate-y-1/2 h-8 w-8 rounded-full bg-black/50 text-white flex items-center justify-center hover:bg-black/70 transition"
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => setCurrentImage((p) => (p + 1) % auction.images.length)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 h-8 w-8 rounded-full bg-black/50 text-white flex items-center justify-center hover:bg-black/70 transition"
                  >
                    <ChevronRight className="h-4 w-4" />
                  </button>
                  <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex gap-1.5">
                    {auction.images.map((_, i) => (
                      <button key={i} onClick={() => setCurrentImage(i)}
                        className={`h-2 w-2 rounded-full transition ${i === currentImage ? "bg-white" : "bg-white/40"}`} />
                    ))}
                  </div>
                </>
              )}
            </>
          ) : (
            <div className="w-full h-full flex items-center justify-center text-muted-foreground/30">
              <span className="text-5xl">🖼</span>
            </div>
          )}
        </div>

        {auction.images?.length > 1 && (
          <div className="flex gap-2 overflow-x-auto">
            {auction.images.map((img, i) => (
              <button key={i} onClick={() => setCurrentImage(i)}
                className={`flex-shrink-0 w-20 h-20 rounded-lg overflow-hidden border-2 transition ${i === currentImage ? "border-primary" : "border-transparent opacity-60 hover:opacity-100"}`}>
                <Image src={img} alt={`${auction.title} ${i + 1}`} width={80} height={80} unoptimized className="w-full h-full object-cover" />
              </button>
            ))}
          </div>
        )}

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{auction.title}</h1>
            <div className="flex items-center gap-1.5 mt-1.5 text-sm text-muted-foreground">
              <User className="h-3.5 w-3.5" />
              {auction.seller_name}
            </div>
          </div>
          {isAuthenticated && user?.id === auction.seller_id && auction.status === 1 && (
            <Link href={`/auctions/${auction.id}/edit`}>
              <Button variant="outline" size="sm" className="gap-1.5">
                <Pencil className="h-3.5 w-3.5" />
                {t("common.edit")}
              </Button>
            </Link>
          )}
        </div>

        {auction.description && (
          <p className="text-muted-foreground text-[15px] leading-relaxed whitespace-pre-wrap">{auction.description}</p>
        )}

        {/* Bid History */}
        <div>
          <div className="flex items-baseline justify-between mb-3">
            <h2 className="text-sm font-semibold">{t("auction.bidHistory")}</h2>
            <span className="text-xs text-muted-foreground">{auction.bid_count} {t("auction.bids")}</span>
          </div>
          {bids.length === 0 ? (
            <div className="text-center py-10 border border-dashed rounded-lg">
              <p className="text-sm text-muted-foreground">{t("auction.noBids")}</p>
            </div>
          ) : (
            <div className="divide-y rounded-lg border overflow-hidden">
              {bids.map((bid, i) => (
                <div key={bid.id} className={`flex items-center justify-between px-4 py-3 ${i === 0 ? "bg-primary/[0.04]" : ""}`}>
                  <div className="flex items-center gap-2.5">
                    <span className={`inline-flex items-center justify-center h-7 w-7 rounded-full text-[11px] font-bold ${
                      i === 0 ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                    }`}>
                      {bid.username[0].toUpperCase()}
                    </span>
                    <div>
                      <span className="text-sm font-medium">{bid.username}</span>
                      {i === 0 && <Badge variant="default" className="ml-2 text-[10px] px-1.5 py-0">{t("auction.top")}</Badge>}
                      <p className="text-[11px] text-muted-foreground">{formatRelativeTime(bid.created_at)}</p>
                    </div>
                  </div>
                  <span className="font-bold text-sm">{formatCurrency(bid.amount)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Right: Bid Panel */}
      <div className="lg:col-span-2">
        <Card className="sticky top-20">
          <CardContent className="pt-6 space-y-5">
            {/* Price */}
            <div className="text-center">
              <p className="text-[11px] text-muted-foreground uppercase tracking-widest">{t("auction.currentPrice")}</p>
              <p className="text-3xl font-extrabold text-primary tracking-tight mt-1">{formatCurrency(auction.current_price)}</p>
              <p className="text-[11px] text-muted-foreground mt-1">{t("auction.startedAt")} {formatCurrency(auction.starting_price)}</p>
            </div>

            <Separator />

            {/* Timer */}
            <div className="text-center">
              <Badge
                variant={isEnded ? "secondary" : isUrgent ? "destructive" : "outline"}
                className="gap-1.5 text-xs px-3 py-1"
              >
                <Clock className="h-3 w-3" />
                {isEnded ? t("auction.ended") : formatTimeLeft(timeLeft)}
              </Badge>
            </div>

            {/* Winner */}
            {auction.winner_name && (
              <div className="text-center p-3 rounded-lg bg-muted/50">
                <div className="flex items-center justify-center gap-1 text-[11px] text-muted-foreground mb-0.5">
                  <Trophy className="h-3 w-3" />
                  {isEnded ? t("auction.winner") : t("auction.leading")}
                </div>
                <p className="font-semibold text-sm">{auction.winner_name}</p>
              </div>
            )}

            {/* Stats */}
            <div className="grid grid-cols-2 gap-3">
              <div className="text-center p-2.5 rounded-lg bg-muted/50">
                <p className="text-[11px] text-muted-foreground">{t("auction.bids")}</p>
                <p className="font-bold">{auction.bid_count}</p>
              </div>
              <div className="text-center p-2.5 rounded-lg bg-muted/50">
                <p className="text-[11px] text-muted-foreground">{t("auction.minBid")}</p>
                <p className="font-bold">{formatCurrency(auction.current_price + 1)}</p>
              </div>
            </div>

            {/* Bid Form */}
            {!isEnded && (
              <>
                <Separator />
                <form onSubmit={handleBid} className="space-y-2.5">
                  <Input
                    type="number" step="0.01" min={minBid} value={bidAmount}
                    onChange={(e) => setBidAmount(e.target.value)}
                    placeholder={`$${minBid.toFixed(2)} or higher`}
                    className="h-11 text-center text-lg font-semibold"
                    disabled={bidding || (user?.id === auction.seller_id)}
                  />
                  <Button
                    type="submit" className="w-full h-10 gap-2 font-semibold"
                    disabled={bidding || !isAuthenticated || (user?.id === auction.seller_id)}
                  >
                    <TrendingUp className="h-4 w-4" />
                    {bidding ? t("auction.placing") : !isAuthenticated ? t("auction.signInToBid")
                      : user?.id === auction.seller_id ? t("auction.yourAuction") : t("auction.placeBid")}
                  </Button>
                </form>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
