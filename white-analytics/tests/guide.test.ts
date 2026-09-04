import { describe, it, expect } from "vitest";
import { GUIDE_SECTIONS, getGuideSection, guideSectionForPath } from "@/features/guide/content";

describe("in-app guide", () => {
  it("has unique, non-empty sections", () => {
    const ids = GUIDE_SECTIONS.map((s) => s.id);
    expect(new Set(ids).size).toBe(ids.length);
    for (const s of GUIDE_SECTIONS) {
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
      "/panduan",
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
    expect(guideSectionForPath("/clients/x/ads")).toBe("ads");
    expect(guideSectionForPath("/clients/x/reports")).toBe("laporan");
    expect(guideSectionForPath("/clients/new")).toBe("mulai");
    expect(guideSectionForPath("/")).toBe("mulai");
    expect(guideSectionForPath("/setup?step=meta")).toBe("integrasi");
  });
});
