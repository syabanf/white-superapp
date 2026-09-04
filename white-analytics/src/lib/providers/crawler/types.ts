/** Crawler contract + on-page issue catalog (messages in Bahasa Indonesia, matching seed wording). */

export type IssueSeverityKey = "ERROR" | "WARNING" | "NOTICE";

export type IssueCode =
  | "MISSING_TITLE"
  | "TITLE_TOO_LONG"
  | "TITLE_TOO_SHORT"
  | "MISSING_META_DESCRIPTION"
  | "META_DESCRIPTION_TOO_LONG"
  | "MISSING_H1"
  | "MULTIPLE_H1"
  | "IMG_MISSING_ALT"
  | "BROKEN_INTERNAL_LINK"
  | "MISSING_CANONICAL"
  | "NOINDEX"
  | "THIN_CONTENT"
  | "SLOW_RESPONSE"
  | "NO_STRUCTURED_DATA"
  | "REDIRECT_CHAIN"
  | "MIXED_CONTENT";

export const ISSUE_DEFS: Record<IssueCode, { severity: IssueSeverityKey; message: string }> = {
  MISSING_TITLE: { severity: "ERROR", message: "Halaman tidak memiliki tag <title>." },
  TITLE_TOO_LONG: { severity: "WARNING", message: "Title lebih dari 60 karakter dan berpotensi terpotong di hasil pencarian." },
  TITLE_TOO_SHORT: { severity: "WARNING", message: "Title kurang dari 30 karakter." },
  MISSING_META_DESCRIPTION: { severity: "WARNING", message: "Meta description tidak ditemukan." },
  META_DESCRIPTION_TOO_LONG: { severity: "NOTICE", message: "Meta description lebih dari 160 karakter." },
  MISSING_H1: { severity: "ERROR", message: "Halaman tidak memiliki heading H1." },
  MULTIPLE_H1: { severity: "WARNING", message: "Halaman memiliki lebih dari satu H1." },
  IMG_MISSING_ALT: { severity: "WARNING", message: "Gambar tanpa atribut alt." },
  BROKEN_INTERNAL_LINK: { severity: "ERROR", message: "Tautan internal mengarah ke halaman 404." },
  MISSING_CANONICAL: { severity: "NOTICE", message: "Tag canonical tidak ditemukan." },
  NOINDEX: { severity: "WARNING", message: "Halaman diberi robots noindex." },
  THIN_CONTENT: { severity: "WARNING", message: "Konten kurang dari 300 kata." },
  SLOW_RESPONSE: { severity: "WARNING", message: "Waktu respons server lebih dari 1,5 detik." },
  NO_STRUCTURED_DATA: { severity: "NOTICE", message: "Tidak ada structured data (JSON-LD)." },
  REDIRECT_CHAIN: { severity: "WARNING", message: "Rantai redirect lebih dari satu lompatan." },
  MIXED_CONTENT: { severity: "ERROR", message: "Aset dimuat melalui HTTP pada halaman HTTPS." },
};

export type CrawlIssue = {
  code: IssueCode;
  severity: IssueSeverityKey;
  message: string;
  /** page where the issue was found */
  url: string;
};

export type CrawlPageData = {
  url: string;
  statusCode: number;
  title: string | null;
  metaDescription: string | null;
  h1Count: number;
  wordCount: number;
  canonical: string | null;
  indexable: boolean;
  imagesTotal: number;
  imagesMissingAlt: number;
  internalLinks: number;
  externalLinks: number;
  loadMs: number | null;
  hasJsonLd: boolean;
  hasHreflang: boolean;
};

export type CrawlResult = {
  pages: CrawlPageData[];
  issues: CrawlIssue[];
  pagesCrawled: number;
  durationMs: number;
};

export const CRAWLER_UA = "WHITEAnalyticsBot/1.0 (+https://white.id/bot)";
export const CRAWLER_UA_TOKEN = "whiteanalyticsbot";
