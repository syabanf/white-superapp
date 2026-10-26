"use client";

import * as React from "react";
import Link from "next/link";
import { AlertTriangle, ArrowRight, ChevronRight, ExternalLink, Inbox } from "lucide-react";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useIsMobile } from "@/hooks/use-mobile";
import { cn } from "@/lib/utils";
import { useDrill } from "./use-drill";
import type { DrillTarget } from "./href";
import { ds } from "./strings";

export type DrillItem = {
  id: string;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  /** right-aligned figure, e.g. an amount */
  value?: React.ReactNode;
  /** small text under the value, e.g. a date or a DeltaText */
  meta?: React.ReactNode;
  badge?: React.ReactNode;
  /** detail page of the record; absolute http(s) URLs open in a new tab */
  href?: string;
  /** force a new tab (e.g. a same-origin file download) */
  external?: boolean;
};

export type DrillStat = { label: string; value: React.ReactNode };

export type DrillResult = {
  items: DrillItem[];
  /** headline figures above the list (count, total, average...) */
  summary?: DrillStat[];
  /** total matching records when `items` is a capped preview */
  total?: number;
};

export type DrillAction = { label: string; href: string; icon?: React.ReactNode };

type DrillState =
  { status: "loading" } | { status: "error"; message: string } | { status: "ready"; result: DrillResult };

export type DrillSheetViewProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  /** period and active filters, e.g. ["Maret 2026", "Creative"] */
  chips?: React.ReactNode[];
  /** secondary routes, e.g. a time drill ("Lihat per hari") or "Saring dasbor" */
  actions?: DrillAction[];
  state: DrillState;
  onRetry?: () => void;
  fullListHref?: string;
  fullListLabel?: string;
  emptyLabel?: string;
};

/**
 * Presentational drill panel: right side on desktop, bottom sheet on phones.
 * Header (title, chips, actions), summary stats, the record list, and a
 * footer link to the full list. Use `DrillSheet` unless you manage state yourself.
 */
export function DrillSheetView({
  open,
  onOpenChange,
  title,
  description,
  chips,
  actions,
  state,
  onRetry,
  fullListHref,
  fullListLabel = ds.fullList,
  emptyLabel = ds.empty,
}: DrillSheetViewProps) {
  const mobile = useIsMobile();
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side={mobile ? "bottom" : "right"}
        className={cn(
          "gap-0 overflow-hidden p-0",
          mobile ? "max-h-[85dvh] rounded-t-[1.5rem]" : "data-[side=right]:w-[calc(100vw-1rem)] data-[side=right]:sm:max-w-md",
        )}
      >
        {mobile ? (
          <span aria-hidden className="mx-auto mt-2 h-1 w-10 shrink-0 rounded-full bg-muted-foreground/30" />
        ) : null}
        <SheetHeader className="shrink-0 gap-2 border-b pr-12">
          <SheetTitle className="text-[15px] leading-snug font-semibold">{title}</SheetTitle>
          {description ? (
            <SheetDescription className="text-xs">{description}</SheetDescription>
          ) : (
            <SheetDescription className="sr-only">{title}</SheetDescription>
          )}
          {chips && chips.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {chips.map((chip, i) => (
                <span
                  key={i}
                  className="max-w-full truncate rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground"
                >
                  {chip}
                </span>
              ))}
            </div>
          ) : null}
          {actions && actions.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {actions.map((a) => (
                <Button key={a.href} asChild variant="outline" size="xs">
                  <Link href={a.href} scroll={false}>
                    {a.icon}
                    {a.label}
                  </Link>
                </Button>
              ))}
            </div>
          ) : null}
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
          <DrillBody state={state} onRetry={onRetry} emptyLabel={emptyLabel} />
        </div>

        {fullListHref ? (
          <div className="shrink-0 border-t p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]">
            <Button asChild className="w-full">
              <Link href={fullListHref}>
                {fullListLabel}
                <ArrowRight aria-hidden />
              </Link>
            </Button>
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

function DrillBody({
  state,
  onRetry,
  emptyLabel,
}: {
  state: DrillState;
  onRetry?: () => void;
  emptyLabel: string;
}) {
  if (state.status === "loading") {
    return (
      <div className="flex flex-col gap-3 p-4" aria-busy="true" aria-label={ds.loading}>
        <div className="grid grid-cols-2 gap-2">
          <Skeleton className="h-14 rounded-xl" />
          <Skeleton className="h-14 rounded-xl" />
        </div>
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-11 rounded-lg" />
        ))}
      </div>
    );
  }
  if (state.status === "error") {
    return (
      <div className="flex flex-col items-center gap-2 px-6 py-10 text-center">
        <AlertTriangle className="size-5 text-negative" aria-hidden />
        <p className="text-sm font-medium">{ds.error}</p>
        <p className="text-xs text-muted-foreground">{state.message}</p>
        {onRetry ? (
          <Button variant="outline" size="xs" onClick={onRetry} className="mt-1">
            {ds.retry}
          </Button>
        ) : null}
      </div>
    );
  }
  const { items, summary, total } = state.result;
  return (
    <div className="flex flex-col gap-3 p-4">
      {summary && summary.length > 0 ? (
        <dl className="grid grid-cols-2 gap-2">
          {summary.map((s) => (
            <div key={s.label} className="min-w-0 rounded-xl bg-muted/50 px-3 py-2">
              <dt className="label-mono truncate text-muted-foreground">{s.label}</dt>
              <dd className="mt-0.5 truncate text-[15px] font-bold tracking-[-0.02em] tabular">{s.value}</dd>
            </div>
          ))}
        </dl>
      ) : null}
      {items.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-10 text-center text-sm text-muted-foreground">
          <Inbox className="size-5" aria-hidden />
          {emptyLabel}
        </div>
      ) : (
        <>
          {total != null && total > items.length ? (
            <p className="text-xs text-muted-foreground">{ds.showing(items.length, total)}</p>
          ) : null}
          <ul className="-mx-2 flex flex-col">
            {items.map((item) => (
              <li key={item.id}>
                <DrillRow item={item} />
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}

function DrillRow({ item }: { item: DrillItem }) {
  const body = (
    <>
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="truncate text-sm font-medium">{item.title}</span>
          {item.badge}
        </div>
        {item.subtitle ? <div className="truncate text-xs text-muted-foreground">{item.subtitle}</div> : null}
      </div>
      {item.value != null || item.meta ? (
        <div className="flex shrink-0 flex-col items-end text-right">
          {item.value != null ? <span className="text-sm font-semibold tabular">{item.value}</span> : null}
          {item.meta ? <span className="text-xs text-muted-foreground">{item.meta}</span> : null}
        </div>
      ) : null}
    </>
  );
  const row = "flex min-h-11 items-center gap-3 rounded-lg px-2 py-2";
  if (!item.href) return <div className={row}>{body}</div>;
  if (item.external || /^https?:\/\//i.test(item.href)) {
    return (
      <a
        href={item.href}
        target="_blank"
        rel="noopener noreferrer"
        className={cn(row, "outline-none hover:bg-muted/60 focus-visible:ring-2 focus-visible:ring-ring")}
      >
        {body}
        <ExternalLink className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </a>
    );
  }
  return (
    <Link
      href={item.href}
      className={cn(row, "outline-none hover:bg-muted/60 focus-visible:ring-2 focus-visible:ring-ring")}
    >
      {body}
      <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    </Link>
  );
}

export type DrillSheetProps = {
  /** drill kind(s) this sheet answers to (`?drill=<kind>:<key>`) */
  kind: string | readonly string[];
  title: (target: DrillTarget) => React.ReactNode;
  description?: (target: DrillTarget) => React.ReactNode;
  chips?: (target: DrillTarget) => React.ReactNode[];
  actions?: (target: DrillTarget) => DrillAction[];
  /** records behind the mark; may be sync (data already on the client) or async */
  load: (target: DrillTarget, signal: AbortSignal) => DrillResult | Promise<DrillResult>;
  fullListHref?: (target: DrillTarget) => string | undefined;
  fullListLabel?: string;
  emptyLabel?: string;
};

/**
 * URL-driven drill sheet. Opens when `?drill=` matches `kind`, runs `load`
 * for the key, and closes through `useDrill().close()` so Back works.
 * Mount once per dashboard (or per kind); trigger it with `useDrill().open(kind, key)`.
 */
export function DrillSheet({
  kind,
  title,
  description,
  chips,
  actions,
  load,
  fullListHref,
  fullListLabel,
  emptyLabel,
}: DrillSheetProps) {
  const { drill, close } = useDrill();
  const kinds = typeof kind === "string" ? [kind] : kind;
  const target = drill && kinds.includes(drill.kind) ? drill : null;
  // the last target stays mounted while the close animation runs; state keeps its identity stable
  const [shown, setShown] = React.useState<DrillTarget | null>(target);
  if (target && (target.kind !== shown?.kind || target.key !== shown?.key)) setShown(target);

  return (
    <DrillSheetLoader
      key={shown ? `${shown.kind}:${shown.key}` : "none"}
      target={shown}
      open={target != null}
      onClose={close}
      title={title}
      description={description}
      chips={chips}
      actions={actions}
      load={load}
      fullListHref={fullListHref}
      fullListLabel={fullListLabel}
      emptyLabel={emptyLabel}
    />
  );
}

function DrillSheetLoader({
  target,
  open,
  onClose,
  title,
  description,
  chips,
  actions,
  load,
  fullListHref,
  fullListLabel,
  emptyLabel,
}: Omit<DrillSheetProps, "kind"> & { target: DrillTarget | null; open: boolean; onClose: () => void }) {
  const [state, setState] = React.useState<DrillState>({ status: "loading" });
  const [attempt, setAttempt] = React.useState(0);
  const loadRef = React.useRef(load);
  React.useEffect(() => {
    loadRef.current = load;
  });

  React.useEffect(() => {
    if (!target) return;
    const ctrl = new AbortController();
    Promise.resolve()
      .then(() => loadRef.current(target, ctrl.signal))
      .then(
        (result) => !ctrl.signal.aborted && setState({ status: "ready", result }),
        (err: unknown) => {
          if (ctrl.signal.aborted) return;
          setState({
            status: "error",
            message: err instanceof Error && err.message ? err.message : ds.error,
          });
        },
      );
    return () => ctrl.abort();
  }, [target, attempt]);

  if (!target) return null;
  return (
    <DrillSheetView
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={title(target)}
      description={description?.(target)}
      chips={chips?.(target)}
      actions={actions?.(target)}
      state={state}
      onRetry={() => {
        setState({ status: "loading" });
        setAttempt((n) => n + 1);
      }}
      fullListHref={fullListHref?.(target)}
      fullListLabel={fullListLabel}
      emptyLabel={emptyLabel}
    />
  );
}
