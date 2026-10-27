import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, Briefcase, CalendarClock, MapPin, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ApplyForm } from "@/features/erp/recruitment/public/apply-form";
import { loadPublic } from "@/features/erp/recruitment/public/load";
import { PortalProblem } from "@/features/erp/recruitment/public/states";
import { cp } from "@/features/erp/recruitment/strings";
import type { PublicJob } from "@/features/erp/recruitment/types";
import { formatDate } from "@/lib/format";

const path = (slug: string) => `/v1/recruitment/public/jobs/${encodeURIComponent(slug)}`;

export async function generateMetadata(props: PageProps<"/karier/[slug]">): Promise<Metadata> {
  const { slug } = await props.params;
  const res = await loadPublic<PublicJob>(path(slug));
  return {
    title: res.data?.title ?? cp.career.eyebrow,
    description: res.data?.description.slice(0, 160),
    robots: { index: Boolean(res.data), follow: true },
  };
}

export default async function JobDetailPage(props: PageProps<"/karier/[slug]">) {
  const { slug } = await props.params;
  const res = await loadPublic<PublicJob>(path(slug));
  if (!res.data) return <PortalProblem kind={res.error} message={res.message} />;
  const job = res.data;
  const c = cp.career;

  return (
    <>
      <Button variant="ghost" size="sm" asChild className="-ml-2 self-start">
        <Link href="/karier">
          <ArrowLeft className="size-4" /> {c.back}
        </Link>
      </Button>
      <div>
        {job.department ? <p className="label-mono text-muted-foreground">{job.department}</p> : null}
        <h1 className="mt-1 text-[28px] leading-tight font-semibold md:text-[34px]">{job.title}</h1>
        <ul className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5 text-sm text-muted-foreground">
          {job.location ? (
            <li className="flex items-center gap-1.5">
              <MapPin className="size-4" /> {job.location}
            </li>
          ) : null}
          <li className="flex items-center gap-1.5">
            <Briefcase className="size-4" /> {job.employmentType} · {job.workMode}
          </li>
          <li className="flex items-center gap-1.5">
            <Users className="size-4" /> {c.headcount(job.headcount)}
          </li>
          {job.closingDate ? (
            <li className="flex items-center gap-1.5">
              <CalendarClock className="size-4" /> {c.closing(formatDate(job.closingDate))}
            </li>
          ) : null}
        </ul>
      </div>
      {(
        [
          [c.description, job.description],
          [c.requirements, job.requirements],
          [c.benefits, job.benefits],
        ] as const
      )
        .filter(([, body]) => body.trim())
        .map(([title, body]) => (
          <section key={title}>
            <h2 className="mb-2 text-base font-semibold">{title}</h2>
            <p className="text-sm leading-relaxed whitespace-pre-wrap text-foreground/90">{body}</p>
          </section>
        ))}
      <Card id="lamar" className="scroll-mt-6 p-(--card-pad)">
        <h2 className="mb-4 text-lg font-semibold">{c.applyTitle}</h2>
        <ApplyForm slug={job.slug} />
      </Card>
    </>
  );
}
