import type { Metadata } from "next";
import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card } from "@/components/ui/card";
import { SHOWCASE_TOPICS } from "@/features/showcase/topics";
import { requireUser } from "@/lib/rbac";

export const metadata: Metadata = { title: "Showcase AI" };

export default async function ShowcasePage() {
  await requireUser();
  return (
    <>
      <PageHeader eyebrow="Showcase" title="Showcase AI" description="Simulasi jawaban atas pertanyaan klien, memakai data contoh dan tanpa koneksi WhatsApp." />
      <div className="grid gap-(--gap) md:grid-cols-2 xl:grid-cols-3">
        {SHOWCASE_TOPICS.map(({ id, title, summary, icon: Icon, questions }) => (
          <Link key={id} href={`/showcase/${id}`} className="group/nudge rounded-[1.5rem] outline-none focus-visible:ring-2 focus-visible:ring-ring">
            <Card className="h-full gap-3 p-(--card-pad) transition-colors hover:bg-muted/40">
              <div className="flex items-start justify-between gap-3">
                <span className="flex size-9 items-center justify-center rounded-xl bg-brand/10 text-brand">
                  <Icon className="size-[18px]" />
                </span>
                <ArrowUpRight className="nudge size-4 text-muted-foreground" />
              </div>
              <h2 className="font-semibold">{title}</h2>
              <p className="text-sm text-muted-foreground">{summary}</p>
              <ul className="mt-auto flex flex-col gap-1 border-t pt-3 text-xs text-muted-foreground">
                {questions.map((q) => (
                  <li key={q}>{q}</li>
                ))}
              </ul>
            </Card>
          </Link>
        ))}
      </div>
    </>
  );
}
