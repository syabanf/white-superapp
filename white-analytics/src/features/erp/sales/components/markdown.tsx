import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * Minimal Markdown for contract bodies: headings, paragraphs, bullet and
 * numbered lists, pipe tables, rules, **bold** and *italic*. Builds React
 * nodes (no HTML injection), safe in server and client components.
 */
export function Markdown({ source, className }: { source: string; className?: string }) {
  return <div className={cn("contract-md space-y-3 text-sm leading-relaxed", className)}>{blocks(source)}</div>;
}

function inline(text: string, keyBase: string): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  const re = /(\*\*[^*]+\*\*|\*[^*]+\*)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let i = 0;
  while ((m = re.exec(text))) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const tok = m[0];
    out.push(
      tok.startsWith("**") ? (
        <strong key={`${keyBase}-${i++}`}>{tok.slice(2, -2)}</strong>
      ) : (
        <em key={`${keyBase}-${i++}`}>{tok.slice(1, -1)}</em>
      ),
    );
    last = m.index + tok.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

const cells = (line: string) =>
  line
    .trim()
    .replace(/^\|/, "")
    .replace(/\|$/, "")
    .split("|")
    .map((c) => c.trim());

function blocks(source: string): React.ReactNode[] {
  const lines = source.replace(/\r\n/g, "\n").split("\n");
  const out: React.ReactNode[] = [];
  let i = 0;
  let k = 0;
  while (i < lines.length) {
    const line = lines[i];
    const t = line.trim();
    if (!t) {
      i++;
      continue;
    }
    const key = `b${k++}`;
    const h = /^(#{1,4})\s+(.*)$/.exec(t);
    if (h) {
      const level = h[1].length;
      const cls =
        level === 1 ? "text-lg font-semibold text-center" : level === 2 ? "pt-2 text-[15px] font-semibold" : "font-semibold";
      out.push(React.createElement(`h${Math.min(level + 1, 6)}`, { key, className: cls }, inline(h[2], key)));
      i++;
      continue;
    }
    if (/^(-{3,}|\*{3,})$/.test(t)) {
      out.push(<hr key={key} className="border-border" />);
      i++;
      continue;
    }
    if (t.startsWith("|")) {
      const rows: string[][] = [];
      while (i < lines.length && lines[i].trim().startsWith("|")) {
        const r = lines[i].trim();
        if (!/^\|?\s*:?-{2,}/.test(r)) rows.push(cells(r));
        i++;
      }
      const [head, ...body] = rows;
      out.push(
        <div key={key} className="overflow-x-auto">
          <table className="w-full border-collapse text-xs">
            <thead>
              <tr>
                {head?.map((c, j) => (
                  <th key={j} className="border px-2 py-1.5 text-left font-semibold">
                    {inline(c, `${key}h${j}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {body.map((r, ri) => (
                <tr key={ri}>
                  {r.map((c, j) => (
                    <td key={j} className="border px-2 py-1.5 align-top">
                      {inline(c, `${key}r${ri}c${j}`)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      );
      continue;
    }
    const bullet = /^[-*]\s+/;
    const numbered = /^\d+[.)]\s+/;
    if (bullet.test(t) || numbered.test(t)) {
      const ordered = numbered.test(t);
      const re = ordered ? numbered : bullet;
      const items: string[] = [];
      while (i < lines.length && re.test(lines[i].trim())) {
        items.push(lines[i].trim().replace(re, ""));
        i++;
      }
      const Tag = ordered ? "ol" : "ul";
      out.push(
        <Tag key={key} className={cn("space-y-1 pl-5", ordered ? "list-decimal" : "list-disc")}>
          {items.map((it, j) => (
            <li key={j}>{inline(it, `${key}-${j}`)}</li>
          ))}
        </Tag>,
      );
      continue;
    }
    const para: string[] = [];
    while (i < lines.length && lines[i].trim() && !/^(#{1,4}\s|\||[-*]\s|\d+[.)]\s|-{3,}$)/.test(lines[i].trim())) {
      para.push(lines[i].trim());
      i++;
    }
    if (!para.length) {
      para.push(t);
      i++;
    }
    out.push(<p key={key}>{inline(para.join(" "), key)}</p>);
  }
  return out;
}
