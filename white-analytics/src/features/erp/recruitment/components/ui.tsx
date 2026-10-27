"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Check, Copy, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { CANDIDATE_STATUS, RECRUITMENT_NAV, kinds, rs } from "../strings";

/** Link bar across the recruitment pages. */
export function RecruitmentNav() {
  const pathname = usePathname();
  return (
    <nav aria-label={rs.dashboard.title} className="overflow-x-auto scrollbar-thin">
      <ul className="flex w-max gap-1 rounded-full bg-muted p-1">
        {RECRUITMENT_NAV.map((item) => {
          const active = "exact" in item ? pathname === item.href : pathname.startsWith(item.href);
          return (
            <li key={item.href}>
              <Link
                href={item.href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "block rounded-full px-3 py-1.5 text-[13px] font-medium whitespace-nowrap text-muted-foreground transition-colors hover:text-foreground",
                  active && "bg-background text-foreground shadow-sm",
                )}
              >
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

export function CandidateBadge({ status }: { status: string }) {
  return <ErpStatusBadge status={status} labels={CANDIDATE_STATUS} kinds={kinds.candidate} />;
}

/** Copies an absolute URL for a path on this site (portal links). */
function CopyLinkButton({ path }: { path: string }) {
  const [done, setDone] = React.useState(false);
  return (
    <Button
      type="button"
      variant="outline"
      size="xs"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(`${window.location.origin}${path}`);
          setDone(true);
          toast.success(rs.copied);
          setTimeout(() => setDone(false), 2000);
        } catch {
          toast.error(`${window.location.origin}${path}`);
        }
      }}
    >
      {done ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
      {rs.copy}
    </Button>
  );
}

/** Portal link row for writers: open and copy. */
export function PortalLink({ path }: { path: string }) {
  return (
    <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/40 px-3 py-2 text-xs">
      <code className="min-w-0 flex-1 truncate text-muted-foreground">{path}</code>
      <Button variant="ghost" size="xs" asChild>
        <a href={path} target="_blank" rel="noreferrer">
          <ExternalLink className="size-3.5" /> {rs.open}
        </a>
      </Button>
      <CopyLinkButton path={path} />
    </div>
  );
}

/** Label and value pair. */
export function Info({ label, children, className }: { label: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={cn("min-w-0", className)}>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 text-sm break-words">{children || rs.none}</dd>
    </div>
  );
}

/** Small list of strings as chips. */
export function Chips({ items }: { items: string[] }) {
  if (!items.length) return <span className="text-sm text-muted-foreground">{rs.none}</span>;
  return (
    <div className="flex flex-wrap gap-1.5">
      {items.map((s) => (
        <span key={s} className="rounded-full bg-muted px-2.5 py-0.5 text-xs">
          {s}
        </span>
      ))}
    </div>
  );
}
