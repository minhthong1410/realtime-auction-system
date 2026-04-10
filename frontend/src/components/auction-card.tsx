"use client";

import Link from "next/link";
import Image from "next/image";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useCountdown, formatTimeLeft } from "@/hooks/use-countdown";
import { formatCurrency } from "@/lib/format";
import type { Auction } from "@/lib/types";

export function AuctionCard({ auction }: { auction: Auction }) {
  const timeLeft = useCountdown(auction.end_time);
  const isEnded = timeLeft.total <= 0 || auction.status !== 1;

  return (
    <Link href={`/auctions/${auction.id}`}>
      <Card className="overflow-hidden hover:shadow-lg transition-shadow cursor-pointer h-full">
        <div className="aspect-video bg-muted relative">
          {auction.image_url ? (
            <Image
              src={auction.image_url}
              alt={auction.title}
              width={800}
              height={400}
              unoptimized
              className="w-full h-full object-cover"
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-muted-foreground">
              No Image
            </div>
          )}
          <Badge
            variant={isEnded ? "secondary" : "default"}
            className="absolute top-2 right-2"
          >
            {isEnded ? "Ended" : formatTimeLeft(timeLeft)}
          </Badge>
        </div>
        <CardContent className="p-4 space-y-2">
          <h3 className="font-semibold truncate">{auction.title}</h3>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-muted-foreground">Current bid</p>
              <p className="font-bold text-lg">{formatCurrency(auction.current_price)}</p>
            </div>
            <div className="text-right">
              <p className="text-xs text-muted-foreground">Bids</p>
              <p className="font-semibold">{auction.bid_count}</p>
            </div>
          </div>
          <p className="text-xs text-muted-foreground">by {auction.seller_name}</p>
        </CardContent>
      </Card>
    </Link>
  );
}
