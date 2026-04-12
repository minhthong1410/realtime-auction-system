"use client";

import { useAuthStore } from "@/stores/auth-store";
import { useTranslation } from "@/i18n";
import { formatCurrency, formatDate } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/error";
import { ShieldCheck, ShieldOff } from "lucide-react";

export default function ProfilePage() {
  const { user, totpDisable } = useAuthStore();
  const { t } = useTranslation();

  const handleDisableTotp = async () => {
    if (!confirm(t("profile.disableConfirm"))) return;
    try {
      await totpDisable();
      toast.success(t("profile.disabled"));
    } catch (err) {
      toast.error(getErrorMessage(err, t("profile.disableFailed")));
    }
  };

  if (!user) return null;

  return (
    <div className="max-w-lg mx-auto space-y-6">
      <div className="text-center py-6">
        <div className="inline-flex items-center justify-center h-16 w-16 rounded-full bg-primary text-primary-foreground text-2xl font-bold mb-3">
          {user.username[0].toUpperCase()}
        </div>
        <h1 className="text-xl font-bold">{user.username}</h1>
        <p className="text-sm text-muted-foreground">{user.email}</p>
      </div>

      <Card>
        <CardContent className="pt-5">
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-lg bg-muted/40 p-3.5">
              <p className="text-[11px] text-muted-foreground uppercase tracking-wide">{t("wallet.balance")}</p>
              <p className="text-lg font-bold text-primary mt-0.5">{formatCurrency(user.balance)}</p>
            </div>
            <div className="rounded-lg bg-muted/40 p-3.5">
              <p className="text-[11px] text-muted-foreground uppercase tracking-wide">{t("profile.memberSince")}</p>
              <p className="text-sm font-semibold mt-0.5">{formatDate(user.created_at)}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-5">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex items-center justify-center h-9 w-9 rounded-lg bg-success/10">
                <ShieldCheck className="h-4 w-4 text-success" />
              </div>
              <div>
                <p className="text-sm font-semibold">{t("profile.twoFactorAuth")}</p>
                <p className="text-[11px] text-muted-foreground">{t("profile.twoFactorEnabled")}</p>
              </div>
            </div>
            <Button variant="outline" size="sm" onClick={handleDisableTotp} className="gap-1.5 text-xs text-destructive hover:text-destructive h-8">
              <ShieldOff className="h-3.5 w-3.5" />
              {t("profile.disable")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
