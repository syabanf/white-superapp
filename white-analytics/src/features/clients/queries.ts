import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { addDays, type DateRange } from "@/lib/dates";
import { computeDelta, type Delta } from "@/lib/metrics";
import { requireUser } from "@/lib/rbac";

// ── Portfolio ────────────────────────────────────────────────

export type PortfolioTotals = {
  followers: number;
  /** growth this period vs growth previous period (same semantic as overview) */
  followersDelta: Delta;
  clicks: number;
  clicksDelta: Delta;
  spend: number;
  spendDelta: Delta;
};

/** Aggregated KPI strip across a set of clients (honest deltas, no reconstruction). */
export const getPortfolioTotals = cache(async (clientIds: string[], range: DateRange, previous: DateRange): Promise<PortfolioTotals> => {
  const zero = computeDelta(0, 0);
  if (clientIds.length === 0) {
    return { followers: 0, followersDelta: zero, clicks: 0, clicksDelta: zero, spend: 0, spendDelta: zero };
  }
  const accounts = await db.socialAccount.findMany({
    where: { clientId: { in: clientIds }, isCompetitor: false },
    select: { id: true },
  });
  const accountIds = accounts.map((a) => a.id);

  const [snapshots, clicksCur, clicksPrev, spendCur, spendPrev] = await Promise.all([
    accountIds.length
      ? db.socialSnapshot.findMany({
          where: { socialAccountId: { in: accountIds }, date: { gte: addDays(previous.from, -1), lte: range.to } },
          orderBy: { date: "asc" },
          select: { socialAccountId: true, date: true, followers: true },
        })
      : Promise.resolve([]),
    db.seoDailyMetric.aggregate({
      _sum: { clicks: true },
      where: { property: { clientId: { in: clientIds } }, date: { gte: range.from, lte: range.to } },
    }),
    db.seoDailyMetric.aggregate({
      _sum: { clicks: true },
      where: { property: { clientId: { in: clientIds } }, date: { gte: previous.from, lte: previous.to } },
    }),
    db.adDailyInsight.aggregate({
      _sum: { spend: true },
      where: { adAccount: { clientId: { in: clientIds } }, adId: { not: null }, date: { gte: range.from, lte: range.to } },
    }),
    db.adDailyInsight.aggregate({
      _sum: { spend: true },
      where: { adAccount: { clientId: { in: clientIds } }, adId: { not: null }, date: { gte: previous.from, lte: previous.to } },
    }),
  ]);

  // total followers at a date = latest snapshot on/before that date per account
  const followersAt = (d: Date) => {
    const perAccount = new Map<string, number>();
    for (const s of snapshots) if (s.date <= d) perAccount.set(s.socialAccountId, s.followers);
    let sum = 0;
    for (const v of perAccount.values()) sum += v;
    return sum;
  };
  const now = followersAt(range.to);
  const start = followersAt(addDays(range.from, -1));
  const prevStart = followersAt(addDays(previous.from, -1));

  const clicks = clicksCur._sum.clicks ?? 0;
  const spend = spendCur._sum.spend ?? 0;
  return {
    followers: now,
    followersDelta: computeDelta(now - start, start - prevStart),
    clicks,
    clicksDelta: computeDelta(clicks, clicksPrev._sum.clicks ?? 0),
    spend,
    spendDelta: computeDelta(spend, spendPrev._sum.spend ?? 0),
  };
});

/** Latest SyncJob start time per client (for "sinkron terakhir" on cards). */
export const getLatestSyncByClient = cache(async (clientIds: string[]): Promise<Record<string, string>> => {
  if (clientIds.length === 0) return {};
  const rows = await db.syncJob.groupBy({
    by: ["clientId"],
    where: { clientId: { in: clientIds } },
    _max: { startedAt: true },
  });
  const out: Record<string, string> = {};
  for (const r of rows) {
    if (r.clientId && r._max.startedAt) out[r.clientId] = r._max.startedAt.toISOString();
  }
  return out;
});

// ── Klien list ───────────────────────────────────────────────

export type ClientRow = {
  id: string;
  name: string;
  slug: string;
  description: string;
  industry: string | null;
  websiteUrl: string | null;
  logoUrl: string | null;
  currency: string;
  timezone: string;
  createdAt: string;
  membersCount: number;
  /** the current user's effective role for this client */
  myRole: "ADMIN" | "MANAGER" | "VIEWER";
};

export const listClientsWithMeta = cache(async (): Promise<ClientRow[]> => {
  const user = await requireUser();
  const where = user.role === "ADMIN" ? {} : { members: { some: { userId: user.id } } };
  const clients = await db.client.findMany({
    where,
    orderBy: { name: "asc" },
    include: {
      _count: { select: { members: true } },
      members: { where: { userId: user.id }, select: { role: true } },
    },
  });
  return clients.map((c) => ({
    id: c.id,
    name: c.name,
    slug: c.slug,
    description: c.description,
    industry: c.industry,
    websiteUrl: c.websiteUrl,
    logoUrl: c.logoUrl,
    currency: c.currency,
    timezone: c.timezone,
    createdAt: c.createdAt.toISOString(),
    membersCount: c._count.members,
    myRole: user.role === "ADMIN" ? "ADMIN" : (c.members[0]?.role ?? "VIEWER"),
  }));
});

// ── Pengaturan klien ─────────────────────────────────────────

export type SettingsConnection = {
  id: string;
  provider: "META" | "GOOGLE";
  accountId: string;
  displayName: string;
  expiresAt: string | null;
  scopes: string[];
  createdAt: string;
};

export type SettingsData = {
  connections: SettingsConnection[];
  socialAccounts: { id: string; platform: string; username: string; displayName: string }[];
  seoProperties: { id: string; siteUrl: string; ga4PropertyId: string | null }[];
  adAccounts: { id: string; externalId: string; name: string; currency: string }[];
  members: { userId: string; role: "MANAGER" | "VIEWER"; name: string; email: string; isActive: boolean; createdAt: string }[];
};

export const getClientSettings = cache(async (clientId: string): Promise<SettingsData> => {
  const [connections, socialAccounts, seoProperties, adAccounts, members] = await Promise.all([
    db.connection.findMany({
      where: { clientId },
      orderBy: { createdAt: "asc" },
      select: { id: true, provider: true, accountId: true, displayName: true, expiresAt: true, scopes: true, createdAt: true },
    }),
    db.socialAccount.findMany({
      where: { clientId, isCompetitor: false },
      orderBy: [{ platform: "asc" }, { username: "asc" }],
      select: { id: true, platform: true, username: true, displayName: true },
    }),
    db.seoProperty.findMany({
      where: { clientId },
      orderBy: { siteUrl: "asc" },
      select: { id: true, siteUrl: true, ga4PropertyId: true },
    }),
    db.adAccount.findMany({
      where: { clientId },
      orderBy: { name: "asc" },
      select: { id: true, externalId: true, name: true, currency: true },
    }),
    db.clientMember.findMany({
      where: { clientId },
      orderBy: { createdAt: "asc" },
      include: { user: { select: { id: true, name: true, email: true, isActive: true } } },
    }),
  ]);
  return {
    connections: connections.map((c) => ({
      id: c.id,
      provider: c.provider,
      accountId: c.accountId,
      displayName: c.displayName,
      expiresAt: c.expiresAt ? c.expiresAt.toISOString() : null,
      scopes: c.scopes,
      createdAt: c.createdAt.toISOString(),
    })),
    socialAccounts,
    seoProperties,
    adAccounts,
    members: members.map((m) => ({
      userId: m.userId,
      role: m.role,
      name: m.user.name,
      email: m.user.email,
      isActive: m.user.isActive,
      createdAt: m.createdAt.toISOString(),
    })),
  };
});

/** Active users that can be added as members (id/name/email only). */
export const listUserOptions = cache(async (): Promise<{ id: string; name: string; email: string }[]> => {
  return db.user.findMany({
    where: { isActive: true },
    orderBy: { name: "asc" },
    select: { id: true, name: true, email: true },
  });
});
