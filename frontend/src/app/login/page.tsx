"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/stores/auth-store";
import { useTranslation } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/error";
import { Gavel } from "lucide-react";

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuthStore();
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const result = await login(email, password);
      if (result.totp_enabled) router.push("/verify-otp");
      else if (result.require_totp_setup) router.push("/totp-setup");
    } catch (err) {
      toast.error(getErrorMessage(err, t("auth.loginFailed")));
    } finally { setLoading(false); }
  };

  return (
    <div className="flex items-center justify-center min-h-[65vh]">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center h-10 w-10 rounded-lg bg-primary text-primary-foreground mb-4">
            <Gavel className="h-5 w-5" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight">{t("auth.welcomeBack")}</h1>
          <p className="text-sm text-muted-foreground mt-1">{t("auth.signInSubtitle")}</p>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="email" className="text-xs font-medium">{t("auth.email")}</Label>
            <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" className="h-10" required />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="password" className="text-xs font-medium">{t("auth.password")}</Label>
            <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="h-10" required />
          </div>
          <Button type="submit" className="w-full h-10 font-semibold" disabled={loading}>
            {loading ? t("auth.signingIn") : t("nav.signIn")}
          </Button>
        </form>
        <p className="text-center text-sm text-muted-foreground mt-6">
          {t("auth.noAccount")}{" "}
          <Link href="/register" className="text-primary font-medium hover:underline">{t("auth.createOne")}</Link>
        </p>
      </div>
    </div>
  );
}
