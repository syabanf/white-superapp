"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { PlayCircle } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { runAudit } from "@/features/seo/actions";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";

const COOLDOWN_MS = 10 * 60 * 1000;

export function RunAuditButton({
  clientId,
  propertyId,
  url,
  lastRunAt,
}: {
  clientId: string;
  propertyId: string;
  url?: string;
  /** ISO timestamp of the most recent audit (any strategy) */
  lastRunAt: string | null;
}) {
  const router = useRouter();
  const [pending, startTransition] = React.useTransition();
  // Cooldown is evaluated on the client at render time; the server action re-checks it.
  const [now, setNow] = React.useState(() => Date.now());
  React.useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 30_000);
    return () => clearInterval(id);
  }, []);
  const coolingDown = lastRunAt != null && now - new Date(lastRunAt).getTime() < COOLDOWN_MS;

  const onClick = () => {
    startTransition(async () => {
      const res = await runAudit(clientId, propertyId, url);
      if (res.ok) {
        toast.success(res.data.message);
        router.refresh();
      } else {
        toast.error(res.error);
      }
    });
  };

  const button = (
    <Button size="sm" onClick={onClick} disabled={pending || coolingDown}>
      {pending ? <Spinner className="size-4" /> : <PlayCircle className="size-4" />}
      {pending ? t.seo.runningAudit : t.seo.runAudit}
    </Button>
  );

  if (!coolingDown) return button;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0}>{button}</span>
      </TooltipTrigger>
      <TooltipContent className="max-w-60 text-xs">{s.cooldownNote}</TooltipContent>
    </Tooltip>
  );
}
