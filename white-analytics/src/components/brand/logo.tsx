import { cn } from "@/lib/utils";

/**
 * WHITE brand mark — eight pinwheel petals around a small centre, drawn inline so it
 * inherits currentColor in both themes. `variant="mark"` renders only the flower.
 * (Raster original lives at /public/logo-white.png for PDF/export use.)
 */
export function WhiteLogo({
  className,
  variant = "full",
  size = 24,
}: {
  className?: string;
  variant?: "full" | "mark";
  size?: number;
}) {
  const mark = (
    <svg
      width={size}
      height={size}
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      className="shrink-0"
    >
      {Array.from({ length: 8 }).map((_, i) => (
        <path
          key={i}
          // petal: square inner-left corner, rounded outer end → pinwheel rhythm
          d="M18.6 18.4V11a5.4 5.4 0 0 1 10.8 0v5.6a1.8 1.8 0 0 1-1.8 1.8Z"
          fill="currentColor"
          transform={`rotate(${i * 45} 24 24)`}
        />
      ))}
      <circle cx="24" cy="24" r="3.4" fill="currentColor" />
    </svg>
  );
  if (variant === "mark")
    return <span className={cn("inline-flex items-center text-foreground", className)}>{mark}</span>;
  return (
    <span className={cn("inline-flex items-center gap-2 text-foreground", className)}>
      {mark}
      <span className="font-semibold tracking-tight" style={{ fontSize: size * 0.85, lineHeight: 1 }}>
        white
      </span>
    </span>
  );
}
