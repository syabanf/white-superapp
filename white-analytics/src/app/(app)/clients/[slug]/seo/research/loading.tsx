import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="flex flex-col gap-6">
      <div className="space-y-2">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-80 max-w-full" />
      </div>
      <Skeleton className="h-10 w-full max-w-xl" />
      <Skeleton className="h-16 rounded-xl" />
      <div className="space-y-4 rounded-xl border p-5">
        <Skeleton className="h-8 w-96 max-w-full" />
        <Skeleton className="h-96 rounded-lg" />
      </div>
    </div>
  );
}
