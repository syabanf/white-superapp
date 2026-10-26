import { cn } from "@/lib/utils";
import { formatRupiah, parseMoney } from "./money";

/** Rupiah amount, tabular figures; `signed` colors negatives. Safe in server components. */
export function Money({
  value,
  signed,
  muteZero,
  className,
}: {
  value: string | number | null | undefined;
  signed?: boolean;
  muteZero?: boolean;
  className?: string;
}) {
  const n = parseMoney(value);
  return (
    <span
      className={cn(
        "tabular whitespace-nowrap",
        signed && n < 0 && "text-negative",
        muteZero && n === 0 && "text-muted-foreground",
        className,
      )}
    >
      {formatRupiah(n)}
    </span>
  );
}
