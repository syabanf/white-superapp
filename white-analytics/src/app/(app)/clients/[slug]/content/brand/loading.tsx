import { Skeleton } from "@/components/ui/skeleton";

/** Header, then the form sections beside the brand preview card. */
export default function Loading() {
  return (
    <div className="page-stack" aria-busy="true">
      <Skeleton className="h-16 w-full max-w-md rounded-[1.25rem]" />
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_22rem]">
        <div className="order-2 grid gap-5 lg:order-1">
          <Skeleton className="h-96 rounded-[1.5rem]" />
          <Skeleton className="h-80 rounded-[1.5rem]" />
          <Skeleton className="h-80 rounded-[1.5rem]" />
        </div>
        <Skeleton className="order-1 h-[30rem] rounded-[1.5rem] lg:order-2" />
      </div>
    </div>
  );
}
