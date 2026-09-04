import { Skeleton } from "@/components/ui/skeleton";

/** Shared loading frame for every /publish route. */
export function PublishingSkeleton({ variant }: { variant: "calendar" | "table" | "composer" | "grid" | "detail" }) {
  return (
    <div className="flex flex-col gap-6">
      <div className="space-y-2">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-4 w-72 max-w-full" />
      </div>
      {variant === "calendar" ? (
        <>
          <Skeleton className="h-16 rounded-xl" />
          <Skeleton className="h-9 w-80 max-w-full" />
          <Skeleton className="h-[520px] rounded-xl" />
        </>
      ) : variant === "table" ? (
        <>
          <Skeleton className="h-10 w-[560px] max-w-full rounded-full" />
          <Skeleton className="h-96 rounded-xl" />
        </>
      ) : variant === "grid" ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {Array.from({ length: 12 }, (_, i) => (
            <Skeleton key={i} className="aspect-[4/5] rounded-xl" />
          ))}
        </div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
          <div className="space-y-4">
            {Array.from({ length: variant === "detail" ? 3 : 5 }, (_, i) => (
              <Skeleton key={i} className="h-40 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-[420px] rounded-xl" />
        </div>
      )}
    </div>
  );
}
