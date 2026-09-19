import Link from "next/link";
import { AlertTriangle, FlaskConical, PlugZap } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { InfoHint } from "@/components/dashboard/info-hint";
import { t } from "@/i18n/id";
import type { ProviderErrorCode } from "@/lib/action-result";

/** Shown when a module renders seed/mock data: a small chip, the explanation lives in its tooltip. */
export function DemoBanner({ message, settingsHref, className }: { message: string; settingsHref?: string; className?: string }) {
  return (
    <div className={cn("flex h-8 w-fit items-center gap-2 rounded-full bg-card pr-3 pl-2.5 text-xs text-muted-foreground shadow-(--card-shadow)", className)}>
      <FlaskConical className="size-3.5" />
      <span className="font-medium text-foreground">{t.common.demoData}</span>
      <InfoHint>{message}</InfoHint>
      {settingsHref ? (
        <Link href={settingsHref} className="flex items-center gap-1 font-medium text-brand-ink hover:underline">
          <PlugZap className="size-3" /> {t.common.connect}
        </Link>
      ) : null}
    </div>
  );
}

/** Normalized provider error → banner with reconnect CTA. */
export function ProviderErrorBanner({ code, message, reconnectHref, className }: { code: ProviderErrorCode; message?: string; reconnectHref?: string; className?: string }) {
  const showReconnect = code === "TOKEN_EXPIRED" || code === "PERMISSION";
  return (
    <Alert variant="destructive" className={cn("py-2", className)}>
      <AlertTriangle className="size-4" />
      <AlertTitle className="text-xs font-medium">{t.errors[code] ?? t.errors.UNKNOWN}</AlertTitle>
      {message || (showReconnect && reconnectHref) ? (
        <AlertDescription className="flex flex-wrap items-center gap-2 text-xs">
          {message ? <span>{message}</span> : null}
          {showReconnect && reconnectHref ? (
            <Button asChild variant="link" size="sm" className="h-auto p-0 text-xs">
              <Link href={reconnectHref}>{t.common.reconnect}</Link>
            </Button>
          ) : null}
        </AlertDescription>
      ) : null}
    </Alert>
  );
}
