"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import api from "@/lib/api";
import { useTranslation } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { toast } from "sonner";
import Image from "next/image";
import { getErrorMessage } from "@/lib/error";
import type { ApiResponse, Auction } from "@/lib/types";

export default function CreateAuctionPage() {
  const router = useRouter();
  const { t } = useTranslation();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [startingPrice, setStartingPrice] = useState("");
  const [endTime, setEndTime] = useState("");
  const [imageUrl, setImageUrl] = useState("");
  const [uploading, setUploading] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const { data } = await api.post<ApiResponse<{ url: string }>>("/api/upload/image", formData, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      setImageUrl(data.data.url);
      toast.success(t("auction.imageUploaded"));
    } catch {
      toast.error(t("auction.imageFailed"));
    } finally { setUploading(false); }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const { data } = await api.post<ApiResponse<Auction>>("/api/auctions", {
        title, description, image_url: imageUrl,
        starting_price: Math.round(parseFloat(startingPrice) * 100),
        end_time: new Date(endTime).toISOString(),
      });
      toast.success(t("auction.created"));
      router.push(`/auctions/${data.data.id}`);
    } catch (err) {
      toast.error(getErrorMessage(err, t("auction.createFailed")));
    } finally { setLoading(false); }
  };

  const minEndTime = new Date(Date.now() + 2 * 60 * 1000).toISOString().slice(0, 16);

  return (
    <div className="max-w-2xl mx-auto">
      <Card>
        <CardHeader>
          <CardTitle>{t("auction.createTitle")}</CardTitle>
          <CardDescription>{t("auction.createSubtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="title">{t("auction.title")}</Label>
              <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} maxLength={255} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">{t("auction.description")}</Label>
              <textarea id="description" value={description} onChange={(e) => setDescription(e.target.value)}
                className="flex min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="image">{t("auction.image")}</Label>
              <Input id="image" type="file" accept="image/jpeg,image/png,image/webp" onChange={handleImageUpload} disabled={uploading} />
              {imageUrl && <Image src={imageUrl} alt="Preview" width={128} height={128} unoptimized className="w-32 h-32 object-cover rounded mt-2" />}
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="price">{t("auction.startingPriceLabel")}</Label>
                <Input id="price" type="number" step="0.01" min="0.01" value={startingPrice} onChange={(e) => setStartingPrice(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="endTime">{t("auction.endTime")}</Label>
                <Input id="endTime" type="datetime-local" min={minEndTime} value={endTime} onChange={(e) => setEndTime(e.target.value)} required />
              </div>
            </div>
            <Button type="submit" className="w-full" disabled={loading || uploading}>
              {loading ? t("common.processing") : t("auction.createTitle")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
