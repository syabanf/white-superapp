"use client";

import { useTransition } from "react";
import { RefreshCw } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { syncSeo } from "@/features/seo/actions";
import { s } from "@/features/seo/strings";
import { t } from "@/i18n/id";

/** "Sinkronkan" for Search Console + GA4 (real when a Google connection exists). */
export function SeoSyncButton({ clientId }: { clientId: string }) {
  const [pending, startTransition] = useTransition();
  const run = () =>
    startTransition(async () => {
      const res = await syncSeo(clientId);
      if (!res.ok) return void toast.error(res.error);
      if (res.data.mode === "real") toast.success(s.syncDoneReal(res.data.daily, res.data.ga4));
      else toast.info(s.syncDoneDemo);
    });
  return (
    <Button variant="outline" size="sm" onClick={run} disabled={pending}>
      <RefreshCw className={pending ? "size-4 animate-spin" : "size-4"} />
      {pending ? t.common.syncing : t.common.sync}
    </Button>
  );
}
