import { FORMAT_SIZE, type DesignFormat, type DesignLayout } from "./types";

/**
 * Renders a layout as an on-brand SVG poster: the mock "generated image" and
 * a quick visual of the composition. Pure string building, no DOM. Text is
 * escaped; the result is only ever shown as a data URL in <img>, which keeps
 * any markup inert.
 */

const esc = (s: string) => s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!);

/** Greedy word wrap by character budget (good enough for a poster preview). */
export function wrapText(text: string, maxChars: number, maxLines: number): string[] {
  const words = text.trim().split(/\s+/).filter(Boolean);
  const lines: string[] = [];
  let line = "";
  for (const w of words) {
    const next = line ? `${line} ${w}` : w;
    if (next.length > maxChars && line) {
      lines.push(line);
      line = w;
    } else line = next;
  }
  if (line) lines.push(line);
  if (lines.length > maxLines) {
    const kept = lines.slice(0, maxLines);
    kept[maxLines - 1] = `${kept[maxLines - 1]!.replace(/[.,;:!?]?$/, "")}…`;
    return kept;
  }
  return lines;
}

function textBlock(lines: string[], x: number, y: number, size: number, color: string, weight: number, anchor: "start" | "middle", font: string): string {
  return lines
    .map(
      (l, i) =>
        `<text x="${x}" y="${y + i * size * 1.15}" font-family="${esc(font)}, 'DM Sans', Arial, sans-serif" font-size="${size}" font-weight="${weight}" fill="${color}" text-anchor="${anchor}">${esc(l)}</text>`,
    )
    .join("");
}

export function renderLayoutSvg(layout: DesignLayout, format: DesignFormat, fonts: { heading: string; body: string }): string {
  const { width: W, height: H } = FORMAT_SIZE[format];
  const [bg = "#2b2265", surface = "#fbeaf2", ink = "#ffffff", accent = "#f59e0b"] = layout.palette;
  const pad = 96;
  const tall = H / W > 1.5;
  const headSize = layout.composition === "big-type" ? 132 : tall ? 104 : 92;
  const head = wrapText(layout.headline, layout.composition === "big-type" ? 12 : 16, 4);
  const sub = wrapText(layout.subline, 34, 3);
  const parts: string[] = [];

  parts.push(
    `<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${bg}"/><stop offset="1" stop-color="${accent}" stop-opacity="0.55"/></linearGradient></defs>`,
    `<rect width="${W}" height="${H}" fill="url(#g)"/>`,
  );

  const cta = (x: number, y: number, anchor: "start" | "middle") => {
    const w = Math.max(280, layout.cta.length * 26 + 96);
    const left = anchor === "middle" ? x - w / 2 : x;
    return `<rect x="${left}" y="${y}" width="${w}" height="96" rx="48" fill="${accent}"/>${textBlock([layout.cta], left + w / 2, y + 62, 40, bg, 700, "middle", fonts.body)}`;
  };

  switch (layout.composition) {
    case "split-left": {
      parts.push(`<rect x="${W * 0.55}" y="0" width="${W * 0.45}" height="${H}" fill="${surface}" opacity="0.92"/>`);
      parts.push(`<circle cx="${W * 0.78}" cy="${H * 0.42}" r="${W * 0.16}" fill="${accent}" opacity="0.8"/>`);
      parts.push(textBlock(head, pad, H * 0.3, headSize * 0.8, ink, 800, "start", fonts.heading));
      parts.push(textBlock(sub, pad, H * 0.3 + head.length * headSize * 0.95 + 40, 40, ink, 400, "start", fonts.body));
      parts.push(cta(pad, H - pad - 96, "start"));
      break;
    }
    case "bottom-band": {
      parts.push(`<rect x="0" y="${H * 0.58}" width="${W}" height="${H * 0.42}" fill="${surface}"/>`);
      parts.push(`<circle cx="${W / 2}" cy="${H * 0.3}" r="${W * 0.2}" fill="${accent}" opacity="0.75"/>`);
      parts.push(textBlock(head, pad, H * 0.58 + pad + headSize * 0.7, headSize * 0.75, bg, 800, "start", fonts.heading));
      parts.push(textBlock(sub, pad, H * 0.58 + pad + head.length * headSize * 0.85 + 70, 38, bg, 400, "start", fonts.body));
      break;
    }
    case "framed": {
      parts.push(`<rect x="${pad / 2}" y="${pad / 2}" width="${W - pad}" height="${H - pad}" rx="40" fill="none" stroke="${surface}" stroke-width="10"/>`);
      parts.push(textBlock(head, W / 2, H * 0.38, headSize, ink, 800, "middle", fonts.heading));
      parts.push(textBlock(sub, W / 2, H * 0.38 + head.length * headSize * 1.15 + 30, 40, surface, 400, "middle", fonts.body));
      parts.push(cta(W / 2, H - pad * 2 - 40, "middle"));
      break;
    }
    case "big-type": {
      parts.push(textBlock(head, pad, H * 0.22 + headSize, headSize, ink, 900, "start", fonts.heading));
      parts.push(`<rect x="${pad}" y="${H - pad - 180}" width="${W * 0.3}" height="12" rx="6" fill="${accent}"/>`);
      parts.push(textBlock(sub, pad, H - pad - 110, 40, surface, 500, "start", fonts.body));
      break;
    }
    default: {
      parts.push(`<circle cx="${W * 0.85}" cy="${H * 0.12}" r="${W * 0.22}" fill="${surface}" opacity="0.18"/>`);
      parts.push(`<circle cx="${W * 0.1}" cy="${H * 0.9}" r="${W * 0.28}" fill="${accent}" opacity="0.35"/>`);
      parts.push(textBlock(head, W / 2, H * 0.36, headSize, ink, 800, "middle", fonts.heading));
      parts.push(textBlock(sub, W / 2, H * 0.36 + head.length * headSize * 1.15 + 30, 40, surface, 400, "middle", fonts.body));
      parts.push(cta(W / 2, H * 0.36 + head.length * headSize * 1.15 + sub.length * 46 + 110, "middle"));
    }
  }

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">${parts.join("")}</svg>`;
}

export function svgDataUrl(svg: string): string {
  return `data:image/svg+xml;base64,${Buffer.from(svg, "utf8").toString("base64")}`;
}
