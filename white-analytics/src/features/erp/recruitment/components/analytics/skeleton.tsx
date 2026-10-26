import { Skeleton } from "@/components/ui/skeleton";

/** Placeholder for the dashboard body: KPI row, two chart rows and a table. */
export function AnalyticsSkeleton() {
  return (
    <div className="flex flex-col gap-4 sm:gap-5" aria-busy="true">
      <Skeleton className="h-4 w-72" />
      <div className="grid grid-cols-2 gap-3 sm:gap-5 lg:grid-cols-4">
        {Array.from({ length: 8 }, (_, i) => (
          <Skeleton key={i} className="h-28 rounded-2xl" />
        ))}
      </div>
      <div className="grid gap-4 sm:gap-5 xl:grid-cols-3">
        <Skeleton className="h-80 rounded-2xl xl:col-span-2" />
        <Skeleton className="h-80 rounded-2xl" />
      </div>
      <div className="grid gap-4 sm:gap-5 xl:grid-cols-3">
        <Skeleton className="h-72 rounded-2xl xl:col-span-2" />
        <Skeleton className="h-72 rounded-2xl" />
      </div>
      <Skeleton className="h-64 rounded-2xl" />
    </div>
  );
}
