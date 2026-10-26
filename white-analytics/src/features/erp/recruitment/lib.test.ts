import { describe, expect, it } from "vitest";
import { averageScore, formatCountdown, splitLines, toRupiahInt, wibInstant, wibParts } from "./lib";

describe("recruitment helpers", () => {
  it("formats a countdown", () => {
    expect(formatCountdown(125)).toBe("02:05");
    expect(formatCountdown(-3)).toBe("00:00");
  });
  it("turns money strings into whole rupiah", () => {
    expect(toRupiahInt("9500000.00")).toBe("9500000");
    expect(toRupiahInt("")).toBeUndefined();
    expect(toRupiahInt("-1")).toBeUndefined();
  });
  it("round-trips WIB day and time", () => {
    const iso = wibInstant("2026-10-26", "09:30");
    expect(iso).toBe("2026-10-26T02:30:00.000Z");
    expect(wibParts(iso)).toEqual({ day: "2026-10-26", time: "09:30" });
    expect(wibInstant("2026-10-26", "")).toBeNull();
  });
  it("averages scorecard scores", () => {
    expect(averageScore([3, 4, 4])).toBe(3.7);
    expect(averageScore([])).toBeNull();
  });
  it("splits benefit lines", () => {
    expect(splitLines(" BPJS \n\nLaptop ")).toEqual(["BPJS", "Laptop"]);
  });
});
