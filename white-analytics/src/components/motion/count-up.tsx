"use client";

import * as React from "react";

type Figure = { number: number; render: (n: number) => string };

/**
 * Reads the first number out of an already-formatted value ("Rp 1.250.000",
 * "12,5%", "$1.19M") and returns a renderer that rewrites that number while
 * keeping the prefix, suffix, separators and decimal places. A single separator
 * followed by exactly three digits counts as grouping (id-ID "1.250").
 */
export function parseFigure(text: string): Figure | null {
  const match = /-?\d[\d.,]*/.exec(text);
  if (!match) return null;
  const token = match[0].replace(/[.,]+$/, "");
  const dots = token.split(".").length - 1;
  const commas = token.split(",").length - 1;

  let decimal: "." | "," | null = null;
  if (dots && commas) decimal = token.lastIndexOf(".") > token.lastIndexOf(",") ? "." : ",";
  else if (dots + commas === 1) {
    const sep = dots ? "." : ",";
    if (token.split(sep)[1].length !== 3) decimal = sep;
  }
  const group = decimal === "." ? "," : decimal === "," ? "." : dots ? "." : commas ? "," : "";
  const places = decimal ? token.split(decimal)[1].length : 0;
  const digits = group ? token.split(group).join("") : token;
  const magnitude = Number(decimal ? digits.replace(decimal, ".") : digits);
  if (!Number.isFinite(magnitude)) return null;

  // "-Rp 3.400" puts the sign before the currency; keep it there.
  let before = text.slice(0, match.index);
  const signFirst = before.startsWith("-");
  if (signFirst) before = before.slice(1);
  const after = text.slice(match.index + token.length);
  return {
    number: signFirst ? -magnitude : magnitude,
    render: (n) => {
      const [int, frac] = Math.abs(n).toFixed(places).split(".");
      const grouped = group ? int.replace(/\B(?=(\d{3})+(?!\d))/g, group) : int;
      const sign = n < 0 ? "-" : "";
      const figure = `${grouped}${frac ? decimal + frac : ""}`;
      return signFirst ? `${sign}${before}${figure}${after}` : `${before}${sign}${figure}${after}`;
    },
  };
}

const easeOut = (p: number) => 1 - (1 - p) ** 3;

/**
 * Counts a formatted figure up from zero on mount, and from the previous value
 * when it changes. Frames write straight to the text node React rendered, so
 * React only re-renders when the value itself changes. Screen readers get the
 * final value only; reduced-motion users see it at once.
 */
export function CountUp({ value, duration = 900 }: { value: string; duration?: number }) {
  const ref = React.useRef<HTMLSpanElement>(null);
  // Last number on screen, so a new value (or a Strict Mode re-run) continues from it.
  const at = React.useRef(0);

  React.useLayoutEffect(() => {
    const text = ref.current?.firstChild;
    const figure = parseFigure(value);
    if (!text || !figure || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    const start = performance.now();
    const a = at.current;
    const b = figure.number;
    let frame = 0;
    const tick = (now: number) => {
      const p = Math.min(1, (now - start) / duration);
      at.current = a + (b - a) * easeOut(p);
      text.nodeValue = p === 1 ? value : figure.render(at.current);
      if (p < 1) frame = requestAnimationFrame(tick);
    };
    tick(start);
    return () => {
      cancelAnimationFrame(frame);
      text.nodeValue = value;
    };
  }, [value, duration]);

  return (
    <>
      <span className="sr-only">{value}</span>
      <span ref={ref} aria-hidden className="tabular-nums">
        {value}
      </span>
    </>
  );
}
