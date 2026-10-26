"use client";

import { Printer } from "lucide-react";
import { Button } from "@/components/ui/button";

/**
 * Print button plus print rules that hide the app shell, so any page that
 * renders it prints only its content. Mark extra chrome with `data-print-hide`.
 */
export function PrintButton({ label = "Cetak" }: { label?: string }) {
  return (
    <>
      <style>{`@media print {
  [data-slot="sidebar"], [data-slot="sidebar-gap"], [data-slot="sidebar-container"], header.sticky, nav[aria-label], [data-print-hide] { display: none !important; }
  [data-slot="sidebar-inset"] { margin: 0 !important; }
  body { background: white !important; }
  .print-break { break-before: page; }
}`}</style>
      <Button type="button" variant="outline" size="sm" onClick={() => window.print()} data-print-hide>
        <Printer className="size-3.5" /> {label}
      </Button>
    </>
  );
}
