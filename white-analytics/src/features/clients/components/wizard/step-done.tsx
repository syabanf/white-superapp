import Link from "next/link";
import { ArrowUpRight, Check, FileText, Megaphone, Search, Share2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { tc } from "@/features/clients/strings";
import { t } from "@/i18n/id";

/** Step 5 — confirmation and the two things you'd actually do next. */
export function StepDone({
  slug,
  clientName,
  modules,
  sourceCount,
  shareEnabled,
}: {
  slug: string;
  clientName: string;
  modules: string[];
  sourceCount: number;
  shareEnabled: boolean;
}) {
  const icons: Record<string, typeof Share2> = { SOCIAL: Share2, SEO: Search, ADS: Megaphone };
  const labels: Record<string, string> = {
    SOCIAL: tc.wizard.socialLabel,
    SEO: tc.wizard.seoLabel,
    ADS: tc.wizard.adsLabel,
  };

  return (
    <div className="space-y-7">
      <div className="flex flex-col items-center gap-3 py-4 text-center">
        <span className="flex size-12 items-center justify-center rounded-full bg-brand/10 text-brand ring-1 ring-brand/25">
          <Check className="size-6" strokeWidth={2.5} />
        </span>
        <div>
          <h3 className="text-lg font-bold tracking-[-0.02em]">{tc.wizard.doneTitle}</h3>
          <p className="mt-1 text-sm text-muted-foreground">{tc.wizard.doneSubtitle}</p>
        </div>
      </div>

      <dl className="overflow-hidden rounded-xl border bg-card">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
          <dt className="label-mono text-muted-foreground">{tc.wizard.summaryChannels}</dt>
          <dd className="flex flex-wrap items-center gap-1.5">
            {modules.length === 0 ? (
              <span className="text-sm text-muted-foreground">–</span>
            ) : (
              modules.map((m) => {
                const Icon = icons[m] ?? Share2;
                return (
                  <span
                    key={m}
                    className="inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium"
                  >
                    <Icon className="size-3" /> {labels[m] ?? m}
                  </span>
                );
              })
            )}
          </dd>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
          <dt className="label-mono text-muted-foreground">{tc.wizard.summarySources}</dt>
          <dd className="text-sm font-medium">
            {sourceCount > 0 ? tc.wizard.sourcesCount(sourceCount) : tc.wizard.noSourceYet}
          </dd>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
          <dt className="label-mono text-muted-foreground">{tc.wizard.summaryShare}</dt>
          <dd>
            <StatusBadge kind={shareEnabled ? "good" : "neutral"}>
              {shareEnabled ? tc.wizard.shareOn : tc.wizard.shareOff}
            </StatusBadge>
          </dd>
        </div>
      </dl>

      <div className="flex flex-col gap-2 border-t pt-5 sm:flex-row sm:justify-end">
        <Button asChild variant="outline" className="group/nudge">
          <Link href={`/clients/${slug}/reports`}>
            <FileText className="size-4" /> {tc.wizard.doneReport}
          </Link>
        </Button>
        <Button asChild className="group/nudge">
          <Link href={`/clients/${slug}`}>
            {tc.wizard.doneOpen} <ArrowUpRight className="nudge size-4" />
          </Link>
        </Button>
      </div>
      <p className="text-center text-xs text-muted-foreground">
        {clientName} · {t.common.settings}:{" "}
        <Link href={`/clients/${slug}/settings`} className="underline underline-offset-2 hover:text-foreground">
          {t.nav.settings}
        </Link>
      </p>
    </div>
  );
}
