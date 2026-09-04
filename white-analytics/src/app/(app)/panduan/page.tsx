import type { Metadata } from "next";
import Link from "next/link";
import { BookOpen, Rocket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { PageHeader } from "@/components/dashboard/page-header";
import { GuideBody } from "@/features/guide/components/guide-blocks";
import { GUIDE_SECTIONS } from "@/features/guide/content";
import { requireUser } from "@/lib/rbac";
import { tg } from "@/features/guide/strings";

export const metadata: Metadata = { title: tg.title };

/** Full in-app guide. The topbar help button opens the same content, one section at a time. */
export default async function GuidePage() {
  await requireUser();

  return (
    <>
      <PageHeader
        eyebrow={tg.eyebrow}
        title={tg.title}
        description={tg.subtitle}
        actions={
          <Button asChild size="sm" className="group/nudge">
            <Link href="/clients/new">
              <Rocket className="size-4" /> {tg.startCta}
            </Link>
          </Button>
        }
      />

      <div className="grid gap-8 lg:grid-cols-[220px_1fr]">
        {/* ── Table of contents ── */}
        <nav aria-label={tg.toc} className="hidden lg:block">
          <p className="label-mono mb-3 text-brand">{tg.toc}</p>
          <ol className="space-y-0.5">
            {GUIDE_SECTIONS.map((s, i) => (
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
        <div className="min-w-0 space-y-5">
          {GUIDE_SECTIONS.map((s, i) => (
            <Card key={s.id} id={s.id} className="scroll-mt-20 gap-0 overflow-hidden py-0">
              <header className="border-b px-6 py-5">
                <p className="label-mono text-brand">{String(i + 1).padStart(2, "0")}</p>
                <h2 className="mt-1.5 flex items-center gap-2 text-lg font-bold tracking-[-0.025em]">
                  <BookOpen className="size-4 text-muted-foreground" />
                  {s.title}
                </h2>
                <p className="mt-1 text-sm text-muted-foreground">{s.summary}</p>
              </header>
              <div className="px-6 py-6">
                <GuideBody section={s} />
              </div>
            </Card>
          ))}
        </div>
      </div>
    </>
  );
}
