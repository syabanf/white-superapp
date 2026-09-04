import { cn } from "@/lib/utils";
import { rateScore, type CwvRating } from "@/lib/metrics";
import { t } from "@/i18n/id";

const RATING_COLOR: Record<CwvRating, string> = {
  good: "var(--status-good)",
  "needs-improvement": "var(--status-warning)",
  poor: "var(--status-critical)",
};

export const RATING_LABEL: Record<CwvRating, string> = {
  good: t.common.good,
  "needs-improvement": t.common.needsImprovement,
  poor: t.common.poor,
};

/**
 * Meter for a 0–100 score (Lighthouse style). Fill = status color by rating,
 * track = muted. Always paired with a numeric label (never color alone).
 */
export function ScoreRing({
  score,
  label,
  size = 84,
  stroke = 7,
  className,
  rating,
}: {
  score: number | null | undefined;
  label?: string;
  size?: number;
  stroke?: number;
  className?: string;
  rating?: CwvRating;
}) {
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const val = score == null ? 0 : Math.max(0, Math.min(100, score));
  const rt = rating ?? (score == null ? null : rateScore(val));
  const color = rt ? RATING_COLOR[rt] : "var(--muted-foreground)";
  return (
    <div className={cn("flex flex-col items-center gap-1.5", className)}>
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} className="-rotate-90">
          <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--muted)" strokeWidth={stroke} />
          <circle
            cx={size / 2}
            cy={size / 2}
            r={r}
            fill="none"
            stroke={color}
            strokeWidth={stroke}
            strokeLinecap="round"
            strokeDasharray={`${(val / 100) * c} ${c}`}
            className="transition-[stroke-dasharray] duration-500"
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-lg font-semibold" style={{ fontSize: size * 0.26 }}>
            {score == null ? "–" : Math.round(val)}
          </span>
        </div>
      </div>
      {label ? <span className="text-xs text-muted-foreground">{label}</span> : null}
      {rt ? (
        <span className="text-[10px] font-medium" style={{ color }}>
          {RATING_LABEL[rt]}
        </span>
      ) : null}
    </div>
  );
}
