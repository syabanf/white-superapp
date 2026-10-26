import { describe, expect, it } from "vitest";
import { filterPeople } from "./people-filter";
import type { Department } from "./types";

const depts = [
  { id: "d1", name: "Creative" },
  { id: "d2", name: "Finance" },
] as Department[];
const rows = [
  { id: "a", employee: { id: "e1", departmentName: "Creative" } },
  { id: "b", employee: { id: "e2", departmentName: "Finance" } },
  { id: "c", employee: { id: "e3", departmentName: "Creative" } },
];

describe("filterPeople", () => {
  it("keeps every row without filters", () => {
    expect(filterPeople(rows, depts, {})).toHaveLength(3);
  });
  it("matches the department by id through its name", () => {
    expect(filterPeople(rows, depts, { dept: "d1" }).map((r) => r.id)).toEqual(["a", "c"]);
  });
  it("combines department and employee", () => {
    expect(filterPeople(rows, depts, { dept: "d1", employee: "e3" }).map((r) => r.id)).toEqual(["c"]);
  });
  it("returns nothing for an unknown department", () => {
    expect(filterPeople(rows, depts, { dept: "nope" })).toEqual([]);
  });
});
