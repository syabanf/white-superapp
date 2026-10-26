import { Skeleton } from "@/components/ui/skeleton";
import { TabSkeleton } from "@/features/erp/performance/dashboard/skeleton";

export default function PerformanceLoading() {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-4 w-80 max-w-full" />
      </div>
      <div className="flex flex-wrap gap-2">
        <Skeleton className="h-8 w-40 rounded-full" />
        <Skeleton className="h-8 w-44 rounded-full" />
        <Skeleton className="h-8 w-44 rounded-full" />
        <Skeleton className="h-8 w-56 rounded-full" />
      </div>
      <Skeleton className="h-9 w-full max-w-xl rounded-full" />
      <TabSkeleton />
    </>
  );
}
