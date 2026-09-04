import { Skeleton } from "@/components/ui/skeleton";

export default function Loading() {
  return (
    <div className="mx-auto w-full max-w-5xl space-y-6">
      <Skeleton className="h-20 w-2/3" />
      <div className="grid gap-8 lg:grid-cols-[220px_1fr]">
        <Skeleton className="hidden h-64 lg:block" />
        <Skeleton className="h-[480px]" />
      </div>
    </div>
  );
}
