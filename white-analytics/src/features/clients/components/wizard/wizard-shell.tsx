import type { ReactNode } from "react";
import Link from "next/link";
import { Check } from "lucide-react";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { tc } from "@/features/clients/strings";

export type WizardShellStep = { key: string; index: string; title: string; description: string };

/**
 * Wizard chrome. Desktop: a numbered rail on the left (clickable for steps
 * already passed) and the active step's panel on the right. Mobile: a compact
 * progress header (step n/N, segmented bar, next step) above a full-bleed
 * panel — the rail would eat the whole viewport there.
 */
export function WizardShell({
  steps,
  current,
  hrefFor,
  eyebrow = tc.wizard.eyebrow,
  children,
  footer,
}: {
  steps: readonly WizardShellStep[];
  current: string;
  /** link for a step already passed; return null (or omit) to keep it static */
  hrefFor?: (key: string) => string | null;
  eyebrow?: string;
  children: ReactNode;
  footer?: ReactNode;
}) {
  const currentIdx = steps.findIndex((s) => s.key === current);
  const meta = steps[currentIdx] ?? steps[0]!;
  const upcoming = steps[currentIdx + 1];
  const linkFor = (key: string, done: boolean) => (done && hrefFor ? hrefFor(key) : null);

  return (
    <div className="mx-auto grid w-full max-w-5xl gap-6 lg:grid-cols-[220px_1fr] lg:gap-8">
      {/* ── Rail (desktop) ── */}
      <nav aria-label={tc.wizard.stepOf} className="hidden lg:block">
        <p className="label-mono mb-4 text-brand">{eyebrow}</p>
        <ol className="relative space-y-1">
          {steps.map((m, i) => {
            const done = i < currentIdx;
            const active = i === currentIdx;
            const href = linkFor(m.key, done);
            const reachable = Boolean(href);
            const row = (
              <span
                className={cn(
                  "flex items-start gap-3 rounded-lg px-2 py-2 transition-colors duration-150",
                  active && "bg-accent",
                  reachable && "hover:bg-muted",
                )}
              >
                <span
                  className={cn(
                    "mt-px flex size-5 shrink-0 items-center justify-center rounded-full border text-[10px] font-semibold transition-colors duration-200",
                    done && "border-brand bg-brand text-primary-foreground",
                    active && "border-brand text-brand",
                    !done && !active && "border-border text-muted-foreground",
                  )}
                >
                  {done ? <Check className="size-3" strokeWidth={3} /> : i + 1}
                </span>
                <span className="min-w-0">
                  <span
                    className={cn(
                      "block text-[13px] leading-tight font-medium",
                      active ? "text-brand-ink" : done ? "text-foreground" : "text-muted-foreground",
                    )}
                  >
                    {m.title}
                  </span>
                </span>
              </span>
            );
            return (
              <li key={m.key}>
                {href ? (
                  <Link href={href} className="block">
                    {row}
                  </Link>
                ) : (
                  row
                )}
              </li>
            );
          })}
        </ol>
      </nav>

      {/* ── Panel ── */}
      <div className="min-w-0">
        {/* Mobile progress header */}
        <div className="mb-4 lg:hidden">
          <div className="flex items-baseline justify-between gap-3">
            <p className="label-mono shrink-0 whitespace-nowrap text-brand">
              {tc.wizard.stepOf} {currentIdx + 1}/{steps.length}
            </p>
            {upcoming ? (
              <p className="label-mono truncate text-muted-foreground">
                {tc.wizard.upNext}: {upcoming.title}
              </p>
            ) : null}
          </div>
          <ol className="mt-2 flex gap-1" aria-hidden>
            {steps.map((m, i) => {
              const done = i < currentIdx;
              const active = i === currentIdx;
              const href = linkFor(m.key, done);
              const seg = (
                <span
                  className={cn(
                    "block h-1 w-full rounded-full transition-colors duration-200",
                    done || active ? "bg-brand" : "bg-border",
                    active && "opacity-60",
                  )}
                />
              );
              return (
                <li key={m.key} className="flex-1">
                  {href ? (
                    <Link href={href} className="block py-2" aria-label={m.title}>
                      {seg}
                    </Link>
                  ) : (
                    <span className="block py-2">{seg}</span>
                  )}
                </li>
              );
            })}
          </ol>
        </div>

        <Card className="-mx-4 gap-0 overflow-visible rounded-none border-x-0 py-0 sm:mx-0 sm:overflow-hidden sm:rounded-xl sm:border-x">
          <header className="border-b px-4 py-4 sm:px-6 sm:py-5">
            <p className="label-mono text-brand">{meta.index}</p>
            <h2 className="mt-1.5 text-lg font-bold tracking-[-0.03em] sm:text-xl">{meta.title}</h2>
            <p className="mt-1.5 text-sm text-muted-foreground">{meta.description}</p>
          </header>
          <div className="px-4 py-5 sm:px-6 sm:py-6">{children}</div>
          {footer ? (
            <div className="flex items-center justify-between gap-3 border-t bg-muted/30 px-4 py-4 sm:px-6">{footer}</div>
          ) : null}
        </Card>
      </div>
    </div>
  );
}
