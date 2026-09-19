import type { ReactNode } from "react";
import Link from "next/link";
import { InfoHint } from "@/components/dashboard/info-hint";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { MarketingFunnel3D } from "@/components/brand/marketing-funnel-3d";
import type { Delta } from "@/lib/metrics";

export type HeroStat = {
  label: string;
  /** already formatted */
  value: string;
  delta?: Delta | null;
  lowerIsBetter?: boolean;
  caption?: ReactNode;
  hint?: string;
  href?: string;
};

/**
 * Page hero: greeting headline, the 3D funnel on a tinted stage, and up to four
 * headline numbers as glass cards flanking it. Replaces PageHeader + KpiGrid on
 * landing pages (portfolio, client overview). On phones the funnel sits on top
 * and the cards fall into a 2-column grid.
 */
export function HeroStage({
  eyebrow,
  title,
  description,
  actions,
  stats,
}: {
  eyebrow?: ReactNode;
  /** wrap the quiet half in <span className="text-muted-foreground"> for the two-tone headline */
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  stats: HeroStat[];
}) {
  const left = stats.slice(0, 2);
  const right = stats.slice(2, 4);
  return (
    <section className="stage-tint relative overflow-hidden rounded-[1.75rem] p-5 md:p-5">
      <div className="relative z-10 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div className="min-w-0 max-w-xl">
          {eyebrow ? <p className="label-mono mb-1.5 text-muted-foreground">{eyebrow}</p> : null}
          <h1 className="text-[28px] leading-[1.12] font-medium md:text-[34px]">{title}</h1>
          {description ? <p className="mt-2 text-sm text-muted-foreground">{description}</p> : null}
        </div>
        {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div> : null}
      </div>

      <div className="mt-4 grid items-center gap-3 md:mt-1 lg:grid-cols-[minmax(0,15rem)_1fr_minmax(0,15rem)]">
        <div className="order-2 grid grid-cols-2 gap-3 lg:order-1 lg:grid-cols-1">
          {left.map((s) => (
            <StatGlass key={s.label} stat={s} />
          ))}
        </div>
        <div className="order-1 h-32 sm:h-40 md:h-44 lg:order-2">
          <MarketingFunnel3D />
        </div>
        <div className="order-3 grid grid-cols-2 gap-3 lg:grid-cols-1">
          {right.map((s) => (
            <StatGlass key={s.label} stat={s} />
          ))}
        </div>
      </div>
    </section>
  );
}

function StatGlass({ stat }: { stat: HeroStat }) {
  const content = (
    <div className="surface-glass lift min-w-0 rounded-[1.25rem] p-3 md:p-3.5">
      <div className="flex items-start justify-between gap-2 text-muted-foreground">
        <span className="label-mono line-clamp-2">{stat.label}</span>
        {stat.hint ? <InfoHint className="ml-auto">{stat.hint}</InfoHint> : null}
      </div>
      <p className="mt-1.5 whitespace-nowrap text-[23px] leading-none font-semibold tracking-[-0.03em] md:text-[26px]">
        {stat.value}
      </p>
      {stat.delta || stat.caption ? (
        <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
          <DeltaBadge delta={stat.delta} lowerIsBetter={stat.lowerIsBetter} />
          {stat.caption}
        </div>
      ) : null}
    </div>
  );
  return stat.href ? (
    <Link
      href={stat.href}
      className="block rounded-[1.25rem] outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      aria-label={`${stat.label}: ${stat.value}. Buka rincian`}
    >
      {content}
    </Link>
  ) : (
    content
  );
}
