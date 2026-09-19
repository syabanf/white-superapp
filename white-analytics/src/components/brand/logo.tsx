import { cn } from "@/lib/utils";

/**
 * WHITE brand mark: eight shutter blades around an open centre, drawn inline so it
 * inherits currentColor in both themes. `variant="mark"` renders only the blades.
 * Traced by eye from the raster logo. Swap the path for the official SVG when it arrives.
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
          // blade: straight leading edge, swept trailing curve, open centre
          d="M21.2 17.6V4.6c6 .2 9.8 4.9 8.7 10.6l-.5 2.4Z"
          fill="currentColor"
          transform={`rotate(${i * 45} 24 24)`}
        />
      ))}
    </svg>
  );
  if (variant === "mark")
    return <span className={cn("inline-flex items-center text-foreground", className)}>{mark}</span>;
  return (
    <span className={cn("inline-flex items-center gap-2 text-foreground", className)}>
      {mark}
      <span className="relative font-semibold tracking-[-0.04em]" style={{ fontSize: size * 0.95, lineHeight: 1 }}>
        white
      </span>
    </span>
  );
}
