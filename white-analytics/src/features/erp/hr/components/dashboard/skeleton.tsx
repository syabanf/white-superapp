import { Skeleton } from "@/components/ui/skeleton";

/** Tab body placeholder: KPI tiles, a wide chart beside a narrow one, a table. */
export function DashboardBodySkeleton() {
  return (
    <div className="space-y-5" aria-busy="true">
      <div className="grid grid-cols-2 gap-3 sm:gap-5 lg:grid-cols-4">
        {Array.from({ length: 8 }, (_, i) => (
          <Skeleton key={i} className={i < 4 ? "h-32 rounded-2xl" : "h-24 rounded-2xl"} />
        ))}
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3 lg:gap-5">
        <Skeleton className="h-80 rounded-2xl lg:col-span-2" />
        <Skeleton className="h-80 rounded-2xl" />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3 lg:gap-5">
        <Skeleton className="h-72 rounded-2xl lg:col-span-2" />
        <Skeleton className="h-72 rounded-2xl" />
      </div>
      <Skeleton className="h-64 rounded-2xl" />
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
      <Skeleton className="h-[92px] rounded-[1.25rem]" />
      <Skeleton className="h-9 w-full max-w-md rounded-full" />
      <DashboardBodySkeleton />
    </>
  );
}
