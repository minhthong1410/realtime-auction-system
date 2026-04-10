"use client";

import { useState, useEffect } from "react";
import { useAuthStore } from "@/stores/auth-store";
import api from "@/lib/api";
import { formatCurrency, formatDate } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import type { Deposit, ApiResponse, DepositResponse } from "@/lib/types";

export default function WalletPage() {
  const { user } = useAuthStore();
  const [amount, setAmount] = useState("");
  const [deposits, setDeposits] = useState<Deposit[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchDeposits();
  }, []);

  const fetchDeposits = async () => {
    try {
      const { data } = await api.get<ApiResponse<Deposit[]>>("/api/wallet/deposits", {
        params: { size: 20 },
      });
      setDeposits(data.data || []);
    } catch {
      // ignore
    }
  };

  const handleDeposit = async (e: React.FormEvent) => {
    e.preventDefault();
    const cents = Math.round(parseFloat(amount) * 100);
    if (isNaN(cents) || cents < 100) {
      toast.error("Minimum deposit is $1.00");
      return;
    }

    setLoading(true);
    try {
      const { data } = await api.post<ApiResponse<DepositResponse>>("/api/wallet/deposit", {
        amount: cents,
      });
      // Redirect to Stripe Checkout
      window.location.href = data.data.checkout_url;
    } catch (err: any) {
      toast.error(err.response?.data?.message || "Failed to create deposit");
      setLoading(false);
    }
  };

  const statusColor = (status: string) => {
    switch (status) {
      case "completed": return "default";
      case "pending": return "secondary";
      case "failed": return "destructive";
      default: return "outline";
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Wallet</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-center">
            <p className="text-sm text-muted-foreground">Available Balance</p>
            <p className="text-4xl font-bold">{formatCurrency(user?.balance || 0)}</p>
          </div>

          <form onSubmit={handleDeposit} className="flex gap-2">
            <div className="flex-1">
              <Label htmlFor="amount" className="sr-only">Amount</Label>
              <Input
                id="amount"
                type="number"
                step="0.01"
                min="1"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder="Amount in USD"
              />
            </div>
            <Button type="submit" disabled={loading}>
              {loading ? "Processing..." : "Deposit"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Deposit History</CardTitle>
        </CardHeader>
        <CardContent>
          {deposits.length === 0 ? (
            <p className="text-muted-foreground text-sm">No deposits yet</p>
          ) : (
            <div className="space-y-3">
              {deposits.map((d) => (
                <div key={d.id} className="flex items-center justify-between py-2 border-b last:border-0">
                  <div>
                    <p className="font-medium">{formatCurrency(d.amount)}</p>
                    <p className="text-xs text-muted-foreground">{formatDate(d.created_at)}</p>
                  </div>
                  <Badge variant={statusColor(d.status) as any}>{d.status}</Badge>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
