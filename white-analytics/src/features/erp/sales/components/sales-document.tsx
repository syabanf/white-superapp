import type { ReactNode } from "react";
import { MixistLogo } from "@/components/brand/logo";
import { cn } from "@/lib/utils";
import { COMPANY } from "../company";
import { ss } from "../strings";

const s = ss.doc;

/** One `LABEL : value` row of the document's info block. */
export type DocRow = { label: string; value: ReactNode };

/**
 * Paper layout shared by quotations and invoices, in the order a printed copy
 * reads: letterhead, a centred title band with the number, a two-column info
 * block, then the body (items, notes, totals) passed as children.
 */
export function SalesDocument({
  title,
  number,
  left,
  right,
  children,
  className,
}: {
  title: string;
  number: ReactNode;
  left: DocRow[];
  right: DocRow[];
  children: ReactNode;
  className?: string;
}) {
  return (
    <article className={cn("overflow-hidden rounded-[1.5rem] border bg-card text-card-foreground shadow-(--card-shadow)", className)}>
      <header className="flex flex-col gap-5 px-(--card-pad) py-6 sm:flex-row sm:items-start sm:justify-between sm:px-8">
        <MixistLogo size={44} />
        <div className="flex max-w-md flex-col gap-1 sm:items-end sm:text-right">
          <p className="text-lg font-bold tracking-[-0.02em] sm:text-xl">{COMPANY.legalName}</p>
          <p className="text-xs leading-relaxed text-muted-foreground italic">{COMPANY.address}</p>
          <a href={`mailto:${COMPANY.email}`} className="text-xs text-brand underline-offset-2 hover:underline">
            {COMPANY.email}
          </a>
          {COMPANY.npwp ? (
            <p className="text-xs text-muted-foreground">
              {s.npwp}: {COMPANY.npwp}
            </p>
          ) : null}
        </div>
      </header>

      <div className="border-y bg-muted/50 px-(--card-pad) py-4 text-center">
        <h2 className="text-base font-bold tracking-[0.08em]">{title}</h2>
        <p className="mt-1 text-xs text-muted-foreground">
          {s.number} : <span className="font-medium text-foreground tabular">{number}</span>
        </p>
      </div>

      <div className="grid border-b md:grid-cols-2 md:divide-x">
        <DocRows rows={left} />
        <DocRows rows={right} />
      </div>

      <div className="flex flex-col gap-6 px-(--card-pad) py-6 sm:px-8">{children}</div>
    </article>
  );
}

function DocRows({ rows }: { rows: DocRow[] }) {
  return (
    <dl className="flex flex-col gap-2.5 px-(--card-pad) py-5 sm:px-8">
      {rows.map((r) => (
        <div key={r.label} className="grid min-h-7 grid-cols-[112px_10px_minmax(0,1fr)] items-baseline gap-x-2 sm:grid-cols-[136px_12px_minmax(0,1fr)]">
          <dt className="text-[11px] font-medium tracking-[0.04em] text-muted-foreground uppercase italic">{r.label}</dt>
          <span aria-hidden className="text-muted-foreground">
            :
          </span>
          <dd className="min-w-0 text-sm break-words">{r.value || <span className="text-muted-foreground">-</span>}</dd>
        </div>
      ))}
    </dl>
  );
}

/** Section heading inside the document body. */
export function DocSection({ title, actions, children }: { title: string; actions?: ReactNode; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-xs font-semibold tracking-[0.08em] text-muted-foreground uppercase">{title}</h3>
        {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
      </div>
      {children}
    </section>
  );
}

/** Totals panel on the right of the document body. */
export function DocTotals({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-2.5 rounded-2xl border bg-muted/30 p-(--card-pad) text-sm">{children}</div>;
}
