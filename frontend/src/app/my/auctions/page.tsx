"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import api from "@/lib/api";
import { useWebSocket } from "@/hooks/use-websocket";
import { AuctionCard } from "@/components/auction-card";
import type { Auction, ApiResponse, WSMessage, WSNewBid, WSAuctionEnded } from "@/lib/types";

export default function MyAuctionsPage() {
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [loading, setLoading] = useState(true);

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
    api.get<ApiResponse<Auction[]>>("/api/my/auctions", { params: { size: 50 } })
      .then(({ data }) => setAuctions(data.data || []))
      .catch(() => setAuctions([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">My Auctions</h1>

      {loading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-72 bg-muted animate-pulse rounded-lg" />
          ))}
        </div>
      ) : auctions.length === 0 ? (
        <p className="text-muted-foreground">You haven&apos;t created any auctions yet.</p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {auctions.map((a) => (
            <AuctionCard key={a.id} auction={a} />
          ))}
        </div>
      )}
    </div>
  );
}
