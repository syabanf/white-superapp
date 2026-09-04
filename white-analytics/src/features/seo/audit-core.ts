/**
 * Auth-free core of the site audit (PageSpeed mobile + desktop, internal crawler) so the
 * server action and the scheduled daily job share one implementation.
 */
import "server-only";
import { db } from "@/lib/db";
import { isProviderError } from "@/lib/action-result";
import { runPagespeed, type PsiResult } from "@/lib/providers/pagespeed";
import { crawlSite } from "@/lib/providers/crawler";
import type { Prisma } from "@/generated/prisma/client";
import { s } from "./strings";

export const AUDIT_COOLDOWN_MS = 10 * 60 * 1000;

export function auditTarget(siteUrl: string, override?: string): string {
  if (override) return override;
  if (siteUrl.startsWith("sc-domain:")) return `https://${siteUrl.slice("sc-domain:".length)}/`;
  return siteUrl;
}

function reasonMessage(reason: unknown): string {
  if (isProviderError(reason)) return reason.message;
  if (reason instanceof Error) return reason.message;
  return String(reason);
}

export type AuditCoreResult = {
  slug: string;
  target: string;
  allFailed: boolean;
  /** human summary fragments, e.g. "PageSpeed 2/2 strategi" */
  parts: string[];
  errorParts: string[];
  audits: number;
  pagesCrawled: number;
  issues: number;
};

/** Run the audit for a property and persist everything (SeoAudit, SeoCrawl(+pages/issues), SyncJob). */
export async function runAuditCore(propertyId: string, url?: string): Promise<AuditCoreResult> {
  const property = await db.seoProperty.findUniqueOrThrow({ where: { id: propertyId }, include: { client: { select: { slug: true } } } });
  const target = auditTarget(property.siteUrl, url);

  const [job, crawlRecord] = await Promise.all([
    db.syncJob.create({ data: { clientId: property.clientId, kind: "SEO_AUDIT", status: "RUNNING", message: target } }),
    db.seoCrawl.create({ data: { propertyId: property.id, status: "RUNNING", maxPages: 30 } }),
  ]);

  const [psiMobile, psiDesktop, crawlOutcome] = await Promise.allSettled([
    runPagespeed(target, "mobile"),
    runPagespeed(target, "desktop"),
    crawlSite(target, { maxPages: 30, timeBudgetMs: 45_000 }),
  ]);

  const psiResults: PsiResult[] = [];
  const psiErrors: string[] = [];
  for (const settled of [psiMobile, psiDesktop]) {
    if (settled.status === "fulfilled") psiResults.push(settled.value);
    else psiErrors.push(reasonMessage(settled.reason));
  }
  for (const psi of psiResults) {
    await db.seoAudit.create({
      data: {
        propertyId: property.id,
        url: target,
        strategy: psi.strategy === "mobile" ? "MOBILE" : "DESKTOP",
        performance: psi.scores.performance,
        seo: psi.scores.seo,
        accessibility: psi.scores.accessibility,
        bestPractices: psi.scores.bestPractices,
        lcpMs: psi.metrics.lcpMs,
        inpMs: psi.metrics.inpMs,
        cls: psi.metrics.cls,
        fcpMs: psi.metrics.fcpMs,
        ttfbMs: psi.metrics.ttfbMs,
        tbtMs: psi.metrics.tbtMs,
        speedIndexMs: psi.metrics.speedIndexMs,
        raw: psi.raw as Prisma.InputJsonValue,
      },
    });
  }

  let pagesCrawled = 0;
  let issueCount = 0;
  let crawlError: string | null = null;
  if (crawlOutcome.status === "fulfilled") {
    const res = crawlOutcome.value;
    pagesCrawled = res.pagesCrawled;
    issueCount = res.issues.length;
    await db.seoCrawlPage.createMany({
      data: res.pages.map((p) => ({
        crawlId: crawlRecord.id,
        url: p.url,
        statusCode: p.statusCode,
        title: p.title,
        metaDescription: p.metaDescription,
        h1Count: p.h1Count,
        wordCount: p.wordCount,
        canonical: p.canonical,
        indexable: p.indexable,
        imagesTotal: p.imagesTotal,
        imagesMissingAlt: p.imagesMissingAlt,
        internalLinks: p.internalLinks,
        externalLinks: p.externalLinks,
        loadMs: p.loadMs,
        hasJsonLd: p.hasJsonLd,
        hasHreflang: p.hasHreflang,
      })),
    });
    if (res.issues.length > 0) {
      await db.seoIssue.createMany({
        data: res.issues.map((i) => ({ crawlId: crawlRecord.id, severity: i.severity, code: i.code, message: i.message, url: i.url })),
      });
    }
    await db.seoCrawl.update({ where: { id: crawlRecord.id }, data: { status: "SUCCESS", finishedAt: new Date(), pagesCrawled: res.pagesCrawled } });
  } else {
    crawlError = reasonMessage(crawlOutcome.reason);
    await db.seoCrawl.update({ where: { id: crawlRecord.id }, data: { status: "FAILED", finishedAt: new Date(), error: crawlError } });
  }

  const allFailed = psiResults.length === 0 && crawlOutcome.status === "rejected";
  const parts: string[] = [];
  if (psiResults.length > 0) parts.push(`${s.psiLabel} ${psiResults.length}/2 strategi`);
  if (crawlOutcome.status === "fulfilled") parts.push(`${s.crawlLabel} ${pagesCrawled} halaman, ${issueCount} isu`);
  const errorParts = [...psiErrors, ...(crawlError ? [crawlError] : [])];

  await db.syncJob.update({
    where: { id: job.id },
    data: {
      status: allFailed ? "FAILED" : "SUCCESS",
      finishedAt: new Date(),
      message: parts.join(" · ") || null,
      error: errorParts.length ? errorParts.join(" | ").slice(0, 900) : null,
    },
  });

  return { slug: property.client.slug, target, allFailed, parts, errorParts, audits: psiResults.length, pagesCrawled, issues: issueCount };
}
