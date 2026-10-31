import { describe, expect, it } from "vitest";
import { parseFigure } from "./count-up";

describe("parseFigure", () => {
  it.each([
    ["Rp 494.255.821", 494255821, 1000, "Rp 1.000"],
    ["12,5%", 12.5, 3.25, "3,3%"],
    ["$1.19M", 1.19, 0.5, "$0.50M"],
    ["1,2 jt", 1.2, 0.04, "0,0 jt"],
    ["-Rp 3.400", -3400, -1200, "-Rp 1.200"],
    ["36 hari", 36, 7, "7 hari"],
    ["1,250", 1250, 999999, "999,999"],
  ])("%s", (text, number, n, rendered) => {
    const f = parseFigure(text);
    expect(f?.number).toBe(number);
    expect(f?.render(n)).toBe(rendered);
    expect(f?.render(number)).toBe(text);
  });

  it("skips text without a number", () => {
    expect(parseFigure("-")).toBeNull();
  });
});
