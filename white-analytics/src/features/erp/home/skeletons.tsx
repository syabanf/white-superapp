import { Skeleton } from "@/components/ui/skeleton";

// Kept apart from sections.tsx: erp/loading.tsx wraps every /erp page, so
// anything it imports ships on all of them (sections.tsx pulls in recharts).

export function ModuleSkeleton() {
  return (
    <div className="flex flex-col gap-(--gap-inner)" aria-busy="true">
      <Skeleton className="h-8 w-40" />
      <div className="page-stack">
        <div className="grid grid-cols-2 gap-(--gap) lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-32 rounded-[1.5rem]" />
          ))}
        </div>
        <Skeleton className="h-80 rounded-[1.5rem]" />
      </div>
    </div>
  );
}

export function AttentionSkeleton() {
  return (
    <div className="flex flex-col gap-(--gap-inner)" aria-busy="true">
      <Skeleton className="h-8 w-36" />
      <div className="grid grid-cols-2 gap-(--gap) sm:grid-cols-[repeat(auto-fit,minmax(8rem,1fr))]">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-28 rounded-xl" />
        ))}
      </div>
    </div>
  );
}
