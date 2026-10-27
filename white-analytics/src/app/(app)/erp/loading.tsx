import { Skeleton } from "@/components/ui/skeleton";
import { AttentionSkeleton, ModuleSkeleton } from "@/features/erp/home/skeletons";

export default function ErpHomeLoading() {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-12" />
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-4 w-96 max-w-full" />
      </div>
      <div className="flex flex-wrap gap-2">
        <Skeleton className="h-8 w-44 rounded-full" />
        <Skeleton className="h-8 w-44 rounded-full" />
      </div>
      <AttentionSkeleton />
      <ModuleSkeleton />
      <ModuleSkeleton />
    </>
  );
}
