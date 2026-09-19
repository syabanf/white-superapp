"use client";

import { useEffect } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { t } from "@/i18n/id";

export type ErrorBoundaryProps = { error: Error & { digest?: string }; reset: () => void };

/** Shared body for every route-level error boundary. `fullPage` centres it in the viewport for routes without the app shell. */
export function ErrorPanel({ error, reset, fullPage = false }: ErrorBoundaryProps & { fullPage?: boolean }) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <div className={`flex flex-1 flex-col items-center justify-center gap-4 px-6 py-20 text-center ${fullPage ? "min-h-dvh bg-background" : ""}`}>
      <div className="flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
        <AlertTriangle className="size-6" />
      </div>
      <div>
        <h2 className="text-lg font-semibold">{t.common.error}</h2>
        <p className="mt-1 max-w-md text-sm text-muted-foreground">
          {t.errors.pageLoad}
          {error.digest ? ` (kode ${error.digest})` : ""}
        </p>
      </div>
      <Button onClick={reset} variant="outline">
        {t.common.tryAgain}
      </Button>
    </div>
  );
}
