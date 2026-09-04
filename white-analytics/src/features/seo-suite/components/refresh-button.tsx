"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { RefreshCw } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { ActionResult } from "@/lib/action-result";
import { refreshRanks } from "@/features/seo-suite/actions";
import { refreshBacklinks, refreshDomains } from "@/features/seo-suite/actions-domains";
import { s } from "@/features/seo-suite/strings";

const COOLDOWN_MS = 10 * 60 * 1000;

const ACTIONS: Record<"ranks" | "backlinks" | "domains", { run: (input: { clientId: string; propertyId: string }) => Promise<ActionResult<unknown>>; done: string }> = {
  ranks: { run: refreshRanks, done: s.ranksRefreshed },
  backlinks: { run: refreshBacklinks, done: s.backlinksRefreshed },
  domains: { run: refreshDomains, done: s.domainsRefreshed },
};

/** "Perbarui sekarang" for the suite pages — shared 10-minute cooldown shown client-side, enforced server-side. */
export function RefreshButton({ kind, clientId, propertyId, lastRunAt }: { kind: keyof typeof ACTIONS; clientId: string; propertyId: string; lastRunAt: string | null }) {
  const router = useRouter();
  const [pending, startTransition] = React.useTransition();
  const [now, setNow] = React.useState(() => Date.now());
  React.useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 30_000);
    return () => clearInterval(id);
  }, []);
  const coolingDown = lastRunAt != null && now - new Date(lastRunAt).getTime() < COOLDOWN_MS;

  const onClick = () => {
    startTransition(async () => {
      const res = await ACTIONS[kind].run({ clientId, propertyId });
      if (res.ok) {
        toast.success(ACTIONS[kind].done);
        router.refresh();
      } else {
        toast.error(res.error);
      }
    });
  };

  const button = (
    <Button size="sm" onClick={onClick} disabled={pending || coolingDown}>
      {pending ? <Spinner className="size-4" /> : <RefreshCw className="size-4" />}
      {pending ? s.refreshing : s.refreshNow}
    </Button>
  );
  if (!coolingDown) return button;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0}>{button}</span>
      </TooltipTrigger>
      <TooltipContent className="max-w-60 text-xs">{s.refreshCooldown}</TooltipContent>
    </Tooltip>
  );
}
