"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";
import Image from "next/image";
import { getErrorMessage } from "@/lib/error";
import type { TotpSetupResponse } from "@/lib/types";

export default function TotpSetupPage() {
  const router = useRouter();
  const { totpSetup, totpConfirm } = useAuthStore();
  const [setupData, setSetupData] = useState<TotpSetupResponse | null>(null);
  const [code, setCode] = useState("");
  const [backupCodes, setBackupCodes] = useState<string[] | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    totpSetup()
      .then(setSetupData)
      .catch(() => toast.error("Failed to setup 2FA"));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleConfirm = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const codes = await totpConfirm(code);
      setBackupCodes(codes);
      toast.success("Two-factor authentication enabled!");
    } catch (err) {
      toast.error(getErrorMessage(err, "Invalid code"));
    } finally {
      setLoading(false);
    }
  };

  // Show backup codes after confirmation
  if (backupCodes) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle>Save Your Backup Codes</CardTitle>
            <CardDescription>
              Store these codes in a safe place. Each code can only be used once.
              You will not be able to see them again.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-2">
              {backupCodes.map((code, i) => (
                <code
                  key={i}
                  className="bg-muted px-3 py-2 rounded text-center font-mono text-sm"
                >
                  {code}
                </code>
              ))}
            </div>
            <Button className="w-full" onClick={() => router.push("/")}>
              I&apos;ve saved my codes
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Setup Two-Factor Authentication</CardTitle>
          <CardDescription>
            Scan the QR code with your authenticator app (Google Authenticator, Authy, etc.)
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {setupData ? (
            <>
              <div className="flex justify-center">
                <Image
                  src={`data:image/png;base64,${setupData.qr_code}`}
                  alt="TOTP QR Code"
                  width={192}
                  height={192}
                  unoptimized
                  className="w-48 h-48"
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">Manual entry key</Label>
                <code className="block bg-muted px-3 py-2 rounded text-center font-mono text-sm break-all">
                  {setupData.secret}
                </code>
              </div>
              <form onSubmit={handleConfirm} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="code">Enter 6-digit code to verify</Label>
                  <Input
                    id="code"
                    value={code}
                    onChange={(e) => setCode(e.target.value)}
                    placeholder="000000"
                    maxLength={6}
                    className="text-center text-2xl tracking-widest"
                    required
                  />
                </div>
                <Button type="submit" className="w-full" disabled={loading}>
                  {loading ? "Verifying..." : "Enable 2FA"}
                </Button>
              </form>
            </>
          ) : (
            <div className="text-center text-muted-foreground">Loading...</div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
