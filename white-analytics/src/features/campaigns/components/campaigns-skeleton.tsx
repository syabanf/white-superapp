import { Skeleton } from "@/components/ui/skeleton";

function HeaderSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-4 w-24" />
      <Skeleton className="h-3 w-20" />
      <Skeleton className="h-8 w-56" />
    </div>
  );
}

/** Route-level loading for the campaign list and the builder. */
export function CampaignsSkeleton({ variant }: { variant: "list" | "builder" }) {
  if (variant === "list") {
    return (
      <>
        <HeaderSkeleton />
        <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-44 rounded-[1.5rem]" />
          ))}
        </div>
      </>
    );
  }
  return (
    <>
      <HeaderSkeleton />
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px] xl:grid-cols-[minmax(0,1fr)_380px]">
        <div className="space-y-5">
          <Skeleton className="h-96 rounded-[1.5rem]" />
          <Skeleton className="h-56 rounded-[1.5rem]" />
          <Skeleton className="h-40 rounded-[1.5rem]" />
          <Skeleton className="h-96 rounded-[1.5rem]" />
        </div>
        <Skeleton className="hidden h-[34rem] rounded-[1.5rem] lg:block" />
      </div>
    </>
  );
}
