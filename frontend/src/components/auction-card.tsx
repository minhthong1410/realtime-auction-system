"use client";

import Link from "next/link";
import Image from "next/image";
import { Badge } from "@/components/ui/badge";
import { useCountdown, formatTimeLeft } from "@/hooks/use-countdown";
import { formatCurrency } from "@/lib/format";
import { Clock, Users } from "lucide-react";
import { useTranslation } from "@/i18n";
import type { Auction } from "@/lib/types";

export function AuctionCard({ auction }: { auction: Auction }) {
  const { t } = useTranslation();
  const timeLeft = useCountdown(auction.end_time);
  const isEnded = timeLeft.total <= 0 || auction.status !== 1;
  const isUrgent = !isEnded && timeLeft.total < 300;

  return (
    <Link href={`/auctions/${auction.id}`} className="group block">
      <div className="rounded-xl border border-border/50 overflow-hidden bg-card hover:shadow-lg hover:shadow-primary/[0.04] hover:border-border transition-all duration-300">
        {/* Image */}
        <div className="aspect-[4/3] bg-muted relative overflow-hidden">
          {(auction.images?.[0] || auction.image_url) ? (
            <Image
              src={auction.images?.[0] || auction.image_url}
              alt={auction.title}
              width={600}
              height={450}
              unoptimized
              className="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500 ease-out"
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-muted-foreground/30 bg-gradient-to-br from-muted to-muted/60">
              <span className="text-3xl">🖼</span>
            </div>
          )}
          <Badge
            variant={isEnded ? "secondary" : isUrgent ? "destructive" : "default"}
            className="absolute top-3 right-3 gap-1 text-[11px] font-medium backdrop-blur-sm"
          >
            <Clock className="h-3 w-3" />
            {isEnded ? t("auction.ended") : formatTimeLeft(timeLeft)}
          </Badge>
        </div>

        {/* Content */}
        <div className="p-4 space-y-3">
          <h3 className="font-semibold text-[15px] truncate group-hover:text-primary transition-colors">
            {auction.title}
          </h3>
          <div className="flex items-end justify-between">
            <div>
              <p className="text-[11px] text-muted-foreground uppercase tracking-wide">{t("auction.currentBid")}</p>
              <p className="font-bold text-lg tracking-tight text-primary">{formatCurrency(auction.current_price)}</p>
            </div>
            <div className="flex items-center gap-1 text-muted-foreground text-xs">
              <Users className="h-3 w-3" />
              {auction.bid_count}
            </div>
          </div>
          <p className="text-[11px] text-muted-foreground/70">by {auction.seller_name}</p>
        </div>
      </div>
    </Link>
  );
}
