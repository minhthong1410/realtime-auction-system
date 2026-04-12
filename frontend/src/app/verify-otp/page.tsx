"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { useTranslation } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/error";
import { Gavel, ShieldCheck, KeyRound } from "lucide-react";

export default function VerifyOtpPage() {
  const router = useRouter();
  const { verifyOtp } = useAuthStore();
  const { t } = useTranslation();
  const [code, setCode] = useState("");
  const [useBackup, setUseBackup] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      await verifyOtp(code.trim());
      router.push("/");
    } catch (err) {
      toast.error(getErrorMessage(err, t("totp.verifyFailed")));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-[65vh]">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-primary text-primary-foreground mb-4">
            <Gavel className="h-5 w-5" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight">{t("totp.verifyTitle")}</h1>
          <p className="text-sm text-muted-foreground mt-1">{t("totp.verifyDescription")}</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {!useBackup ? (
            <div className="space-y-1.5">
              <Label htmlFor="code" className="text-xs font-medium flex items-center gap-1.5">
                <ShieldCheck className="h-3.5 w-3.5" />
                {t("totp.verifyCode")}
              </Label>
              <Input
                id="code"
                value={code}
                onChange={(e) => {
                  const val = e.target.value.replace(/\D/g, "").slice(0, 6);
                  setCode(val);
                }}
                placeholder="000000"
                inputMode="numeric"
                className="text-center text-2xl tracking-[0.5em] h-12 font-mono"
                required
                autoFocus
              />
            </div>
          ) : (
            <div className="space-y-1.5">
              <Label htmlFor="backup" className="text-xs font-medium flex items-center gap-1.5">
                <KeyRound className="h-3.5 w-3.5" />
                {t("totp.backupCodes")}
              </Label>
              <Input
                id="backup"
                value={code}
                onChange={(e) => {
                  const val = e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 8);
                  setCode(val);
                }}
                placeholder="ABCD1234"
                className="text-center text-xl tracking-[0.3em] h-12 font-mono uppercase"
                required
                autoFocus
              />
            </div>
          )}

          <Button type="submit" className="w-full h-10 font-semibold" disabled={loading || (!useBackup && code.length !== 6) || (useBackup && code.length !== 8)}>
            {loading ? t("totp.verifying") : t("totp.verify")}
          </Button>
        </form>

        <button
          type="button"
          onClick={() => { setUseBackup(!useBackup); setCode(""); }}
          className="w-full text-center text-sm text-primary hover:underline mt-4 cursor-pointer"
        >
          {useBackup ? t("totp.verifyDescription") : t("totp.backupCodes")}
        </button>
      </div>
    </div>
  );
}
