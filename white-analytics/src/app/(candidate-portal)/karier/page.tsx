import type { Metadata } from "next";
import Link from "next/link";
import { Briefcase, ChevronRight, MapPin, Search } from "lucide-react";
import { EmptyState } from "@/components/dashboard/empty-state";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { sp1 } from "@/features/erp/shared/load";
import { loadPublic } from "@/features/erp/recruitment/public/load";
import { PortalProblem } from "@/features/erp/recruitment/public/states";
import { cp } from "@/features/erp/recruitment/strings";
import type { PublicJob } from "@/features/erp/recruitment/types";
import { formatDate } from "@/lib/format";

export const metadata: Metadata = {
  title: cp.career.eyebrow,
  description: cp.career.intro,
  robots: { index: true, follow: true },
};

type Jobs = { jobs: PublicJob[] };

export default async function CareerPage(props: PageProps<"/karier">) {
  const sp = await props.searchParams;
  const search = sp1(sp.q)?.trim() ?? "";
  const department = sp1(sp.department) ?? "";
  const query = new URLSearchParams({ ...(search ? { search } : {}), ...(department ? { department } : {}) }).toString();

  const [res, all] = await Promise.all([
    loadPublic<Jobs>(`/v1/recruitment/public/jobs${query ? `?${query}` : ""}`),
    query ? loadPublic<Jobs>("/v1/recruitment/public/jobs") : null,
  ]);
  if (!res.data) return <PortalProblem kind={res.error} message={res.message} />;
  const departments = [...new Set((all?.data ?? res.data).jobs.map((j) => j.department).filter(Boolean))].sort();
  const c = cp.career;

  return (
    <>
      <div>
        <p className="label-mono text-muted-foreground">{c.eyebrow}</p>
        <h1 className="mt-1 text-[28px] leading-tight font-semibold md:text-[34px]">{c.title}</h1>
        <p className="mt-1.5 text-sm text-muted-foreground">{c.intro}</p>
      </div>
      <form method="get" className="flex flex-col gap-2 sm:flex-row" role="search">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input name="q" defaultValue={search} placeholder={c.search} aria-label={c.search} className="h-10 pl-9" />
        </div>
        {departments.length > 1 ? (
          <select
            name="department"
            defaultValue={department}
            aria-label={c.department}
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
          >
            <option value="">{c.department}</option>
            {departments.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>
        ) : null}
        <Button type="submit" className="h-10">
          {c.filter}
        </Button>
      </form>
      {res.data.jobs.length === 0 ? (
        <EmptyState icon={<Briefcase className="size-5" />} title={c.empty} description={c.emptyDesc} />
      ) : (
        <div className="flex flex-col gap-3">
          <p className="text-xs text-muted-foreground">{c.openings(res.data.jobs.length)}</p>
          {res.data.jobs.map((job) => (
            <Link key={job.slug} href={`/karier/${job.slug}`} className="group block">
              <Card className="flex flex-row items-center gap-4 p-4 transition-colors group-hover:bg-accent/40 sm:p-5">
                <div className="min-w-0 flex-1">
                  <h2 className="text-base font-semibold">{job.title}</h2>
                  <p className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    {job.department ? <span>{job.department}</span> : null}
                    {job.location ? (
                      <span className="flex items-center gap-1">
                        <MapPin className="size-3.5" /> {job.location}
                      </span>
                    ) : null}
                    <span>
                      {job.employmentType} · {job.workMode}
                    </span>
                    {job.closingDate ? <span>{c.closing(formatDate(job.closingDate))}</span> : null}
                  </p>
                </div>
                <ChevronRight className="size-5 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
              </Card>
            </Link>
          ))}
        </div>
      )}
    </>
  );
}
