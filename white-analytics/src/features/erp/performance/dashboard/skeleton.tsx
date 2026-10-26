import { Skeleton } from "@/components/ui/skeleton";

/** Placeholder for a dashboard tab: tiles, a wide chart, two charts and a table. */
export function TabSkeleton() {
  return (
    <div className="flex flex-col gap-4 sm:gap-5" aria-busy="true">
      <div className="grid grid-cols-2 gap-3 sm:gap-5 md:grid-cols-3 xl:grid-cols-6">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-32 rounded-2xl" />
        ))}
      </div>
      <Skeleton className="h-80 rounded-2xl" />
      <div className="grid gap-4 sm:gap-5 lg:grid-cols-2">
        <Skeleton className="h-64 rounded-2xl" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
      <Skeleton className="h-72 rounded-2xl" />
    </div>
  );
}
