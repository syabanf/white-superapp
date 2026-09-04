import { describe, it, expect } from "vitest";
import {
  WIZARD_STEPS,
  nextStep,
  parseTypes,
  prevStep,
  resolveStep,
  serializeTypes,
  sourceSectionsFor,
  stepNumber,
  websiteRequiredFor,
  wizardHref,
} from "@/features/clients/wizard";

describe("wizard steps", () => {
  it("has a stable order, starting from the project type", () => {
    expect(WIZARD_STEPS).toEqual(["jenis", "profil", "sumber", "akses", "selesai"]);
    expect(stepNumber("sumber")).toBe(3);
  });

  it("walks forward and back, stopping at the ends", () => {
    expect(nextStep("jenis")).toBe("profil");
    expect(nextStep("selesai")).toBeNull();
    expect(prevStep("profil")).toBe("jenis");
    expect(prevStep("jenis")).toBeNull();
  });

  it("parses and serializes types from the URL, dropping junk", () => {
    expect(parseTypes("SEO,ADS")).toEqual(["SEO", "ADS"]);
    expect(parseTypes("ads,seo")).toEqual(["SEO", "ADS"]);
    expect(parseTypes("SEO,SEO,BOGUS")).toEqual(["SEO"]);
    expect(parseTypes(undefined)).toEqual([]);
    expect(serializeTypes(["ADS", "SEO"])).toBe("SEO,ADS");
  });

  it("cannot reach the profile step without picking a type", () => {
    expect(resolveStep("profil", { hasClient: false, hasTypes: false })).toBe("jenis");
    expect(resolveStep("profil", { hasClient: false, hasTypes: true })).toBe("profil");
  });

  it("cannot reach client-scoped steps before the client exists", () => {
    expect(resolveStep("sumber", { hasClient: false, hasTypes: true })).toBe("profil");
    expect(resolveStep("akses", { hasClient: false, hasTypes: false })).toBe("jenis");
  });

  it("skips settled steps once the client exists", () => {
    expect(resolveStep("jenis", { hasClient: true, hasTypes: true })).toBe("sumber");
    expect(resolveStep("profil", { hasClient: true, hasTypes: true })).toBe("sumber");
    expect(resolveStep("akses", { hasClient: true, hasTypes: true })).toBe("akses");
  });

  it("falls back on junk input", () => {
    expect(resolveStep("tidak-ada", { hasClient: true, hasTypes: true })).toBe("sumber");
    expect(resolveStep(undefined, { hasClient: false, hasTypes: false })).toBe("jenis");
  });

  it("builds resumable URLs carrying type and client", () => {
    expect(wizardHref("profil", { types: ["SEO"] })).toBe("/clients/new?step=profil&types=SEO");
    expect(wizardHref("sumber", { slug: "kopi-nusantara" })).toBe("/clients/new?step=sumber&client=kopi-nusantara");
    expect(wizardHref("jenis")).toBe("/clients/new?step=jenis");
  });

  it("shows only the source sections for the chosen types", () => {
    expect(sourceSectionsFor(["SOCIAL"])).toEqual({ social: true, seo: false, ads: false });
    expect(sourceSectionsFor(["SEO", "ADS"])).toEqual({ social: false, seo: true, ads: true });
    expect(sourceSectionsFor([])).toEqual({ social: false, seo: false, ads: false });
  });

  it("requires a website only for SEO projects", () => {
    expect(websiteRequiredFor(["SEO"])).toBe(true);
    expect(websiteRequiredFor(["SOCIAL", "ADS"])).toBe(false);
  });
});
