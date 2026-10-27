import { Skeleton } from "@/components/ui/skeleton";

/** Mirrors a dashboard tab: KPI row, two chart rows, a table. */
export function TabSkeleton() {
  return (
    <div className="page-stack" aria-busy="true">
      <div className="grid grid-cols-2 gap-(--gap) lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <Skeleton key={i} className="h-32 rounded-[1.5rem]" />
        ))}
      </div>
      <div className="grid gap-(--gap) lg:grid-cols-3">
        <Skeleton className="h-80 rounded-[1.5rem] lg:col-span-2" />
        <Skeleton className="h-80 rounded-[1.5rem]" />
      </div>
      <div className="grid gap-(--gap) lg:grid-cols-2">
        <Skeleton className="h-72 rounded-[1.5rem]" />
        <Skeleton className="h-72 rounded-[1.5rem]" />
      </div>
      <Skeleton className="h-64 rounded-[1.5rem]" />
    </div>
  );
}
