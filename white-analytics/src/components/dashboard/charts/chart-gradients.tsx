/**
 * Shared SVG gradients for chart marks. Rendered once in the root layout; any
 * inline chart SVG on the page can reference them with `url(#id)`. Each series
 * hue runs from its validated color at the data end to a lighter tint at the
 * baseline, so the hue a reader matches against the legend stays the same.
 */
const PAINTS: Record<string, string> = {
  "var(--chart-1)": "c1",
  "var(--chart-2)": "c2",
  "var(--chart-3)": "c3",
  "var(--chart-4)": "c4",
  "var(--chart-5)": "c5",
  "var(--chart-6)": "c6",
  "var(--chart-7)": "c7",
  "var(--chart-8)": "c8",
  "var(--chart-emphasis-muted)": "muted",
  "var(--positive)": "pos",
  "var(--negative)": "neg",
  "var(--brand)": "brand",
};

/**
 * Gradient paint for a bar in `color`, falling back to the flat color when the
 * color has no shared gradient (or the defs are missing). `negative` flips a
 * vertical ramp so a bar below zero is saturated at its data end too.
 */
export function barFill(color: string, horizontal = false, negative = false): string {
  const id = PAINTS[color];
  if (!id) return color;
  const dir = horizontal ? "h" : negative ? "n" : "v";
  return `url(#mx-${dir}-${id}) ${color}`;
}

/** Soft wash under a line in `color`; no wash for colors without a shared gradient. */
export function areaFill(color: string): string {
  const id = PAINTS[color];
  return id ? `url(#mx-a-${id}) none` : "none";
}

/** The same tint ramp as a CSS background, for HTML bars (meters, funnels). */
export function cssBarFill(color: string): string {
  return `linear-gradient(90deg, color-mix(in oklab, ${color} 52%, var(--card)), ${color})`;
}

export function ChartGradientDefs() {
  const entries = Object.entries(PAINTS);
  return (
    <svg aria-hidden width="0" height="0" focusable="false" style={{ position: "absolute" }}>
      <defs>
        {entries.map(([color, id]) => (
          <linearGradient key={`v-${id}`} id={`mx-v-${id}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" style={{ stopColor: color }} />
            <stop offset="100%" style={{ stopColor: `color-mix(in oklab, ${color} 52%, var(--card))` }} />
          </linearGradient>
        ))}
        {entries.map(([color, id]) => (
          <linearGradient key={`n-${id}`} id={`mx-n-${id}`} x1="0" y1="1" x2="0" y2="0">
            <stop offset="0%" style={{ stopColor: color }} />
            <stop offset="100%" style={{ stopColor: `color-mix(in oklab, ${color} 52%, var(--card))` }} />
          </linearGradient>
        ))}
        {entries.map(([color, id]) => (
          <linearGradient key={`h-${id}`} id={`mx-h-${id}`} x1="1" y1="0" x2="0" y2="0">
            <stop offset="0%" style={{ stopColor: color }} />
            <stop offset="100%" style={{ stopColor: `color-mix(in oklab, ${color} 52%, var(--card))` }} />
          </linearGradient>
        ))}
        {entries.map(([color, id]) => (
          <linearGradient key={`a-${id}`} id={`mx-a-${id}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" style={{ stopColor: color, stopOpacity: 0.2 }} />
            <stop offset="100%" style={{ stopColor: color, stopOpacity: 0 }} />
          </linearGradient>
        ))}
      </defs>
    </svg>
  );
}
