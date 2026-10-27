import { Skeleton } from "@/components/ui/skeleton";

/** Header, optional tile row, and a table-sized block. */
export function PageSkeleton({ tiles = 0 }: { tiles?: number }) {
  return (
    <>
      <div className="space-y-2">
        <Skeleton className="h-3 w-14" />
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-4 w-72" />
      </div>
      <Skeleton className="h-(--control-h) w-80 max-w-full rounded-full" />
      {tiles ? (
        <div className="grid grid-cols-2 gap-(--gap) md:grid-cols-4">
          {Array.from({ length: tiles }, (_, i) => (
            <Skeleton key={i} className="h-24 rounded-[1.5rem]" />
          ))}
        </div>
      ) : null}
      <Skeleton className="h-80 rounded-[1.5rem]" />
    </>
  );
}
