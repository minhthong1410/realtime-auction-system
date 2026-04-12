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
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/error";
import { Wallet, ArrowDownToLine, ArrowUpFromLine, Clock, CheckCircle2, XCircle } from "lucide-react";
import { useTranslation } from "@/i18n";
import type { Deposit, Withdrawal, ApiResponse, DepositResponse } from "@/lib/types";

export default function WalletPage() {
  const { user } = useAuthStore();
  const { t } = useTranslation();
  const [depositAmount, setDepositAmount] = useState("");
  const [deposits, setDeposits] = useState<Deposit[]>([]);
  const [depositLoading, setDepositLoading] = useState(false);

  const [withdrawalForm, setWithdrawalForm] = useState({
    amount: "", bank_name: "", account_number: "", account_holder: "", note: "",
  });
  const [withdrawals, setWithdrawals] = useState<Withdrawal[]>([]);
  const [withdrawalLoading, setWithdrawalLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    api.get<ApiResponse<Deposit[]>>("/api/wallet/deposits", { params: { size: 20 } })
      .then(({ data }) => { if (!cancelled) setDeposits(data.data || []); })
      .catch(() => {});
    api.get<ApiResponse<Withdrawal[]>>("/api/wallet/withdrawals", { params: { size: 20 } })
      .then(({ data }) => { if (!cancelled) setWithdrawals(data.data || []); })
      .catch(() => {});
    return () => { cancelled = true; };
  }, []);

  const handleDeposit = async (e: React.FormEvent) => {
    e.preventDefault();
    const cents = Math.round(parseFloat(depositAmount) * 100);
    if (isNaN(cents) || cents < 100) { toast.error(t("wallet.minDepositError")); return; }
    setDepositLoading(true);
    try {
      const { data } = await api.post<ApiResponse<DepositResponse>>("/api/wallet/deposit", { amount: cents });
      window.location.href = data.data.checkout_url;
    } catch (err) {
      toast.error(getErrorMessage(err, t("wallet.depositFailed")));
      setDepositLoading(false);
    }
  };

  const handleWithdrawal = async (e: React.FormEvent) => {
    e.preventDefault();
    const cents = Math.round(parseFloat(withdrawalForm.amount) * 100);
    if (isNaN(cents) || cents < 500) { toast.error(t("wallet.minWithdrawalError")); return; }
    if (!withdrawalForm.bank_name || !withdrawalForm.account_number || !withdrawalForm.account_holder) {
      toast.error(t("wallet.fillBankDetails")); return;
    }
    setWithdrawalLoading(true);
    try {
      const { data } = await api.post<ApiResponse<Withdrawal>>("/api/wallet/withdrawal", {
        amount: cents, bank_name: withdrawalForm.bank_name,
        account_number: withdrawalForm.account_number,
        account_holder: withdrawalForm.account_holder, note: withdrawalForm.note,
      });
      toast.success(t("wallet.withdrawalSubmitted"));
      setWithdrawals((prev) => [data.data, ...prev]);
      setWithdrawalForm({ amount: "", bank_name: "", account_number: "", account_holder: "", note: "" });
    } catch (err) {
      toast.error(getErrorMessage(err, t("wallet.withdrawalFailed")));
    } finally { setWithdrawalLoading(false); }
  };

  const StatusIcon = ({ status }: { status: string }) => {
    switch (status) {
      case "completed": return <CheckCircle2 className="h-4 w-4 text-success" />;
      case "pending": return <Clock className="h-4 w-4 text-warning" />;
      default: return <XCircle className="h-4 w-4 text-destructive" />;
    }
  };

  const statusVariant = (s: string): "default" | "secondary" | "destructive" | "outline" =>
    s === "completed" ? "default" : s === "pending" ? "secondary" : "destructive";

  return (
    <div className="max-w-xl mx-auto space-y-6">
      {/* Balance */}
      <div className="text-center py-8">
        <div className="inline-flex items-center justify-center h-11 w-11 rounded-xl bg-primary/10 mb-3">
          <Wallet className="h-5 w-5 text-primary" />
        </div>
        <p className="text-xs text-muted-foreground uppercase tracking-widest mb-1">{t("wallet.balance")}</p>
        <p className="text-4xl font-extrabold tracking-tight">{formatCurrency(user?.balance || 0)}</p>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="deposit">
        <TabsList className="w-full">
          <TabsTrigger value="deposit" className="gap-1.5 text-sm">
            <ArrowDownToLine className="h-3.5 w-3.5" /> {t("wallet.deposit")}
          </TabsTrigger>
          <TabsTrigger value="withdrawal" className="gap-1.5 text-sm">
            <ArrowUpFromLine className="h-3.5 w-3.5" /> {t("wallet.withdraw")}
          </TabsTrigger>
        </TabsList>

        {/* Deposit */}
        <TabsContent value="deposit" className="space-y-4 mt-4">
          <Card>
            <CardContent className="pt-5">
              <form onSubmit={handleDeposit} className="flex gap-3">
                <Input
                  type="number" step="0.01" min="1" value={depositAmount}
                  onChange={(e) => setDepositAmount(e.target.value)}
                  placeholder="Amount in USD" className="h-10 flex-1"
                />
                <Button type="submit" disabled={depositLoading} className="h-10 px-6">
                  {depositLoading ? "..." : t("wallet.deposit")}
                </Button>
              </form>
              <p className="text-[11px] text-muted-foreground mt-2">{t("wallet.minDeposit")}</p>
            </CardContent>
          </Card>

          <TransactionList
            items={deposits}
            emptyIcon={<ArrowDownToLine className="h-6 w-6" />}
            emptyText={t("wallet.noDeposits")}
            renderItem={(d) => (
              <div key={d.id} className="flex items-center justify-between py-3">
                <div className="flex items-center gap-3">
                  <StatusIcon status={d.status} />
                  <div>
                    <p className="text-sm font-semibold">{formatCurrency(d.amount)}</p>
                    <p className="text-[11px] text-muted-foreground">{formatDate(d.created_at)}</p>
                  </div>
                </div>
                <Badge variant={statusVariant(d.status)} className="text-[11px]">{d.status}</Badge>
              </div>
            )}
          />
        </TabsContent>

        {/* Withdrawal */}
        <TabsContent value="withdrawal" className="space-y-4 mt-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold">{t("wallet.bankTransfer")}</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleWithdrawal} className="space-y-3">
                <div>
                  <Label className="text-xs">{t("wallet.amountUsd")}</Label>
                  <Input type="number" step="0.01" min="5" value={withdrawalForm.amount}
                    onChange={(e) => setWithdrawalForm((f) => ({ ...f, amount: e.target.value }))}
                    placeholder="Min $5.00" className="h-9 mt-1" />
                </div>
                <div>
                  <Label className="text-xs">{t("wallet.bankName")}</Label>
                  <Input value={withdrawalForm.bank_name}
                    onChange={(e) => setWithdrawalForm((f) => ({ ...f, bank_name: e.target.value }))}
                    placeholder="e.g. Chase" className="h-9 mt-1" />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label className="text-xs">{t("wallet.accountNumber")}</Label>
                    <Input value={withdrawalForm.account_number}
                      onChange={(e) => setWithdrawalForm((f) => ({ ...f, account_number: e.target.value }))}
                      placeholder="Account #" className="h-9 mt-1" />
                  </div>
                  <div>
                    <Label className="text-xs">{t("wallet.accountHolder")}</Label>
                    <Input value={withdrawalForm.account_holder}
                      onChange={(e) => setWithdrawalForm((f) => ({ ...f, account_holder: e.target.value }))}
                      placeholder="Full name" className="h-9 mt-1" />
                  </div>
                </div>
                <div>
                  <Label className="text-xs">{t("wallet.note")} ({t("common.optional")})</Label>
                  <Input value={withdrawalForm.note}
                    onChange={(e) => setWithdrawalForm((f) => ({ ...f, note: e.target.value }))}
                    placeholder="Optional" className="h-9 mt-1" />
                </div>
                <Button type="submit" disabled={withdrawalLoading} className="w-full h-10">
                  {withdrawalLoading ? t("common.processing") : t("wallet.submitWithdrawal")}
                </Button>
              </form>
              <p className="text-[11px] text-muted-foreground mt-2">{t("wallet.reviewTime")}</p>
            </CardContent>
          </Card>

          <TransactionList
            items={withdrawals}
            emptyIcon={<ArrowUpFromLine className="h-6 w-6" />}
            emptyText={t("wallet.noWithdrawals")}
            renderItem={(w) => (
              <div key={w.id} className="flex items-center justify-between py-3">
                <div className="flex items-center gap-3">
                  <StatusIcon status={w.status} />
                  <div>
                    <p className="text-sm font-semibold">{formatCurrency(w.amount)}</p>
                    <p className="text-[11px] text-muted-foreground">
                      {w.bank_name} · {w.account_holder}
                    </p>
                    <p className="text-[11px] text-muted-foreground">{formatDate(w.created_at)}</p>
                  </div>
                </div>
                <Badge variant={statusVariant(w.status)} className="text-[11px]">{w.status}</Badge>
              </div>
            )}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function TransactionList<T extends { id: string }>({
  items, emptyIcon, emptyText, renderItem,
}: {
  items: T[]; emptyIcon: React.ReactNode; emptyText: string;
  renderItem: (item: T) => React.ReactNode;
}) {
  return (
    <Card>
      <CardContent className="pt-5">
        {items.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground/40">
            {emptyIcon}
            <p className="text-xs mt-2">{emptyText}</p>
          </div>
        ) : (
          <div className="divide-y">{items.map(renderItem)}</div>
        )}
      </CardContent>
    </Card>
  );
}
