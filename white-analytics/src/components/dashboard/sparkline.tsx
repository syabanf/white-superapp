import { cn } from "@/lib/utils";

/**
 * Tiny inline SVG sparkline (no deps). Muted line, last point in accent.
 * Sized by CSS (default 72×24).
 */
export function Sparkline({
  values,
  className,
  width = 72,
  height = 24,
  stroke = "var(--chart-emphasis-muted)",
  accent = "var(--foreground)",
}: {
  values: number[];
  className?: string;
  width?: number;
  height?: number;
  stroke?: string;
  accent?: string;
}) {
  if (!values || values.length < 2) return <svg width={width} height={height} className={className} aria-hidden />;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min || 1;
  const pad = 3;
  const w = width - pad * 2;
  const h = height - pad * 2;
  const pts = values.map((v, i) => {
    const x = pad + (i / (values.length - 1)) * w;
    const y = pad + h - ((v - min) / span) * h;
    return [x, y] as const;
  });
  const d = pts.map(([x, y], i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`).join(" ");
  const [lx, ly] = pts[pts.length - 1]!;
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} className={cn("shrink-0", className)} aria-hidden>
      <path d={d} fill="none" stroke={stroke} strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
      <circle cx={lx} cy={ly} r={2.5} fill={accent} stroke="var(--card)" strokeWidth={1.5} />
    </svg>
  );
}
