import { describe, expect, it } from "vitest";
import { MARKETPLACE_BRANDS, TEAM, brandSales, costPerThousand, designStatus, leadQuality, qualityScore } from "@/features/showcase/data";

describe("showcase mock helpers", () => {
  it("grades leads by score", () => {
    expect(leadQuality(70)).toBe("qualified");
    expect(leadQuality(69)).toBe("review");
    expect(leadQuality(40)).toBe("review");
    expect(leadQuality(39)).toBe("notTarget");
  });

  it("flags design orders by approvals, deadline and pace", () => {
    expect(designStatus({ target: 12, uploaded: 12, approved: 12, daysLeft: -3 })).toBe("done");
    expect(designStatus({ target: 4, uploaded: 3, approved: 3, daysLeft: -1 })).toBe("late");
    expect(designStatus({ target: 10, uploaded: 2, approved: 1, daysLeft: 2 })).toBe("atRisk");
    expect(designStatus({ target: 8, uploaded: 3, approved: 2, daysLeft: 5 })).toBe("onTrack");
  });

  it("sums marketplace sales across markets", () => {
    const own = MARKETPLACE_BRANDS.find((b) => b.own)!;
    expect(brandSales(own, "all")).toEqual({ units: 1240, revenue: 110_360_000 });
    expect(brandSales(own, "tiktok").units).toBe(340);
  });

  it("ranks quality above punctuality", () => {
    const bima = TEAM.find((m) => m.name === "Bima")!;
    const citra = TEAM.find((m) => m.name === "Citra")!;
    expect(bima.onTime).toBeGreaterThan(citra.onTime);
    expect(qualityScore(citra)).toBeGreaterThan(qualityScore(bima));
  });

  it("computes cost per thousand reached", () => {
    expect(costPerThousand({ cost: 1_500_000, reach: 30_000 })).toBe(50_000);
  });
});
