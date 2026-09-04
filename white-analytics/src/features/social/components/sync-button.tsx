"use client";

import { useTransition } from "react";
import { RefreshCw } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { syncSocial } from "@/features/social/actions";
import type { Platform } from "@/features/social/lib";
import { s } from "@/features/social/strings";
import { t } from "@/i18n/id";

/** Tombol "Sinkronkan" — memicu upsert snapshot hari ini (mode demo) + toast. */
export function SyncButton({ clientId, platform }: { clientId: string; platform: Platform }) {
  const [pending, startTransition] = useTransition();

  const run = () =>
    startTransition(async () => {
      const res = await syncSocial({ clientId, platform });
      if (res.ok) toast.success(s.syncSuccess, { description: `${res.data.updated} ${s.accountsUpdated}` });
      else toast.error(t.common.error, { description: res.error });
    });

  return (
    <Button variant="outline" size="sm" onClick={run} disabled={pending}>
      <RefreshCw className={pending ? "size-4 animate-spin" : "size-4"} />
      {pending ? t.common.syncing : t.common.sync}
    </Button>
  );
}
