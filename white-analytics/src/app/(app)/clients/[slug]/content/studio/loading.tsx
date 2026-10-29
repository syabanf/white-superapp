import { Skeleton } from "@/components/ui/skeleton";

/** Header, brief form, result (image + designer brief), history grid. */
export default function Loading() {
  return (
    <div className="page-stack" aria-busy="true">
      <Skeleton className="h-16 w-full max-w-md rounded-[1.25rem]" />
      <Skeleton className="h-64 rounded-[1.5rem]" />
      <div className="grid gap-5 lg:grid-cols-2">
        <Skeleton className="h-[28rem] rounded-[1.5rem]" />
        <Skeleton className="h-[28rem] rounded-[1.5rem]" />
      </div>
      <div className="grid grid-cols-2 gap-5 sm:grid-cols-3 lg:grid-cols-6">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="aspect-[4/5] rounded-[1.25rem]" />
        ))}
      </div>
    </div>
  );
}
