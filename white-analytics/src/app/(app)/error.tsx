"use client";

import { useEffect } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { t } from "@/i18n/id";

export default function AppError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-4 py-20 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
        <AlertTriangle className="size-6" />
      </div>
      <div>
        <h2 className="text-lg font-semibold">{t.common.error}</h2>
        <p className="mt-1 max-w-md text-sm text-muted-foreground">
          Terjadi kesalahan saat memuat halaman ini{error.digest ? ` (kode ${error.digest})` : ""}.
        </p>
      </div>
      <Button onClick={reset} variant="outline">
        {t.common.tryAgain}
      </Button>
    </div>
  );
}
