import { parseAsInteger, parseAsString } from "nuqs/server";
import { withQuery } from "@/lib/drill-query";

/** URL filters for the social posts list; drills from tiles and charts land here. */
export const postFilterParsers = {
  type: parseAsString,
  day: parseAsInteger,
  hour: parseAsInteger,
  sort: parseAsString,
};

export const POST_SORTS = ["publishedAt", "er", "reach", "views", "engagements"] as const;
export type PostSort = (typeof POST_SORTS)[number];

export type PostFilter = { type?: string | null; day?: number | null; hour?: number | null; date?: string | null };

type Filterable = { type: string; day: number; hour: number; publishedAt: string };

/** `date` is a Jakarta calendar day (YYYY-MM-DD). */
export function filterPosts<T extends Filterable>(rows: T[], f: PostFilter): T[] {
  return rows.filter(
    (r) =>
      (f.type == null || r.type === f.type) &&
      (f.day == null || r.day === f.day) &&
      (f.hour == null || r.hour === f.hour) &&
      (f.date == null || jakartaDate(r.publishedAt) === f.date),
  );
}

const jakartaDay = new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" });

function jakartaDate(iso: string): string {
  return jakartaDay.format(new Date(iso));
}

/** The posts list section of the social page with the given filters. */
export function postsListHref(base: string, f: PostFilter & { sort?: PostSort }, keep = ""): string {
  return withQuery(base, keep, { type: f.type, day: f.day, hour: f.hour, sort: f.sort }, "posts");
}
