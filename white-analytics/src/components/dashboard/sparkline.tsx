import { cn } from "@/lib/utils";
import { changeTone, seriesDelta, type Delta } from "@/lib/metrics";

const TONE_STROKE = {
  positive: "var(--positive)",
  negative: "var(--negative)",
  neutral: "var(--chart-emphasis-muted)",
} as const;

/**
 * Tiny inline SVG sparkline (no deps). The stroke is tinted by the trend over
 * the period (good = positive, bad = negative, flat = muted); pass `trend` to
 * use a known delta instead of first-vs-last, or `stroke` to opt out.
 * Sized by CSS (default 72×24).
 */
export function Sparkline({
  values,
  className,
  width = 72,
  height = 24,
  stroke,
  accent,
  trend,
  lowerIsBetter = false,
}: {
  values: number[];
  className?: string;
  width?: number;
  height?: number;
  /** fixed stroke color; disables trend tinting */
  stroke?: string;
  accent?: string;
  /** delta that decides the tint (defaults to last vs first value) */
  trend?: Delta | null;
  lowerIsBetter?: boolean;
}) {
  if (!values || values.length < 2)
    return <svg width={width} height={height} className={className} aria-hidden />;
  const tone = stroke ? "neutral" : changeTone(trend ?? seriesDelta(values), lowerIsBetter);
  const line = stroke ?? TONE_STROKE[tone];
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
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className={cn("shrink-0", className)}
      aria-hidden
    >
      <path d={d} fill="none" stroke={line} strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
      <circle
        cx={lx}
        cy={ly}
        r={2.5}
        fill={accent ?? (tone === "neutral" ? "var(--foreground)" : line)}
        stroke="var(--card)"
        strokeWidth={1.5}
      />
    </svg>
  );
}
