"use client";

import { useState, useEffect } from "react";
import api from "@/lib/api";
import { AuctionCard } from "@/components/auction-card";
import type { Auction, ApiResponse } from "@/lib/types";

export default function MyAuctionsPage() {
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [loading, setLoading] = useState(true);

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
