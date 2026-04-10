"use client";

import { useState, useEffect } from "react";
import api from "@/lib/api";
import { AuctionCard } from "@/components/auction-card";
import { Button } from "@/components/ui/button";
import type { Auction, ApiResponse } from "@/lib/types";

export default function HomePage() {
  const [auctions, setAuctions] = useState<Auction[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

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
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Active Auctions</h1>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-72 bg-muted animate-pulse rounded-lg" />
          ))}
        </div>
      ) : auctions.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          No active auctions. Be the first to create one!
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {auctions.map((auction) => (
              <AuctionCard key={auction.id} auction={auction} />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                Previous
              </Button>
              <span className="flex items-center text-sm text-muted-foreground">
                Page {page} of {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
