"use client";

import { useState, useEffect, useCallback, use } from "react";
import { useRouter } from "next/navigation";
import api from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { useWebSocket } from "@/hooks/use-websocket";
import { useCountdown, formatTimeLeft } from "@/hooks/use-countdown";
import { formatCurrency, formatRelativeTime } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import Image from "next/image";
import { getErrorMessage } from "@/lib/error";
import type { Auction, Bid, ApiResponse, WSMessage, WSNewBid, WSAuctionEnded } from "@/lib/types";

export default function AuctionDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();

  const [auction, setAuction] = useState<Auction | null>(null);
  const [bids, setBids] = useState<Bid[]>([]);
  const [bidAmount, setBidAmount] = useState("");
  const [loading, setLoading] = useState(true);
  const [bidding, setBidding] = useState(false);

  const timeLeft = useCountdown(auction?.end_time || "");
  const isEnded = (timeLeft.total <= 0 && auction !== null) || auction?.status !== 1;

  // Fetch auction + bids
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
        toast.error("Auction not found");
        router.push("/");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [id, router]);

  // WebSocket real-time updates
  useWebSocket(
    [`auction:${id}`],
    useCallback((msg: WSMessage) => {
      if (msg.type === "new_bid") {
        const data = msg.data as WSNewBid;
        setAuction((prev) =>
          prev
            ? { ...prev, current_price: data.amount, bid_count: data.bid_count, winner_name: data.username }
            : prev
        );
        setBids((prev) => [
          {
            id: crypto.randomUUID(),
            auction_id: data.auction_id,
            user_id: "",
            username: data.username,
            amount: data.amount,
            created_at: data.created_at,
          },
          ...prev,
        ]);
      }

      if (msg.type === "auction_ended") {
        const data = msg.data as WSAuctionEnded;
        setAuction((prev) =>
          prev
            ? { ...prev, status: 2, winner_name: data.winner, current_price: data.final_price }
            : prev
        );
        toast.info(`Auction ended! Winner: ${data.winner || "No bids"}`);
      }
    }, [])
  );

  const handleBid = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }

    const amount = Math.round(parseFloat(bidAmount) * 100); // Convert to cents
    if (isNaN(amount) || amount <= 0) {
      toast.error("Please enter a valid amount");
      return;
    }

    setBidding(true);
    try {
      await api.post(`/api/auctions/${id}/bid`, { amount });
      setBidAmount("");
      toast.success("Bid placed!");
    } catch (err) {
      toast.error(getErrorMessage(err, "Failed to place bid"));
    } finally {
      setBidding(false);
    }
  };

  if (loading) {
    return <div className="h-96 bg-muted animate-pulse rounded-lg" />;
  }

  if (!auction) return null;

  const minBid = (auction.current_price + 1) / 100;

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      {/* Left: Image + Description */}
      <div className="lg:col-span-2 space-y-4">
        <div className="aspect-video bg-muted rounded-lg overflow-hidden">
          {auction.image_url ? (
            <Image src={auction.image_url} alt={auction.title} width={800} height={400} unoptimized className="w-full h-full object-cover" />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-muted-foreground">
              No Image
            </div>
          )}
        </div>

        <div>
          <h1 className="text-2xl font-bold">{auction.title}</h1>
          <p className="text-sm text-muted-foreground mt-1">by {auction.seller_name}</p>
        </div>

        {auction.description && (
          <p className="text-muted-foreground whitespace-pre-wrap">{auction.description}</p>
        )}

        {/* Bid History */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Bid History ({auction.bid_count})</CardTitle>
          </CardHeader>
          <CardContent>
            {bids.length === 0 ? (
              <p className="text-muted-foreground text-sm">No bids yet. Be the first!</p>
            ) : (
              <div className="space-y-2">
                {bids.map((bid, i) => (
                  <div key={bid.id} className="flex items-center justify-between py-2">
                    <div className="flex items-center gap-2">
                      {i === 0 && <Badge variant="default">Highest</Badge>}
                      <span className="font-medium">{bid.username}</span>
                    </div>
                    <div className="text-right">
                      <span className="font-bold">{formatCurrency(bid.amount)}</span>
                      <span className="text-xs text-muted-foreground ml-2">
                        {formatRelativeTime(bid.created_at)}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Right: Bid Panel */}
      <div className="space-y-4">
        <Card>
          <CardContent className="pt-6 space-y-4">
            <div className="text-center">
              <p className="text-sm text-muted-foreground">Current Price</p>
              <p className="text-3xl font-bold">{formatCurrency(auction.current_price)}</p>
              <p className="text-xs text-muted-foreground mt-1">
                Starting: {formatCurrency(auction.starting_price)}
              </p>
            </div>

            <Separator />

            <div className="text-center">
              <Badge variant={isEnded ? "secondary" : "destructive"} className="text-sm px-3 py-1">
                {isEnded ? "Auction Ended" : formatTimeLeft(timeLeft)}
              </Badge>
            </div>

            {auction.winner_name && (
              <div className="text-center">
                <p className="text-sm text-muted-foreground">
                  {isEnded ? "Winner" : "Highest Bidder"}
                </p>
                <p className="font-semibold">{auction.winner_name}</p>
              </div>
            )}

            {!isEnded && (
              <>
                <Separator />
                <form onSubmit={handleBid} className="space-y-3">
                  <div>
                    <Input
                      type="number"
                      step="0.01"
                      min={minBid}
                      value={bidAmount}
                      onChange={(e) => setBidAmount(e.target.value)}
                      placeholder={`Min ${minBid.toFixed(2)}`}
                      className="text-lg"
                      disabled={bidding || (user?.id === auction.seller_id)}
                    />
                  </div>
                  <Button
                    type="submit"
                    className="w-full"
                    disabled={bidding || !isAuthenticated || (user?.id === auction.seller_id)}
                  >
                    {bidding
                      ? "Placing bid..."
                      : !isAuthenticated
                      ? "Login to bid"
                      : user?.id === auction.seller_id
                      ? "You own this auction"
                      : "Place Bid"}
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
