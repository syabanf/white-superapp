import "server-only";
import { db } from "@/lib/db";
import { slugify } from "@/lib/format";
import { parseISODate, todayUtc } from "@/lib/dates";
import type { ParsedAdsRow } from "./csv-import";

export type ImportSummary = {
  campaigns: number;
  adSets: number;
  ads: number;
  days: number;
  demographics: number;
  warnings: string[];
};

/** Objective ODAX yang wajar untuk sebuah resultType (dipakai saat create dari CSV). */
function objectiveForResultType(resultType: string | null): string {
  switch (resultType) {
    case "purchase":
      return "OUTCOME_SALES";
    case "lead":
      return "OUTCOME_LEADS";
    case "messaging_conversation_started":
      return "OUTCOME_ENGAGEMENT";
    case "reach":
    case "impressions":
    case "video_view":
    case "thruplay":
      return "OUTCOME_AWARENESS";
    default:
      return "OUTCOME_TRAFFIC";
  }
}

function chunk<T>(arr: T[], size: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < arr.length; i += size) out.push(arr.slice(i, i + size));
  return out;
}

const FALLBACK_ADSET = "(Impor CSV)";

/**
 * Tulis baris hasil parse CSV ke DB:
 * - pastikan AdAccount ada (buat "Impor CSV" dengan externalId `csv_<clientId>` bila belum),
 * - upsert hierarki kampanye→set→iklan berdasarkan nama (externalId = ID Meta bila ada,
 *   selain itu slug nama),
 * - upsert AdDailyInsight per (akun, tanggal, campaign, adset, ad) — idempoten,
 * - agregasi baris demografi ke AdDemographic per (kampanye, usia, gender) sepanjang
 *   rentang tanggal file.
 */
export async function persistCsvRows(clientId: string, rows: ParsedAdsRow[], parseWarnings: string[]): Promise<ImportSummary> {
  const warnings = [...parseWarnings];

  // 1) Pastikan ada AdAccount (buat akun "Impor CSV" bila belum ada)
  let account = await db.adAccount.findFirst({ where: { clientId }, orderBy: { createdAt: "asc" } });
  if (!account) {
    account = await db.adAccount.create({
      data: { clientId, externalId: `csv_${clientId}`, name: "Impor CSV", currency: "IDR" },
    });
  }

  // 2) Upsert kampanye (kunci: externalId = ID Meta bila ada, selain itu slug nama)
  const campaignKey = (r: ParsedAdsRow) => r.campaignExternalId ?? slugify(r.campaignName);
  const campaignIds = new Map<string, string>(); // key → id DB
  const campaignInfo = new Map<string, { name: string; resultType: string | null }>();
  for (const r of rows) {
    const key = campaignKey(r);
    if (!campaignInfo.has(key)) campaignInfo.set(key, { name: r.campaignName, resultType: r.resultType });
    else if (r.resultType && !campaignInfo.get(key)!.resultType) campaignInfo.get(key)!.resultType = r.resultType;
  }
  for (const [key, info] of campaignInfo) {
    const c = await db.adCampaign.upsert({
      where: { adAccountId_externalId: { adAccountId: account.id, externalId: key } },
      update: { name: info.name },
      create: {
        adAccountId: account.id,
        externalId: key,
        name: info.name,
        objective: objectiveForResultType(info.resultType),
      },
    });
    campaignIds.set(key, c.id);
  }

  // 3) Upsert set iklan. Baris dengan iklan tanpa nama set → set fallback "(Impor CSV)".
  const adSetKeyOf = (r: ParsedAdsRow): string | null => {
    if (r.adSetName) return `${campaignKey(r)}|${r.adSetExternalId ?? slugify(r.adSetName)}`;
    if (r.adName) return `${campaignKey(r)}|${slugify(`${r.campaignName} ${FALLBACK_ADSET}`)}`;
    return null;
  };
  const adSetIds = new Map<string, string>();
  const adSetInfo = new Map<string, { campaignKey: string; externalId: string; name: string }>();
  for (const r of rows) {
    const key = adSetKeyOf(r);
    if (!key || adSetInfo.has(key)) continue;
    adSetInfo.set(key, {
      campaignKey: campaignKey(r),
      externalId: key.split("|")[1]!,
      name: r.adSetName ?? FALLBACK_ADSET,
    });
  }
  for (const [key, info] of adSetInfo) {
    const setRow = await db.adSet.upsert({
      where: { campaignId_externalId: { campaignId: campaignIds.get(info.campaignKey)!, externalId: info.externalId } },
      update: { name: info.name },
      create: { campaignId: campaignIds.get(info.campaignKey)!, externalId: info.externalId, name: info.name },
    });
    adSetIds.set(key, setRow.id);
  }

  // 4) Upsert iklan
  const adKeyOf = (r: ParsedAdsRow): string | null => {
    const setKey = adSetKeyOf(r);
    if (!setKey || !r.adName) return null;
    return `${setKey}|${r.adExternalId ?? slugify(r.adName)}`;
  };
  const adIds = new Map<string, string>();
  const adInfo = new Map<string, { setKey: string; externalId: string; name: string }>();
  for (const r of rows) {
    const key = adKeyOf(r);
    if (!key || adInfo.has(key)) continue;
    adInfo.set(key, { setKey: adSetKeyOf(r)!, externalId: key.split("|")[2]!, name: r.adName! });
  }
  for (const [key, info] of adInfo) {
    const adRow = await db.ad.upsert({
      where: { adSetId_externalId: { adSetId: adSetIds.get(info.setKey)!, externalId: info.externalId } },
      update: { name: info.name },
      create: { adSetId: adSetIds.get(info.setKey)!, externalId: info.externalId, name: info.name },
    });
    adIds.set(key, adRow.id);
  }

  // 5) Agregasi baris harian per (tanggal, campaign, adset, ad) lalu upsert manual
  //    (unik gabungan berisi kolom nullable → findMany + update/createMany).
  type DailyAgg = {
    date: Date;
    campaignId: string | null;
    adSetId: string | null;
    adId: string | null;
    spend: number;
    impressions: number;
    reach: number;
    clicks: number;
    linkClicks: number;
    results: number;
    resultType: string | null;
    purchaseValue: number | null;
  };
  const dailyAgg = new Map<string, DailyAgg>();
  let campaignLevelRows = 0;
  for (const r of rows) {
    if (r.kind !== "daily" || !r.date) continue;
    const date = parseISODate(r.date);
    if (!date) continue;
    const cId = campaignIds.get(campaignKey(r)) ?? null;
    const sKey = adSetKeyOf(r);
    const aKey = adKeyOf(r);
    if (!aKey) campaignLevelRows++;
    const sId = sKey ? (adSetIds.get(sKey) ?? null) : null;
    const aId = aKey ? (adIds.get(aKey) ?? null) : null;
    const mapKey = `${r.date}|${cId}|${sId}|${aId}`;
    const acc = dailyAgg.get(mapKey) ?? {
      date,
      campaignId: cId,
      adSetId: sId,
      adId: aId,
      spend: 0,
      impressions: 0,
      reach: 0,
      clicks: 0,
      linkClicks: 0,
      results: 0,
      resultType: null,
      purchaseValue: null,
    };
    acc.spend += r.spend;
    acc.impressions += r.impressions;
    acc.reach += r.reach;
    acc.clicks += r.clicks;
    acc.linkClicks += r.linkClicks;
    acc.results += r.results;
    if (r.resultType) acc.resultType = r.resultType;
    if (r.purchaseValue != null) acc.purchaseValue = (acc.purchaseValue ?? 0) + r.purchaseValue;
    dailyAgg.set(mapKey, acc);
  }
  if (campaignLevelRows > 0) {
    warnings.push(
      `${campaignLevelRows} baris tanpa kolom iklan diimpor pada level kampanye/set — ekspor level iklan disarankan agar semua rincian terisi.`,
    );
  }

  const dates = [...new Set([...dailyAgg.values()].map((v) => v.date.getTime()))].map((tms) => new Date(tms));
  const existing = dates.length
    ? await db.adDailyInsight.findMany({ where: { adAccountId: account.id, date: { in: dates } } })
    : [];
  const existingByKey = new Map(existing.map((e) => [`${e.date.toISOString().slice(0, 10)}|${e.campaignId}|${e.adSetId}|${e.adId}`, e.id]));
  const toCreate: Array<DailyAgg & { adAccountId: string }> = [];
  const toUpdate: Array<{ id: string; data: Omit<DailyAgg, "date" | "campaignId" | "adSetId" | "adId"> }> = [];
  for (const [key, v] of dailyAgg) {
    const id = existingByKey.get(key);
    if (id) {
      toUpdate.push({
        id,
        data: {
          spend: v.spend,
          impressions: v.impressions,
          reach: v.reach,
          clicks: v.clicks,
          linkClicks: v.linkClicks,
          results: v.results,
          resultType: v.resultType,
          purchaseValue: v.purchaseValue,
        },
      });
    } else {
      toCreate.push({ ...v, adAccountId: account.id });
    }
  }
  for (const part of chunk(toCreate, 500)) {
    await db.adDailyInsight.createMany({ data: part });
  }
  for (const part of chunk(toUpdate, 100)) {
    await db.$transaction(part.map((u) => db.adDailyInsight.update({ where: { id: u.id }, data: u.data })));
  }

  // 6) Demografi: agregasi per (kampanye, usia, gender) sepanjang rentang tanggal file
  const demoRows = rows.filter((r) => r.kind === "demographic");
  let demographics = 0;
  if (demoRows.length > 0) {
    const allDates = rows.map((r) => r.date).filter((d): d is string => d != null).sort();
    const dateFrom = allDates.length ? parseISODate(allDates[0]!)! : todayUtc();
    const dateTo = allDates.length ? parseISODate(allDates[allDates.length - 1]!)! : todayUtc();

    type DemoAgg = { campaignId: string | null; age: string; gender: string; spend: number; impressions: number; reach: number; results: number };
    const demoAgg = new Map<string, DemoAgg>();
    for (const r of demoRows) {
      const cId = campaignIds.get(campaignKey(r)) ?? null;
      const age = r.age ?? "unknown";
      const gender = r.gender ?? "unknown";
      const key = `${cId}|${age}|${gender}`;
      const acc = demoAgg.get(key) ?? { campaignId: cId, age, gender, spend: 0, impressions: 0, reach: 0, results: 0 };
      acc.spend += r.spend;
      acc.impressions += r.impressions;
      acc.reach += r.reach;
      acc.results += r.results;
      demoAgg.set(key, acc);
    }
    demographics = demoAgg.size;

    const existingDemo = await db.adDemographic.findMany({ where: { adAccountId: account.id, dateFrom, dateTo } });
    const demoByKey = new Map(existingDemo.map((e) => [`${e.campaignId}|${e.age}|${e.gender}`, e.id]));
    const demoCreate: Array<DemoAgg & { adAccountId: string; dateFrom: Date; dateTo: Date }> = [];
    for (const [key, v] of demoAgg) {
      const id = demoByKey.get(key);
      if (id) {
        await db.adDemographic.update({
          where: { id },
          data: { spend: v.spend, impressions: v.impressions, reach: v.reach, results: v.results },
        });
      } else {
        demoCreate.push({ ...v, adAccountId: account.id, dateFrom, dateTo });
      }
    }
    if (demoCreate.length) await db.adDemographic.createMany({ data: demoCreate });
  }

  await db.adAccount.update({ where: { id: account.id }, data: { lastSyncedAt: new Date() } });

  return {
    campaigns: campaignInfo.size,
    adSets: adSetInfo.size,
    ads: adInfo.size,
    days: dates.length,
    demographics,
    warnings: warnings.slice(0, 8),
  };
}
