import type { PdfLang } from "@/features/reports/strings";

/** Fully serializable, pre-formatted data for the PDF document (no functions, no Dates). */

export type PdfKpi = {
  label: string;
  value: string;
  /** formatted delta string like "+12,3%" (null → hidden) */
  delta?: string | null;
  /** true = good change, false = bad, null = neutral */
  deltaGood?: boolean | null;
};

export type PdfChartPoint = { date: string; value: number };

export type PdfChart = {
  title: string;
  points: PdfChartPoint[];
  /** formatted max value, printed at the top-left of the plot */
  maxLabel: string;
};

export type PdfTableColumn = { label: string; width: number; align?: "left" | "right" };

export type PdfTable = {
  title: string;
  columns: PdfTableColumn[];
  rows: string[][];
};

/** Sub-block inside a module page: rank tracker, backlinks, publishing performance. */
export type PdfSection = { title: string; kpis: PdfKpi[]; table: PdfTable | null };

export type PdfModuleBlock = {
  key: "SOCIAL" | "SEO" | "ADS";
  title: string;
  kpis: PdfKpi[];
  chart: PdfChart | null;
  tables: PdfTable[];
  sections: PdfSection[];
};

export type ReportPdfData = {
  lang: PdfLang;
  title: string;
  clientName: string;
  periodLabel: string;
  /** "vs 22 Jun – 19 Jul 2026" when compare is on */
  comparedLabel: string | null;
  generatedLabel: string;
  modules: PdfModuleBlock[];
  aiSummary: string | null;
};
