import { Skeleton } from "@/components/ui/skeleton";
import { TabSkeleton } from "@/features/erp/finance/dashboard/skeleton";

export default function FinanceLoading() {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-96 max-w-full" />
      </div>
      <div className="flex flex-wrap gap-(--gap-inner)">
        <Skeleton className="h-(--control-h) w-44 rounded-full" />
        <Skeleton className="h-(--control-h) w-44 rounded-full" />
      </div>
      <Skeleton className="h-(--control-h) w-full max-w-xl rounded-full" />
      <TabSkeleton />
    </>
  );
}
