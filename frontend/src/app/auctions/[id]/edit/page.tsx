"use client";

import { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import api from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { useTranslation } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { toast } from "sonner";
import Image from "next/image";
import { getErrorMessage } from "@/lib/error";
import { X, Plus, GripVertical } from "lucide-react";
import type { ApiResponse, Auction } from "@/lib/types";

const MAX_IMAGES = 5;

export default function EditAuctionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const { user } = useAuthStore();
  const { t } = useTranslation();

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [images, setImages] = useState<string[]>([]);
  const [endTime, setEndTime] = useState("");
  const [uploading, setUploading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [dragIdx, setDragIdx] = useState<number | null>(null);

  useEffect(() => {
    api.get<ApiResponse<Auction>>(`/api/auctions/${id}`)
      .then(({ data }) => {
        const a = data.data;
        if (a.seller_id !== user?.id) {
          toast.error(t("auction.notOwner"));
          router.push("/");
          return;
        }
        if (a.status !== 1) {
          toast.error(t("auction.ended"));
          router.push(`/auctions/${id}`);
          return;
        }
        setTitle(a.title);
        setDescription(a.description || "");
        setImages(a.images || (a.image_url ? [a.image_url] : []));
        // Format end_time for datetime-local input
        const d = new Date(a.end_time);
        const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
        setEndTime(local);
      })
      .catch(() => {
        toast.error(t("auction.notFound"));
        router.push("/");
      })
      .finally(() => setLoading(false));
  }, [id, user?.id, router, t]);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;
    if (images.length + files.length > MAX_IMAGES) {
      toast.error(t("auction.maxImages", { max: MAX_IMAGES }));
      return;
    }

    setUploading(true);
    try {
      const uploaded: string[] = [];
      for (const file of files) {
        const formData = new FormData();
        formData.append("file", file);
        const { data } = await api.post<ApiResponse<{ url: string }>>("/api/upload/image", formData, {
          headers: { "Content-Type": "multipart/form-data" },
        });
        uploaded.push(data.data.url);
      }
      setImages((prev) => [...prev, ...uploaded]);
      toast.success(t("auction.imageUploaded"));
    } catch {
      toast.error(t("auction.imageFailed"));
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  };

  const removeImage = (index: number) => {
    setImages((prev) => prev.filter((_, i) => i !== index));
  };

  const handleDragStart = (index: number) => {
    setDragIdx(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (dragIdx === null || dragIdx === index) return;
    setImages((prev) => {
      const next = [...prev];
      const [moved] = next.splice(dragIdx, 1);
      next.splice(index, 0, moved);
      return next;
    });
    setDragIdx(index);
  };

  const handleDragEnd = () => {
    setDragIdx(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await api.put<ApiResponse<Auction>>(`/api/auctions/${id}`, {
        title,
        description,
        images,
        end_time: new Date(endTime).toISOString(),
      });
      toast.success(t("auction.updated"));
      router.push(`/auctions/${id}`);
    } catch (err) {
      toast.error(getErrorMessage(err, t("auction.updateFailed")));
    } finally {
      setSaving(false);
    }
  };

  const minEndTime = new Date(Date.now() + 2 * 60 * 1000).toISOString().slice(0, 16);

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto">
        <Card>
          <CardContent className="pt-6 space-y-4">
            <div className="h-8 bg-muted animate-pulse rounded w-1/2" />
            <div className="h-32 bg-muted animate-pulse rounded" />
            <div className="h-10 bg-muted animate-pulse rounded" />
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <Card>
        <CardHeader>
          <CardTitle>{t("auction.editTitle")}</CardTitle>
          <CardDescription>{t("auction.editSubtitle")}</CardDescription>
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

            {/* Multi-image upload */}
            <div className="space-y-2">
              <Label>{t("auction.images")} ({images.length}/{MAX_IMAGES})</Label>
              {images.length > 0 && (
                <div className="grid grid-cols-5 gap-2">
                  {images.map((url, i) => (
                    <div
                      key={url}
                      draggable
                      onDragStart={() => handleDragStart(i)}
                      onDragOver={(e) => handleDragOver(e, i)}
                      onDragEnd={handleDragEnd}
                      className={`relative group aspect-square rounded-lg overflow-hidden border-2 transition cursor-grab active:cursor-grabbing ${
                        i === 0 ? "border-primary" : "border-border"
                      } ${dragIdx === i ? "opacity-50" : ""}`}
                    >
                      <Image src={url} alt={`Image ${i + 1}`} width={120} height={120} unoptimized className="w-full h-full object-cover" />
                      <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition" />
                      <button type="button" onClick={() => removeImage(i)}
                        className="absolute top-1 right-1 h-5 w-5 rounded-full bg-destructive text-destructive-foreground flex items-center justify-center opacity-0 group-hover:opacity-100 transition">
                        <X className="h-3 w-3" />
                      </button>
                      <div className="absolute top-1 left-1 opacity-0 group-hover:opacity-100 transition">
                        <GripVertical className="h-4 w-4 text-white drop-shadow" />
                      </div>
                      {i === 0 && (
                        <span className="absolute bottom-1 left-1 text-[9px] font-bold bg-primary text-primary-foreground px-1.5 py-0.5 rounded">
                          COVER
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              )}
              {images.length < MAX_IMAGES && (
                <label className="flex items-center justify-center gap-2 h-20 border-2 border-dashed rounded-lg cursor-pointer hover:border-primary/50 hover:bg-primary/[0.02] transition text-sm text-muted-foreground">
                  <Plus className="h-4 w-4" />
                  {t("auction.addImage")}
                  <input type="file" accept="image/jpeg,image/png,image/webp" multiple onChange={handleImageUpload} disabled={uploading} className="hidden" />
                </label>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="endTime">{t("auction.endTime")}</Label>
              <Input id="endTime" type="datetime-local" min={minEndTime} value={endTime} onChange={(e) => setEndTime(e.target.value)} required />
            </div>

            <div className="flex gap-3">
              <Button type="button" variant="outline" className="flex-1" onClick={() => router.back()}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" className="flex-1" disabled={saving || uploading}>
                {saving ? t("common.processing") : t("auction.saveChanges")}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
