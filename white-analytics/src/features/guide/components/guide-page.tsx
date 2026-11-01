import type { ReactNode } from "react";
import Link from "next/link";
import { BookOpen } from "lucide-react";
import { Card } from "@/components/ui/card";
import { PageHeader } from "@/components/dashboard/page-header";
import { cn } from "@/lib/utils";
import { GuideBody } from "./guide-blocks";
import { GuideVideo } from "./guide-video";
import { GUIDE_TRACKS, type GuideTrack } from "../content";
import { tg } from "../strings";

/** Full tutorial for one track: track switch, table of contents, then every section. */
export function GuidePage({
  track,
  title,
  description,
  action,
}: {
  track: GuideTrack;
  title: string;
  description: string;
  action: ReactNode;
}) {
  const sections = GUIDE_TRACKS[track].sections;
  return (
    <>
      <PageHeader eyebrow={tg.eyebrow} title={title} description={description} actions={action} />

      <nav aria-label={tg.tracks} className="flex w-fit gap-1 rounded-full bg-card p-1 shadow-(--card-shadow)">
        {(Object.keys(GUIDE_TRACKS) as GuideTrack[]).map((key) => (
          <Link
            key={key}
            href={GUIDE_TRACKS[key].href}
            aria-current={key === track ? "page" : undefined}
            className={cn(
              "rounded-full px-4 py-1.5 text-sm font-medium transition-colors duration-150",
              key === track ? "bg-foreground text-background" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {tg[key]}
          </Link>
        ))}
      </nav>

      <div className="grid gap-(--gap) lg:grid-cols-[220px_minmax(0,1fr)]">
        {/* ── Table of contents ── */}
        <nav aria-label={tg.toc} className="hidden lg:block">
          <p className="label-mono mb-3 px-2 text-brand">{tg.toc}</p>
          <ol className="space-y-0.5">
            {sections.map((s, i) => (
              <li key={s.id}>
                <a
                  href={`#${s.id}`}
                  className="flex items-baseline gap-2.5 rounded-lg px-2 py-1.5 text-[13px] text-muted-foreground transition-colors duration-150 hover:bg-muted hover:text-foreground"
                >
                  <span className="label-mono shrink-0 text-brand">{String(i + 1).padStart(2, "0")}</span>
                  <span className="min-w-0">{s.title}</span>
                </a>
              </li>
            ))}
          </ol>
        </nav>

        {/* ── Sections ── */}
        <div className="page-stack min-w-0">
          {sections.map((s, i) => (
            <Card key={s.id} id={s.id} className="scroll-mt-20 gap-0 overflow-hidden py-0">
              <header className="border-b px-(--card-pad) py-4">
                <p className="label-mono text-brand">{String(i + 1).padStart(2, "0")}</p>
                <h2 className="mt-1.5 flex items-center gap-2 text-lg font-bold tracking-[-0.025em]">
                  <BookOpen className="size-4 text-muted-foreground" />
                  {s.title}
                </h2>
                <p className="mt-1 text-sm text-muted-foreground">{s.summary}</p>
              </header>
              <div className="flex flex-col gap-5 p-(--card-pad)">
                <GuideVideo id={s.id} title={s.title} />
                <GuideBody section={s} />
              </div>
            </Card>
          ))}
        </div>
      </div>
    </>
  );
}
