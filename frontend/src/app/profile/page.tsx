"use client";

import { useAuthStore } from "@/stores/auth-store";
import { formatCurrency, formatDate } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/error";

export default function ProfilePage() {
  const { user, totpDisable } = useAuthStore();

  const handleDisableTotp = async () => {
    if (!confirm("Are you sure you want to disable two-factor authentication?")) return;

    try {
      await totpDisable();
      toast.success("Two-factor authentication disabled");
    } catch (err) {
      toast.error(getErrorMessage(err, "Failed to disable 2FA"));
    }
  };

  if (!user) return null;

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">Username</p>
              <p className="font-medium">{user.username}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Email</p>
              <p className="font-medium">{user.email}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Balance</p>
              <p className="font-medium">{formatCurrency(user.balance)}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Member since</p>
              <p className="font-medium">{formatDate(user.created_at)}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Security</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Two-Factor Authentication</p>
              <p className="text-sm text-muted-foreground">
                Protect your account with TOTP-based 2FA
              </p>
            </div>
            <Button variant="destructive" size="sm" onClick={handleDisableTotp}>
              Disable 2FA
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
