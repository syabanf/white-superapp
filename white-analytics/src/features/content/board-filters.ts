/**
 * Board filters shared by the board page (server) and the dashboard drill
 * sheet (client): parse from the URL, match an item, describe as chips and
 * serialise back. Pure. Dates are WIB "YYYY-MM-DD" (the agency time zone the
 * dashboard buckets use).
 */
import { platformLabel } from "@/features/social/components/platform-icon";
import { dayLabel, weekdayLabel } from "@/features/publishing/time";
import { FORMATS, ITEM_STATUSES, PLATFORMS, isOverdue } from "./lib";
import { c, cbf } from "./strings";
import type { ContentFormat, Item, ItemStatus, Platform } from "./types";

export type BoardStage = "COPY" | "DESIGN" | "REVIEW" | "PUBLISH";
export const BOARD_STAGES: BoardStage[] = ["COPY", "DESIGN", "REVIEW", "PUBLISH"];

/** Current statuses that belong to each production stage (PUBLISH = ready to go out). */
export const STAGE_STATUSES: Record<BoardStage, ItemStatus[]> = {
  COPY: ["IDEA", "COPYWRITING"],
  DESIGN: ["DESIGN"],
  REVIEW: ["REVIEW", "REVISION"],
  PUBLISH: ["APPROVED", "SCHEDULED"],
};

// happy path of the dashboard funnel; REVISION counts as REVIEW, FAILED as SCHEDULED
const FUNNEL: ItemStatus[] = [
  "IDEA",
  "COPYWRITING",
  "DESIGN",
  "REVIEW",
  "APPROVED",
  "SCHEDULED",
  "PUBLISHED",
];
function funnelRank(s: ItemStatus): number {
  const n = s === "REVISION" ? "REVIEW" : s === "FAILED" ? "SCHEDULED" : s;
  return Math.max(0, FUNNEL.indexOf(n));
}

export type BoardFilters = {
  status: ItemStatus[];
  stage: BoardStage | "";
  /** items that got at least this far in the funnel */
  reached: ItemStatus | "";
  dateFrom: string;
  dateTo: string;
  /** which date the range applies to */
  dateBy: "planned" | "published";
  overdue: boolean;
  /** exact revision rounds ("0", "1", "2", "3+") or any revision ("any") */
  rounds: string;
  platform: Platform | "";
  format: ContentFormat | "";
  pillar: string;
  /** published weekday (0 = Monday) and hour, WIB */
  weekday: number | null;
  hour: number | null;
};

/** URL keys the board reads besides `plan` and `assignee` (which go to the API). */
export const BOARD_FILTER_KEYS = [
  "status",
  "stage",
  "reached",
  "dateFrom",
  "dateTo",
  "dateBy",
  "overdue",
  "rounds",
  "platform",
  "format",
  "pillar",
  "weekday",
  "hour",
] as const;

type SP = Record<string, string | string[] | undefined> | URLSearchParams;
const ISO = /^\d{4}-\d{2}-\d{2}$/;

function getter(sp: SP) {
  return (k: string): string => {
    if (sp instanceof URLSearchParams) return sp.get(k) ?? "";
    const v = sp[k];
    return (Array.isArray(v) ? v[0] : v) ?? "";
  };
}

function intIn(raw: string, min: number, max: number): number | null {
  if (!/^\d+$/.test(raw)) return null;
  const n = Number(raw);
  return n >= min && n <= max ? n : null;
}

export function parseBoardFilters(sp: SP): BoardFilters {
  const get = getter(sp);
  const status = get("status")
    .split(",")
    .filter((s): s is ItemStatus => ITEM_STATUSES.includes(s as ItemStatus));
  const stage = get("stage") as BoardStage;
  const reached = get("reached") as ItemStatus;
  const platform = get("platform") as Platform;
  const format = get("format") as ContentFormat;
  const dateFrom = get("dateFrom");
  const dateTo = get("dateTo");
  const rounds = get("rounds");
  return {
    status,
    stage: BOARD_STAGES.includes(stage) ? stage : "",
    reached: ITEM_STATUSES.includes(reached) ? reached : "",
    dateFrom: ISO.test(dateFrom) ? dateFrom : "",
    dateTo: ISO.test(dateTo) ? dateTo : "",
    dateBy: get("dateBy") === "published" ? "published" : "planned",
    overdue: get("overdue") === "1",
    rounds: ["0", "1", "2", "3+", "any"].includes(rounds) ? rounds : "",
    platform: PLATFORMS.includes(platform) ? platform : "",
    format: FORMATS.includes(format) ? format : "",
    pillar: get("pillar").slice(0, 80),
    weekday: intIn(get("weekday"), 0, 6),
    hour: intIn(get("hour"), 0, 23),
  };
}

/** True when any filter narrows the board beyond plan and assignee. */
export function hasBoardFilters(f: BoardFilters): boolean {
  return (
    f.status.length > 0 ||
    !!(f.stage || f.reached || f.dateFrom || f.dateTo || f.rounds || f.platform || f.format || f.pillar) ||
    f.overdue ||
    f.weekday != null ||
    f.hour != null
  );
}

const wibParts = new Intl.DateTimeFormat("en-CA", {
  timeZone: "Asia/Jakarta",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  hourCycle: "h23",
  weekday: "short",
});
const WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

/** WIB day, weekday (0 = Monday) and hour of an instant. */
export function wibMoment(iso: string): { day: string; weekday: number; hour: number } {
  const p = Object.fromEntries(wibParts.formatToParts(new Date(iso)).map((x) => [x.type, x.value]));
  return {
    day: `${p.year}-${p.month}-${p.day}`,
    weekday: WEEKDAYS.indexOf(p.weekday!),
    hour: Number(p.hour),
  };
}

type MatchItem = Pick<
  Item,
  | "status"
  | "plannedDate"
  | "publishedAt"
  | "platform"
  | "format"
  | "pillar"
  | "revisionRound"
  | "copyDue"
  | "designDue"
  | "reviewDue"
>;

export function matchItem(it: MatchItem, f: BoardFilters, today: string): boolean {
  if (f.status.length && !f.status.includes(it.status)) return false;
  if (f.stage && !STAGE_STATUSES[f.stage].includes(it.status)) return false;
  if (f.reached && funnelRank(it.status) < funnelRank(f.reached)) return false;
  if (f.overdue && !isOverdue(it, today)) return false;
  if (f.platform && it.platform !== f.platform) return false;
  if (f.format && it.format !== f.format) return false;
  if (f.pillar && it.pillar !== f.pillar) return false;
  if (f.rounds) {
    const r = it.revisionRound;
    if (f.rounds === "any" ? r < 1 : f.rounds === "3+" ? r < 3 : r !== Number(f.rounds)) return false;
  }
  const published = it.status === "PUBLISHED" && it.publishedAt ? wibMoment(it.publishedAt) : null;
  if (f.weekday != null || f.hour != null) {
    if (!published) return false;
    if (f.weekday != null && published.weekday !== f.weekday) return false;
    if (f.hour != null && published.hour !== f.hour) return false;
  }
  if (f.dateFrom || f.dateTo) {
    const day = f.dateBy === "published" ? published?.day : it.plannedDate;
    if (!day) return false;
    if (f.dateFrom && day < f.dateFrom) return false;
    if (f.dateTo && day > f.dateTo) return false;
  }
  return true;
}

/** Filter patch to a query string (empty values dropped), stable key order. */
export function boardFilterQuery(f: Partial<BoardFilters>): URLSearchParams {
  const q = new URLSearchParams();
  if (f.status?.length) q.set("status", f.status.join(","));
  if (f.stage) q.set("stage", f.stage);
  if (f.reached) q.set("reached", f.reached);
  if (f.dateFrom) q.set("dateFrom", f.dateFrom);
  if (f.dateTo) q.set("dateTo", f.dateTo);
  if (f.dateBy === "published" && (f.dateFrom || f.dateTo)) q.set("dateBy", "published");
  if (f.overdue) q.set("overdue", "1");
  if (f.rounds) q.set("rounds", f.rounds);
  if (f.platform) q.set("platform", f.platform);
  if (f.format) q.set("format", f.format);
  if (f.pillar) q.set("pillar", f.pillar);
  if (f.weekday != null) q.set("weekday", String(f.weekday));
  if (f.hour != null) q.set("hour", String(f.hour));
  return q;
}

export type FilterChip = { key: (typeof BOARD_FILTER_KEYS)[number] | "date"; label: string };

/** Human chips for the active filters ("Status: Terbit", "1 Okt – 7 Okt", "Instagram"). */
export function describeBoardFilters(f: BoardFilters): FilterChip[] {
  const chips: FilterChip[] = [];
  if (f.status.length)
    chips.push({ key: "status", label: `${cbf.status}: ${f.status.map((s) => c.itemStatus[s]).join(", ")}` });
  if (f.stage) chips.push({ key: "stage", label: `${cbf.stage}: ${cbf.stages[f.stage]}` });
  if (f.reached) chips.push({ key: "reached", label: cbf.reached(c.itemStatus[f.reached]) });
  if (f.dateFrom || f.dateTo) {
    const by = f.dateBy === "published" ? cbf.dateByPublished : cbf.dateByPlanned;
    const range =
      f.dateFrom && f.dateTo
        ? f.dateFrom === f.dateTo
          ? dayLabel(f.dateFrom, true)
          : `${dayLabel(f.dateFrom, true)} – ${dayLabel(f.dateTo, true)}`
        : f.dateFrom
          ? `≥ ${dayLabel(f.dateFrom, true)}`
          : `≤ ${dayLabel(f.dateTo, true)}`;
    chips.push({ key: "date", label: `${by}: ${range}` });
  }
  if (f.overdue) chips.push({ key: "overdue", label: cbf.overdue });
  if (f.rounds) chips.push({ key: "rounds", label: f.rounds === "any" ? cbf.revised : cbf.rounds(f.rounds) });
  if (f.platform) chips.push({ key: "platform", label: platformLabel(f.platform) });
  if (f.format) chips.push({ key: "format", label: c.format[f.format] });
  if (f.pillar) chips.push({ key: "pillar", label: f.pillar });
  if (f.weekday != null || f.hour != null) {
    const day = f.weekday != null ? weekdayLabel(f.weekday, true) : "";
    const hour = f.hour != null ? `${String(f.hour).padStart(2, "0")}.00` : "";
    chips.push({
      key: f.weekday != null ? "weekday" : "hour",
      label: day && hour ? cbf.weekdayHour(day, hour) : day || hour,
    });
  }
  return chips;
}

/** URL params a chip removes. */
export function chipParams(key: FilterChip["key"]): string[] {
  if (key === "date") return ["dateFrom", "dateTo", "dateBy"];
  if (key === "weekday" || key === "hour") return ["weekday", "hour"];
  return [key];
}
