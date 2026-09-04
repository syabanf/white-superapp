"use client";

import { useTransition } from "react";
import { RefreshCw } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { syncAds } from "@/features/ads/actions";
import { s } from "@/features/ads/strings";
import { t } from "@/i18n/id";

/** Tombol "Sinkronkan" — aksi mock yang menyentuh lastSyncedAt + mencatat SyncJob. */
export function SyncButton({ clientId }: { clientId: string }) {
  const [isPending, startTransition] = useTransition();

  const onClick = () => {
    startTransition(async () => {
      const res = await syncAds(clientId);
      if (res.ok) toast.success(s.syncDone);
      else toast.error(res.error);
    });
  };

  return (
    <Button variant="outline" size="sm" onClick={onClick} disabled={isPending}>
      <RefreshCw className={isPending ? "size-4 animate-spin" : "size-4"} />
      {isPending ? t.common.syncing : t.common.sync}
    </Button>
  );
}
