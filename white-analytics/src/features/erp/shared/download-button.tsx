"use client";

import * as React from "react";
import { Download, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "@/lib/toast";
import { ss } from "./strings";

function fileNameFrom(disposition: string | null, fallback: string): string {
  const m = disposition?.match(/filename\*?=(?:UTF-8'')?"?([^";]+)"?/i);
  return m ? decodeURIComponent(m[1]) : fallback;
}

/**
 * Downloads a byte route through the proxy (`/api/erp/v1/...pdf`, XLSX, CSV)
 * as a blob so gateway errors show as a toast instead of a JSON page.
 */
export function DownloadButton({
  href,
  fileName = "download",
  children = ss.download,
  variant = "outline",
  size = "sm",
  className,
}: {
  href: string;
  fileName?: string;
  children?: React.ReactNode;
  variant?: React.ComponentProps<typeof Button>["variant"];
  size?: React.ComponentProps<typeof Button>["size"];
  className?: string;
}) {
  const [busy, setBusy] = React.useState(false);
  const download = async () => {
    setBusy(true);
    try {
      const res = await fetch(href);
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error ?? ss.downloadFailed);
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = fileNameFrom(res.headers.get("content-disposition"), fileName);
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : ss.downloadFailed);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Button type="button" variant={variant} size={size} onClick={download} disabled={busy} className={className}>
      {busy ? <Loader2 className="size-3.5 animate-spin" /> : <Download className="size-3.5" />}
      {children}
    </Button>
  );
}
