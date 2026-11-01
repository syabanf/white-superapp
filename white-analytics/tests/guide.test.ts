import { describe, it, expect } from "vitest";
import { GUIDE_SECTIONS, getGuideSection, guideHref, guideSectionForPath, guideTrackForPath } from "@/features/guide/content";
import { ERP_GUIDE_SECTIONS } from "@/features/erp/guide";

// Every ERP page route, with sample ids in place of dynamic segments.
const ERP_PATHS = [
  "/erp",
  "/erp/finance",
  "/erp/finance/journals/new",
  "/erp/finance/reimbursements/abc",
  "/erp/hr/employees/abc/edit",
  "/erp/hr/payroll/runs/abc",
  "/erp/hr/recruitment/candidates/abc",
  "/erp/me/payslips",
  "/erp/me/reimbursements",
  "/erp/performance/kpi",
  "/erp/sales/quotations/new",
  "/erp/sales/invoices/abc",
  "/erp/sales/chatbot",
];

describe("in-app guide", () => {
  it("has unique, non-empty sections", () => {
    const ids = [...GUIDE_SECTIONS, ...ERP_GUIDE_SECTIONS].map((s) => s.id);
    expect(new Set(ids).size).toBe(ids.length);
    for (const s of [...GUIDE_SECTIONS, ...ERP_GUIDE_SECTIONS]) {
      expect(s.title.length).toBeGreaterThan(0);
      expect(s.summary.length).toBeGreaterThan(0);
      expect(s.blocks.length).toBeGreaterThan(0);
    }
  });

  it("maps every page to a section that actually exists", () => {
    const paths = [
      "/",
      "/clients",
      "/clients/new",
      "/clients/kopi-nusantara",
      "/clients/kopi-nusantara/social",
      "/clients/kopi-nusantara/social/competitors",
      "/clients/kopi-nusantara/publish",
      "/clients/kopi-nusantara/publish/posts",
      "/clients/kopi-nusantara/publish/new",
      "/clients/kopi-nusantara/publish/media",
      "/clients/kopi-nusantara/seo",
      "/clients/kopi-nusantara/seo/keywords",
      "/clients/kopi-nusantara/seo/audit",
      "/clients/kopi-nusantara/seo/research",
      "/clients/kopi-nusantara/seo/rank",
      "/clients/kopi-nusantara/seo/backlinks",
      "/clients/kopi-nusantara/seo/competitors",
      "/clients/kopi-nusantara/ads",
      "/clients/kopi-nusantara/reports",
      "/clients/kopi-nusantara/settings",
      "/admin/users",
      "/setup",
      "/clients/kopi-nusantara/content/plans",
      "/clients/kopi-nusantara/content/board",
      "/panduan",
      "/panduan/erp",
      ...ERP_PATHS,
    ];
    for (const p of paths) {
      const id = guideSectionForPath(p);
      expect(getGuideSection(id), `no guide section for ${p}`).toBeDefined();
    }
  });

  it("routes module pages to their own section", () => {
    expect(guideSectionForPath("/clients/x/social")).toBe("social");
    expect(guideSectionForPath("/clients/x/seo/audit")).toBe("seo");
    expect(guideSectionForPath("/clients/x/seo/rank")).toBe("seo-suite");
    expect(guideSectionForPath("/clients/x/publish/posts/abc")).toBe("publikasi");
    expect(guideSectionForPath("/clients/x/seo/research")).toBe("seo-suite");
    expect(guideSectionForPath("/clients/x/content/brand")).toBe("konten-ai");
    expect(guideSectionForPath("/clients/x/content/studio")).toBe("konten-ai");
    expect(guideSectionForPath("/clients/x/ads/campaigns/new")).toBe("ads");
    expect(guideSectionForPath("/clients/x/ads")).toBe("ads");
    expect(guideSectionForPath("/clients/x/reports")).toBe("laporan");
    expect(guideSectionForPath("/clients/new")).toBe("mulai");
    expect(guideSectionForPath("/")).toBe("mulai");
    expect(guideSectionForPath("/setup?step=meta")).toBe("integrasi");
  });

  it("sends ERP pages to the ERP tutorial", () => {
    for (const p of ERP_PATHS) {
      expect(guideTrackForPath(p)).toBe("erp");
      expect(guideHref(guideSectionForPath(p))).toMatch(/^\/panduan\/erp#erp-/);
    }
    expect(guideTrackForPath("/panduan/erp")).toBe("erp");
    expect(guideTrackForPath("/clients/x/ads")).toBe("marketing");
    expect(guideTrackForPath("/erpnext")).toBe("marketing");
    expect(guideSectionForPath("/clients/x/content/board")).toBe("konten");
    expect(guideHref("ads")).toBe("/panduan#ads");
  });
});
