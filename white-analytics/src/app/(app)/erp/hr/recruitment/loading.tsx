import { Skeleton } from "@/components/ui/skeleton";
import { AnalyticsSkeleton } from "@/features/erp/recruitment/components/analytics/skeleton";

export default function Loading() {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-80 max-w-full" />
      </div>
      <div className="flex flex-wrap gap-2">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-8 w-36 rounded-full" />
        ))}
      </div>
      <Skeleton className="h-9 w-full max-w-xl rounded-full" />
      <AnalyticsSkeleton />
    </>
  );
}
