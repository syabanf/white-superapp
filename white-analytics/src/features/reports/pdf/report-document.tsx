import { Document, Page, StyleSheet, Text, View } from "@react-pdf/renderer";
import { pdfLabels } from "@/features/reports/strings";
import type { PdfChart, PdfKpi, PdfModuleBlock, PdfSection, PdfTable, ReportPdfData } from "./types";

/**
 * WHITE monochrome A4 report. Built-in Helvetica only (no network fonts) —
 * `safe()` strips the few Unicode chars WinAnsi cannot encode.
 */

const INK = "#111111";
const MUTED = "#6b6b70";
const FAINT = "#e5e5ea";
const BG = "#f5f5f7";

// WinAnsi extras beyond Latin-1 that the built-in Helvetica CAN encode.
const WINANSI_EXTRA = "\u20ac\u201a\u0192\u201e\u2026\u2020\u2021\u02c6\u2030\u0160\u2039\u0152\u017d\u2018\u2019\u201c\u201d\u2022\u2013\u2014\u02dc\u2122\u0161\u203a\u0153\u017e\u0178";
const NON_WINANSI = new RegExp(`[^\\x00-\\xff${WINANSI_EXTRA}]`, "g");

export function safe(s: string): string {
  return s
    .replace(/\u2212/g, "-") // minus sign
    .replace(/[\u202f\u2009\u00a0]/g, " ") // narrow/thin/no-break spaces
    .replace(/[\u2028\u2029]/g, "\n")
    .replace(NON_WINANSI, ""); // emojis & other glyphs Helvetica cannot encode
}

const styles = StyleSheet.create({
  page: { fontFamily: "Helvetica", fontSize: 9, color: INK, paddingTop: 48, paddingBottom: 56, paddingHorizontal: 48 },
  cover: { fontFamily: "Helvetica", color: INK, padding: 48, display: "flex", flexDirection: "column" },
  wordmark: { fontFamily: "Helvetica-Bold", fontSize: 28, letterSpacing: -0.5 },
  h1: { fontFamily: "Helvetica-Bold", fontSize: 24, marginTop: 8, lineHeight: 1.2 },
  h2: { fontFamily: "Helvetica-Bold", fontSize: 14, marginBottom: 10 },
  h3: { fontFamily: "Helvetica-Bold", fontSize: 10.5, marginTop: 12, marginBottom: 6 },
  label: { fontSize: 8, color: MUTED, textTransform: "uppercase", letterSpacing: 0.8 },
  footer: {
    position: "absolute",
    bottom: 24,
    left: 48,
    right: 48,
    flexDirection: "row",
    justifyContent: "space-between",
    borderTopWidth: 0.5,
    borderTopColor: FAINT,
    paddingTop: 8,
    fontSize: 7.5,
    color: MUTED,
  },
  kpiRow: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  kpiTile: { width: "31.5%", borderWidth: 0.5, borderColor: FAINT, borderRadius: 6, padding: 10, backgroundColor: "#ffffff" },
  kpiLabel: { fontSize: 7.5, color: MUTED, marginBottom: 4 },
  kpiValue: { fontFamily: "Helvetica-Bold", fontSize: 14 },
  kpiDelta: { fontSize: 7.5, marginTop: 3 },
  tableHead: { flexDirection: "row", borderBottomWidth: 1, borderBottomColor: INK, paddingBottom: 4, marginBottom: 2 },
  tableRow: { flexDirection: "row", borderBottomWidth: 0.5, borderBottomColor: FAINT, paddingVertical: 4 },
  thText: { fontFamily: "Helvetica-Bold", fontSize: 8 },
  tdText: { fontSize: 8.5 },
});

function Footer({ data }: { data: ReportPdfData }) {
  const L = pdfLabels[data.lang];
  return (
    <View style={styles.footer} fixed>
      <Text>{safe(`${L.footer} · ${data.clientName}`)}</Text>
      <Text render={({ pageNumber, totalPages }) => safe(`${L.page} ${pageNumber} / ${totalPages}`)} />
    </View>
  );
}

function Kpis({ kpis }: { kpis: PdfKpi[] }) {
  return (
    <View style={styles.kpiRow}>
      {kpis.map((k, i) => (
        <View key={i} style={styles.kpiTile} wrap={false}>
          <Text style={styles.kpiLabel}>{safe(k.label.toUpperCase())}</Text>
          <Text style={styles.kpiValue}>{safe(k.value)}</Text>
          {k.delta ? (
            <Text style={[styles.kpiDelta, { color: k.deltaGood === false ? "#8a3434" : k.deltaGood === true ? "#2f6b3a" : MUTED }]}>
              {safe(k.delta)}
            </Text>
          ) : null}
        </View>
      ))}
    </View>
  );
}

/**
 * Bars drawn as plain Views — monochrome, hairline baseline, first/last date labels.
 * Supports negative values (e.g. daily follower growth): positives sit above the
 * baseline in ink, negatives hang below in gray. ~2pt gap so bars read as marks.
 */
function BarChart({ chart }: { chart: PdfChart }) {
  const H = 72;
  const values = chart.points.map((p) => p.value);
  const maxPos = Math.max(0, ...values);
  const maxNeg = Math.max(0, ...values.map((v) => -v));
  const span = maxPos + maxNeg;
  const posH = span === 0 ? H : (maxPos / span) * H;
  const negH = span === 0 ? 0 : (maxNeg / span) * H;
  const gap = chart.points.length > 60 ? 1 : 2;
  const cell = { flexGrow: 1, flexBasis: 1, marginRight: gap } as const;
  return (
    <View wrap={false} style={{ marginTop: 12 }}>
      <View style={{ flexDirection: "row", justifyContent: "space-between", marginBottom: 4 }}>
        <Text style={{ fontFamily: "Helvetica-Bold", fontSize: 9 }}>{safe(chart.title)}</Text>
        <Text style={{ fontSize: 7.5, color: MUTED }}>{safe(`maks ${chart.maxLabel}`)}</Text>
      </View>
      <View style={{ flexDirection: "row", alignItems: "flex-end", height: posH }}>
        {chart.points.map((p, i) => (
          <View
            key={i}
            style={{
              ...cell,
              height: p.value > 0 && maxPos > 0 ? Math.max(1, (p.value / maxPos) * posH) : 0,
              backgroundColor: INK,
              opacity: 0.85,
              borderTopLeftRadius: 1,
              borderTopRightRadius: 1,
            }}
          />
        ))}
      </View>
      <View style={{ borderBottomWidth: 0.5, borderBottomColor: INK }} />
      {negH > 0 ? (
        <View style={{ flexDirection: "row", alignItems: "flex-start", height: negH }}>
          {chart.points.map((p, i) => (
            <View
              key={i}
              style={{
                ...cell,
                height: p.value < 0 && maxNeg > 0 ? Math.max(1, (-p.value / maxNeg) * negH) : 0,
                backgroundColor: "#98989d",
                borderBottomLeftRadius: 1,
                borderBottomRightRadius: 1,
              }}
            />
          ))}
        </View>
      ) : null}
      <View style={{ flexDirection: "row", justifyContent: "space-between", marginTop: 3 }}>
        <Text style={{ fontSize: 7, color: MUTED }}>{safe(chart.points[0]?.date ?? "")}</Text>
        <Text style={{ fontSize: 7, color: MUTED }}>{safe(chart.points[chart.points.length - 1]?.date ?? "")}</Text>
      </View>
    </View>
  );
}

function TableRow({ table, row }: { table: PdfTable; row: string[] }) {
  return (
    <View style={styles.tableRow} wrap={false}>
      {row.map((cell, ci) => {
        const col = table.columns[ci];
        return (
          <Text key={ci} style={[styles.tdText, { width: `${col?.width ?? 20}%`, textAlign: col?.align ?? "left", paddingRight: 6 }]}>
            {safe(cell)}
          </Text>
        );
      })}
    </View>
  );
}

/** Title, header and first row stay together so a page break never strands the header. */
function Table({ table }: { table: PdfTable }) {
  const [first, ...rest] = table.rows;
  return (
    <View style={{ marginTop: 14 }}>
      <View wrap={false}>
        <Text style={{ fontFamily: "Helvetica-Bold", fontSize: 9, marginBottom: 6 }}>{safe(table.title)}</Text>
        <View style={styles.tableHead}>
          {table.columns.map((c, i) => (
            <Text key={i} style={[styles.thText, { width: `${c.width}%`, textAlign: c.align ?? "left", paddingRight: 6 }]}>
              {safe(c.label)}
            </Text>
          ))}
        </View>
        {first ? <TableRow table={table} row={first} /> : null}
      </View>
      {rest.map((row, ri) => (
        <TableRow key={ri} table={table} row={row} />
      ))}
    </View>
  );
}

/** Markdown-lite → PDF: #/##/### headings, -/* bullets, "1." lists, **bold** inline. */
function MarkdownPdf({ text }: { text: string }) {
  const lines = text.split(/\r?\n/);
  const out: React.ReactNode[] = [];
  lines.forEach((raw, i) => {
    const line = raw.trimEnd();
    if (!line.trim()) return;
    const h = /^(#{1,3})\s+(.*)$/.exec(line);
    if (h) {
      out.push(
        <Text key={i} style={styles.h3}>
          {inlineBold(h[2]!)}
        </Text>,
      );
      return;
    }
    const ul = /^\s*[-*•]\s+(.*)$/.exec(line);
    if (ul) {
      out.push(
        <View key={i} style={{ flexDirection: "row", marginBottom: 3, paddingLeft: 4 }}>
          <Text style={{ fontSize: 9, marginRight: 5 }}>{"•"}</Text>
          <Text style={{ fontSize: 9, lineHeight: 1.45, flex: 1 }}>{inlineBold(ul[1]!)}</Text>
        </View>,
      );
      return;
    }
    const ol = /^\s*(\d+)[.)]\s+(.*)$/.exec(line);
    if (ol) {
      out.push(
        <View key={i} style={{ flexDirection: "row", marginBottom: 3, paddingLeft: 4 }}>
          <Text style={{ fontFamily: "Helvetica-Bold", fontSize: 9, marginRight: 5 }}>{safe(`${ol[1]}.`)}</Text>
          <Text style={{ fontSize: 9, lineHeight: 1.45, flex: 1 }}>{inlineBold(ol[2]!)}</Text>
        </View>,
      );
      return;
    }
    out.push(
      <Text key={i} style={{ fontSize: 9, lineHeight: 1.5, marginBottom: 4 }}>
        {inlineBold(line)}
      </Text>,
    );
  });
  return <View>{out}</View>;
}

function inlineBold(s: string): React.ReactNode {
  const parts = s.split(/(\*\*[^*]+\*\*)/g);
  return parts.map((p, i) =>
    p.startsWith("**") && p.endsWith("**") ? (
      <Text key={i} style={{ fontFamily: "Helvetica-Bold" }}>
        {safe(p.slice(2, -2))}
      </Text>
    ) : (
      <Text key={i}>{safe(p)}</Text>
    ),
  );
}

function CoverPage({ data }: { data: ReportPdfData }) {
  const L = pdfLabels[data.lang];
  return (
    <Page size="A4" style={styles.cover}>
      <View style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
        <View style={{ width: 12, height: 12, backgroundColor: INK, borderRadius: 6 }} />
        <Text style={styles.wordmark}>white</Text>
      </View>
      <View style={{ marginTop: 140 }}>
        <Text style={styles.label}>{safe(L.report.toUpperCase())}</Text>
        <Text style={styles.h1}>{safe(data.title)}</Text>
      </View>
      <View style={{ marginTop: 28, borderTopWidth: 1, borderTopColor: INK, paddingTop: 16, gap: 12 }}>
        <View>
          <Text style={styles.label}>{safe(L.preparedFor.toUpperCase())}</Text>
          <Text style={{ fontFamily: "Helvetica-Bold", fontSize: 13, marginTop: 3 }}>{safe(data.clientName)}</Text>
        </View>
        <View style={{ flexDirection: "row", gap: 48 }}>
          <View>
            <Text style={styles.label}>{safe(L.period.toUpperCase())}</Text>
            <Text style={{ fontSize: 10, marginTop: 3 }}>{safe(data.periodLabel)}</Text>
            {data.comparedLabel ? <Text style={{ fontSize: 8, color: MUTED, marginTop: 2 }}>{safe(data.comparedLabel)}</Text> : null}
          </View>
          <View>
            <Text style={styles.label}>{safe(L.generatedOn.toUpperCase())}</Text>
            <Text style={{ fontSize: 10, marginTop: 3 }}>{safe(data.generatedLabel)}</Text>
          </View>
        </View>
      </View>
      <View style={{ position: "absolute", bottom: 40, left: 48, right: 48, flexDirection: "row", justifyContent: "space-between" }}>
        <Text style={{ fontSize: 8, color: MUTED }}>{safe(L.footer)}</Text>
        <Text style={{ fontSize: 8, color: MUTED }}>{safe(data.modules.map((m) => m.title).join(" · "))}</Text>
      </View>
    </Page>
  );
}

/** Sub-block under a module: hairline rule, heading kept with its KPI row, optional table. */
function Section({ section }: { section: PdfSection }) {
  return (
    <View style={{ marginTop: 20, borderTopWidth: 0.5, borderTopColor: FAINT, paddingTop: 12 }}>
      <View wrap={false}>
        <Text style={{ fontFamily: "Helvetica-Bold", fontSize: 11, marginBottom: 8 }}>{safe(section.title)}</Text>
        <Kpis kpis={section.kpis} />
      </View>
      {section.table && section.table.rows.length > 0 ? <Table table={section.table} /> : null}
    </View>
  );
}

function ModulePage({ block, data }: { block: PdfModuleBlock; data: ReportPdfData }) {
  return (
    <Page size="A4" style={styles.page}>
      <View style={{ flexDirection: "row", justifyContent: "space-between", alignItems: "flex-end", borderBottomWidth: 1, borderBottomColor: INK, paddingBottom: 8, marginBottom: 14 }}>
        <Text style={styles.h2}>{safe(block.title)}</Text>
        <Text style={{ fontSize: 8, color: MUTED }}>{safe(`${data.clientName} · ${data.periodLabel}`)}</Text>
      </View>
      <Kpis kpis={block.kpis} />
      {block.chart && block.chart.points.length > 1 ? <BarChart chart={block.chart} /> : null}
      {block.tables.map((tb, i) => (
        <Table key={i} table={tb} />
      ))}
      {block.sections.map((sec, i) => (
        <Section key={i} section={sec} />
      ))}
      <Footer data={data} />
    </Page>
  );
}

export function ReportDocument({ data }: { data: ReportPdfData }) {
  const L = pdfLabels[data.lang];
  return (
    <Document title={data.title} author="WHITE Analytics" creator="WHITE Analytics" producer="WHITE Analytics">
      <CoverPage data={data} />
      {data.modules.map((m) => (
        <ModulePage key={m.key} block={m} data={data} />
      ))}
      {data.aiSummary ? (
        <Page size="A4" style={styles.page}>
          <View style={{ borderBottomWidth: 1, borderBottomColor: INK, paddingBottom: 8, marginBottom: 14 }}>
            <Text style={styles.h2}>{safe(L.executiveSummary)}</Text>
          </View>
          <MarkdownPdf text={data.aiSummary} />
          <Footer data={data} />
        </Page>
      ) : null}
      <Page size="A4" style={styles.cover}>
        <View style={{ marginTop: 240, alignItems: "center", gap: 10 }}>
          <View style={{ width: 10, height: 10, backgroundColor: INK, borderRadius: 5 }} />
          <Text style={{ fontFamily: "Helvetica-Bold", fontSize: 20 }}>{safe(L.closing)}</Text>
          <Text style={{ fontSize: 9.5, color: MUTED, textAlign: "center", maxWidth: 320, lineHeight: 1.5 }}>{safe(L.closingBody)}</Text>
        </View>
        <View style={{ position: "absolute", bottom: 40, left: 48, right: 48, alignItems: "center" }}>
          <Text style={{ fontSize: 8, color: MUTED, backgroundColor: BG, paddingHorizontal: 8, paddingVertical: 3, borderRadius: 3 }}>
            {safe(`${L.footer} · ${data.generatedLabel}`)}
          </Text>
        </View>
      </Page>
    </Document>
  );
}
