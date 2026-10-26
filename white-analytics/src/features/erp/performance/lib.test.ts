import { describe, expect, it } from "vitest";
import { groupBySection, parseQuickCapture, quickAction, scheduleFor } from "./lib";
import type { TaskItem } from "./types";

const TODAY = "2026-10-26"; // Monday

describe("parseQuickCapture", () => {
  it("strips trailing markers into date and priority", () => {
    expect(parseQuickCapture("Kirim laporan besok !!", TODAY)).toEqual({
      title: "Kirim laporan",
      dueDate: "2026-10-27",
      recurrence: "once",
      priority: "urgent",
    });
  });

  it("reads a day name as the next such day and a cadence", () => {
    expect(parseQuickCapture("Rapat tim jumat", TODAY)?.dueDate).toBe("2026-10-30");
    expect(parseQuickCapture("Cek inbox setiap hari", TODAY)?.recurrence).toBe("daily");
  });

  it("keeps a lone marker word as the title", () => {
    expect(parseQuickCapture("besok", TODAY)?.title).toBe("besok");
    expect(parseQuickCapture("   ", TODAY)).toBeNull();
  });
});

describe("scheduleFor", () => {
  it("maps recurrence to the create request fields", () => {
    expect(scheduleFor("once", "2026-10-30")).toEqual({ dueDate: "2026-10-30" });
    expect(scheduleFor("weekly", "2026-11-01")).toEqual({ weeklyDay: 7 });
    expect(scheduleFor("monthly", "2026-10-31")).toEqual({ monthlyDay: 28 });
  });
});

const item = (p: Partial<TaskItem>): TaskItem =>
  ({
    occurrenceId: Math.random().toString(36),
    status: "pending",
    occurrenceDate: TODAY,
    subtaskCount: 0,
    canWork: true,
    canReview: false,
    ...p,
  }) as TaskItem;

describe("groupBySection", () => {
  it("puts late open rows first and keeps review and approved apart", () => {
    const g = groupBySection(
      [
        item({ occurrenceDate: "2026-10-20" }),
        item({}),
        item({ occurrenceDate: "2026-10-28" }),
        item({ status: "done", occurrenceDate: "2026-10-20" }),
        item({ status: "approved" }),
      ],
      TODAY,
    );
    expect([g.overdue.length, g.today.length, g.upcoming.length, g.review.length, g.closed.length]).toEqual([1, 1, 1, 1, 1]);
  });
});

describe("quickAction", () => {
  it("offers the next step the user may take", () => {
    expect(quickAction(item({}))).toBe("start");
    expect(quickAction(item({ status: "in_progress" }))).toBe("done");
    expect(quickAction(item({ status: "in_progress", subtaskCount: 2 }))).toBeNull();
    expect(quickAction(item({ status: "done", canReview: true }))).toBe("approve");
  });
});
