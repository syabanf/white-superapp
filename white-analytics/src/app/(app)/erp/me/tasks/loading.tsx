import { Skeleton } from "@/components/ui/skeleton";

export default function MyTasksLoading() {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-8 w-44" />
      </div>
      <Skeleton className="h-8 w-64 rounded-full" />
      <Skeleton className="h-11 rounded-xl" />
      <div className="space-y-2">
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} className="h-16 rounded-xl" />
        ))}
      </div>
    </>
  );
}
