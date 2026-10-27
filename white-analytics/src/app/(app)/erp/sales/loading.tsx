import { Skeleton } from "@/components/ui/skeleton";
import { DashboardSkeleton } from "@/features/erp/sales/dashboard/skeleton";

export default function SalesLoading() {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-96 max-w-full" />
      </div>
      <div className="flex flex-wrap gap-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-8 w-36 rounded-lg" />
        ))}
      </div>
      <Skeleton className="h-9 w-full max-w-xl rounded-lg" />
      <DashboardSkeleton />
    </>
  );
}
