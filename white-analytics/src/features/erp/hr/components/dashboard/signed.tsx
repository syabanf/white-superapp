import { cn } from "@/lib/utils";

/** "+3 / -1" with joiners in the positive tone and leavers in the negative tone; zero stays muted. */
export function SignedPair({ plus, minus }: { plus: number; minus: number }) {
  return (
    <span className="tabular">
      <span className={cn(plus > 0 ? "text-positive" : "text-muted-foreground")}>+{plus}</span>
      <span className="text-muted-foreground"> / </span>
      <span className={cn(minus > 0 ? "text-negative" : "text-muted-foreground")}>-{minus}</span>
    </span>
  );
}
