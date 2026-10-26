import { describe, it, expect } from "vitest";
import {
  contentFiles,
  currentDue,
  formatBytes,
  isMonthKey,
  isOverdue,
  itemInputFrom,
  mediaKind,
  parseHashtags,
  shiftMonth,
  splitList,
} from "@/features/content/lib";
import type { Item } from "@/features/content/types";

const item = (over: Partial<Item> = {}): Item => ({
  id: "i1",
  planId: "p1",
  clientId: "c1",
  plannedDate: "2026-11-10",
  platform: "INSTAGRAM",
  format: "FEED",
  pillar: "Promo",
  title: "Judul",
  hook: "",
  caption: "",
  visualBrief: "",
  status: "COPYWRITING",
  revisionRound: 0,
  captionVersion: 1,
  hasFinalDesign: false,
  createdAt: "2026-10-01T00:00:00Z",
  updatedAt: "2026-10-01T00:00:00Z",
  copyDue: "2026-11-01",
  designDue: "2026-11-05",
  reviewDue: "2026-11-08",
  ...over,
});

describe("content deadlines", () => {
  it("picks the deadline of the current stage", () => {
    expect(currentDue(item({ status: "IDEA" }))).toEqual({ stage: "COPY", date: "2026-11-01" });
    expect(currentDue(item({ status: "DESIGN" }))).toEqual({ stage: "DESIGN", date: "2026-11-05" });
    expect(currentDue(item({ status: "REVISION" }))).toEqual({ stage: "REVIEW", date: "2026-11-08" });
    expect(currentDue(item({ status: "APPROVED" }))).toBeNull();
    expect(currentDue(item({ status: "DESIGN", designDue: undefined }))).toBeNull();
  });

  it("is overdue only after the due day", () => {
    expect(isOverdue(item(), "2026-11-01")).toBe(false);
    expect(isOverdue(item(), "2026-11-02")).toBe(true);
    expect(isOverdue(item({ status: "PUBLISHED" }), "2027-01-01")).toBe(false);
  });
});

describe("content parsing", () => {
  it("splits comma lists", () => {
    expect(splitList(" a, b ,, c ")).toEqual(["a", "b", "c"]);
  });

  it("normalizes hashtags", () => {
    expect(parseHashtags("#Kopi kopi, ##Promo\nbaru")).toEqual(["#Kopi", "#Promo", "#baru"]);
  });

  it("validates and shifts months", () => {
    expect(isMonthKey("2026-11")).toBe(true);
    expect(isMonthKey("2026-13")).toBe(false);
    expect(shiftMonth("2026-12", 1)).toBe("2027-01");
    expect(shiftMonth("2026-01", -1)).toBe("2025-12");
  });

  it("formats bytes and media kinds", () => {
    expect(formatBytes("1048")).toBe("1,0 KB");
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(5 * 1024 * 1024)).toBe("5,0 MB");
    expect(mediaKind("image/png")).toBe("image");
    expect(mediaKind("video/mp4")).toBe("video");
    expect(mediaKind("application/pdf")).toBe("pdf");
    expect(mediaKind(undefined)).toBe("other");
  });
});

describe("content API helpers", () => {
  it("builds the full planning input with empty optional fields", () => {
    const input = itemInputFrom(item({ copywriterId: undefined, designDue: undefined }));
    expect(input.copywriterId).toBe("");
    expect(input.designDue).toBe("");
    expect(input.format).toBe("FEED");
  });

  it("routes files through the proxy", () => {
    expect(contentFiles.asset("c1", "a1")).toBe("/api/erp/v1/content/clients/c1/assets/a1/download");
    expect(contentFiles.asset("c1", "a1", 2)).toBe("/api/erp/v1/content/clients/c1/assets/a1/download?version=2");
    expect(contentFiles.uploadDesign("c1", "t1")).toBe("/api/erp/api/content/clients/c1/design-tasks/t1/files");
  });
});
