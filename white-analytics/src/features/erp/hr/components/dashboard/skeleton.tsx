import { Skeleton } from "@/components/ui/skeleton";

/** Tab body placeholder: KPI tiles, a wide chart beside a narrow one, a table. */
export function DashboardBodySkeleton() {
  return (
    <div className="page-stack" aria-busy="true">
      <div className="grid grid-cols-2 gap-(--gap) lg:grid-cols-4">
        {Array.from({ length: 8 }, (_, i) => (
          <Skeleton key={i} className={i < 4 ? "h-32 rounded-[1.5rem]" : "h-24 rounded-[1.5rem]"} />
        ))}
      </div>
      <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-3">
        <Skeleton className="h-80 rounded-[1.5rem] lg:col-span-2" />
        <Skeleton className="h-80 rounded-[1.5rem]" />
      </div>
      <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-3">
        <Skeleton className="h-72 rounded-[1.5rem] lg:col-span-2" />
        <Skeleton className="h-72 rounded-[1.5rem]" />
      </div>
      <Skeleton className="h-64 rounded-[1.5rem]" />
    </div>
  );
}

/** Whole-page placeholder: header, filter bar, tabs, then the body. */
export function DashboardPageSkeleton() {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-14" />
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-4 w-80 max-w-full" />
      </div>
      <Skeleton className="h-(--control-h) w-full max-w-3xl rounded-full" />
      <Skeleton className="h-(--control-h) w-full max-w-md rounded-full" />
      <DashboardBodySkeleton />
    </>
  );
}
