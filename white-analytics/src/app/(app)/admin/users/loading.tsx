import { Skeleton } from "@/components/ui/skeleton";

export default function AdminUsersLoading() {
  return (
    <>
      <div className="flex items-end justify-between">
        <div className="space-y-2">
          <Skeleton className="h-3 w-14" />
          <Skeleton className="h-7 w-48" />
          <Skeleton className="h-4 w-64" />
        </div>
        <Skeleton className="h-8 w-28" />
      </div>
      <div className="space-y-3">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-72 rounded-lg" />
      </div>
    </>
  );
}
