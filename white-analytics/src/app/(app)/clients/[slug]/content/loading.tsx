import { Skeleton } from "@/components/ui/skeleton";

/** Mirrors the overview: hero, filter bar, tabs, KPI grid, two chart rows, table. */
export default function Loading() {
  return (
    <div className="page-stack" aria-busy="true">
      <Skeleton className="h-[19rem] rounded-[1.75rem] md:h-60" />
      <Skeleton className="h-(--control-h) w-full max-w-3xl rounded-full" />
      <Skeleton className="h-10 w-full max-w-xl rounded-full" />
      <div className="grid grid-cols-2 gap-(--gap) lg:grid-cols-4">
        {Array.from({ length: 8 }, (_, i) => (
          <Skeleton key={i} className="h-32 rounded-[1.5rem]" />
        ))}
      </div>
      <div className="grid gap-(--gap) lg:grid-cols-5">
        <Skeleton className="h-80 rounded-[1.5rem] lg:col-span-3" />
        <Skeleton className="h-80 rounded-[1.5rem] lg:col-span-2" />
      </div>
      <Skeleton className="h-64 rounded-[1.5rem]" />
    </div>
  );
}
