/**
 * Rule-based anomaly detection for Meta Ads (pure, deterministic; tested in
 * tests/ads-anomaly.test.ts). Import directly from `@/lib/metrics/ads-anomaly`.
 *
 * Each rule compares the anchor day (latest complete day) with a trailing
 * baseline of the previous `BASELINE_DAYS` days. Ratios (CPR, CTR, ROAS) are
 * pooled over the baseline (sum ÷ sum), spend uses the mean over days that
 * actually delivered. Missing days count as zero delivery.
 *
 * Severity tiers: INFO (worth a look, shown in the panel only), WARNING and
 * CRITICAL (also notified). Minimum-volume guards keep small campaigns from
 * producing noise: a CPR jump on 1 result is not an anomaly.
 */
import { formatCurrency, formatNumber } from "@/lib/format";
import { adCtr, costPerResult, frequency, isFatigued, roas } from "./ads";
import { safeDiv } from "./delta";

export type AnomalyRule = "CPR_SPIKE" | "CTR_DROP" | "ROAS_DROP" | "SPEND_SPIKE" | "SPEND_STALL" | "FATIGUE";
export type AnomalySeverity = "INFO" | "WARNING" | "CRITICAL";

export const ANOMALY_RULES: AnomalyRule[] = ["CPR_SPIKE", "CTR_DROP", "ROAS_DROP", "SPEND_SPIKE", "SPEND_STALL", "FATIGUE"];

/** One day of one campaign (or of the whole account). `date` is YYYY-MM-DD. */
export type AdDay = {
  date: string;
  spend: number;
  impressions: number;
  reach: number;
  linkClicks: number;
  /** conversion results only (awareness "reach" results are excluded by the caller) */
  results: number;
  purchaseValue: number | null;
};

export type AnomalySubject = {
  /** "" = account level */
  campaignId: string;
  name: string;
  /** campaign status is ACTIVE (account level: any campaign active) */
  active: boolean;
  /** objective produces conversion results (CPR is meaningful) */
  isConversion: boolean;
  days: AdDay[];
};

export type Anomaly = {
  campaignId: string;
  rule: AnomalyRule;
  severity: AnomalySeverity;
  date: string;
  metric: "cpr" | "ctr" | "roas" | "spend" | "frequency";
  current: number;
  baseline: number;
  /** relative change (0.45 = +45 %), null when not a ratio (stall, fatigue) */
  change: number | null;
  message: string;
};

export const BASELINE_DAYS = 7;
/** Days with spend inside the baseline before ratio/spend rules are trusted. */
export const MIN_BASELINE_ACTIVE_DAYS = 4;

export const THRESHOLDS = {
  CPR_SPIKE: { info: 0.25, warning: 0.4, critical: 0.8, minResults: 3, minBaselineResults: 10 },
  CTR_DROP: { info: -0.2, warning: -0.3, critical: -0.5, minImpressions: 1000, minBaselineImpressions: 5000 },
  ROAS_DROP: { info: -0.2, warning: -0.3, critical: -0.5 },
  SPEND_SPIKE: { info: 0.35, warning: 0.6, critical: 1.0 },
  SPEND_STALL: { minBaselineActiveDays: 3, criticalZeroDays: 2 },
  FATIGUE: { windowDays: 7, minImpressions: 5000, criticalFrequency: 5 },
} as const;

// ── date helpers (string based, UTC) ──────────────────────────

function shiftDay(iso: string, n: number): string {
  const d = new Date(`${iso}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

const ZERO = (date: string): AdDay => ({ date, spend: 0, impressions: 0, reach: 0, linkClicks: 0, results: 0, purchaseValue: null });

/** Sum several days into one (purchaseValue stays null when no day had it). */
export function sumDays(days: AdDay[], date = ""): AdDay {
  const out = ZERO(date);
  for (const d of days) {
    out.spend += d.spend;
    out.impressions += d.impressions;
    out.reach += d.reach;
    out.linkClicks += d.linkClicks;
    out.results += d.results;
    if (d.purchaseValue != null) out.purchaseValue = (out.purchaseValue ?? 0) + d.purchaseValue;
  }
  return out;
}

/** Build the anchor day and baseline window from a (possibly sparse) series. */
export function splitWindow(days: AdDay[], anchor: string, baselineDays = BASELINE_DAYS) {
  const byDate = new Map<string, AdDay>();
  for (const d of days) byDate.set(d.date, byDate.has(d.date) ? sumDays([byDate.get(d.date)!, d], d.date) : d);
  const at = (iso: string) => byDate.get(iso) ?? ZERO(iso);
  const baseline: AdDay[] = [];
  for (let i = baselineDays; i >= 1; i--) baseline.push(at(shiftDay(anchor, -i)));
  return { current: at(anchor), baseline, at };
}

/** Relative change; null when the baseline is 0. */
export function relChange(current: number, baseline: number): number | null {
  if (baseline === 0) return null;
  return current / baseline - 1;
}

/** Tier for an increase (`higher` = bad) or a decrease (thresholds negative). */
export function tier(change: number, t: { info: number; warning: number; critical: number }): AnomalySeverity | null {
  const up = t.critical > 0;
  const hit = (x: number) => (up ? change >= x : change <= x);
  if (hit(t.critical)) return "CRITICAL";
  if (hit(t.warning)) return "WARNING";
  if (hit(t.info)) return "INFO";
  return null;
}

const pct = (change: number) => `${formatNumber(Math.abs(change) * 100)}%`;

/** Subject name mid-sentence ("kampanye \"X\"") and at sentence start. */
function subjectLabel(s: AnomalySubject): { mid: string; start: string } {
  const mid = s.campaignId === "" ? "akun iklan" : `kampanye "${s.name}"`;
  return { mid, start: mid.charAt(0).toUpperCase() + mid.slice(1) };
}

// ── rules ─────────────────────────────────────────────────────

export function detectSubjectAnomalies(subject: AnomalySubject, anchor: string, currency = "IDR"): Anomaly[] {
  const { current, baseline, at } = splitWindow(subject.days, anchor);
  const base = sumDays(baseline);
  const activeDays = baseline.filter((d) => d.spend > 0);
  const { mid, start } = subjectLabel(subject);
  const money = (v: number) => formatCurrency(v, currency);
  const out: Anomaly[] = [];
  const push = (a: Omit<Anomaly, "campaignId" | "date">) => out.push({ ...a, campaignId: subject.campaignId, date: anchor });
  const trusted = activeDays.length >= MIN_BASELINE_ACTIVE_DAYS && current.spend > 0;

  // CPR_SPIKE — cost per result jumps (conversion objectives only)
  if (trusted && subject.isConversion) {
    const t = THRESHOLDS.CPR_SPIKE;
    const baseCpr = costPerResult(base.spend, base.results);
    if (base.results >= t.minBaselineResults && baseCpr > 0) {
      if (current.results === 0 && current.spend >= baseCpr * 2) {
        push({
          rule: "CPR_SPIKE",
          severity: "CRITICAL",
          metric: "cpr",
          current: current.spend,
          baseline: baseCpr,
          change: null,
          message: `${start} menghabiskan ${money(current.spend)} tanpa hasil, padahal biaya per hasil biasanya ${money(baseCpr)}.`,
        });
      } else if (current.results >= t.minResults) {
        const cpr = costPerResult(current.spend, current.results);
        const change = relChange(cpr, baseCpr);
        const sev = change == null ? null : tier(change, t);
        if (sev && change != null)
          push({
            rule: "CPR_SPIKE",
            severity: sev,
            metric: "cpr",
            current: cpr,
            baseline: baseCpr,
            change,
            message: `Biaya per hasil ${mid} naik ${pct(change)}: ${money(cpr)} dibanding rata-rata 7 hari ${money(baseCpr)}.`,
          });
      }
    }
  }

  // CTR_DROP — link CTR falls
  if (trusted) {
    const t = THRESHOLDS.CTR_DROP;
    if (current.impressions >= t.minImpressions && base.impressions >= t.minBaselineImpressions) {
      const ctr = adCtr(current.linkClicks, current.impressions);
      const baseCtr = adCtr(base.linkClicks, base.impressions);
      const change = relChange(ctr, baseCtr);
      const sev = change == null ? null : tier(change, t);
      if (sev && change != null)
        push({
          rule: "CTR_DROP",
          severity: sev,
          metric: "ctr",
          current: ctr,
          baseline: baseCtr,
          change,
          message: `CTR ${mid} turun ${pct(change)}: ${formatNumber(ctr, 2)}% dibanding rata-rata 7 hari ${formatNumber(baseCtr, 2)}%.`,
        });
    }
  }

  // ROAS_DROP — campaign level only (account spend also covers non-sales campaigns),
  // and only when purchase value is reported
  if (trusted && subject.campaignId !== "" && base.purchaseValue != null && base.purchaseValue > 0) {
    const r = roas(current.purchaseValue ?? 0, current.spend) ?? 0;
    const baseRoas = roas(base.purchaseValue, base.spend) ?? 0;
    const change = relChange(r, baseRoas);
    const sev = change == null ? null : tier(change, THRESHOLDS.ROAS_DROP);
    if (sev && change != null)
      push({
        rule: "ROAS_DROP",
        severity: sev,
        metric: "roas",
        current: r,
        baseline: baseRoas,
        change,
        message: `ROAS ${mid} turun ${pct(change)}: ${formatNumber(r, 2)}× dibanding rata-rata 7 hari ${formatNumber(baseRoas, 2)}×.`,
      });
  }

  // SPEND_SPIKE — daily spend far above the usual delivery
  if (trusted) {
    const meanSpend = safeDiv(sumDays(activeDays).spend, activeDays.length);
    const change = relChange(current.spend, meanSpend);
    const sev = change == null ? null : tier(change, THRESHOLDS.SPEND_SPIKE);
    if (sev && change != null)
      push({
        rule: "SPEND_SPIKE",
        severity: sev,
        metric: "spend",
        current: current.spend,
        baseline: meanSpend,
        change,
        message: `Belanja ${mid} melonjak ${pct(change)}: ${money(current.spend)} dibanding rata-rata harian ${money(meanSpend)}.`,
      });
  }

  // SPEND_STALL — an active subject stops delivering after spending in the baseline
  if (subject.active && current.spend === 0 && activeDays.length >= THRESHOLDS.SPEND_STALL.minBaselineActiveDays) {
    let zeroDays = 0;
    for (let i = 0; i < BASELINE_DAYS + 1 && at(shiftDay(anchor, -i)).spend === 0; i++) zeroDays++;
    const meanSpend = safeDiv(sumDays(activeDays).spend, activeDays.length);
    push({
      rule: "SPEND_STALL",
      severity: zeroDays >= THRESHOLDS.SPEND_STALL.criticalZeroDays ? "CRITICAL" : "WARNING",
      metric: "spend",
      current: 0,
      baseline: meanSpend,
      change: null,
      message:
        zeroDays >= 2
          ? `${start} berstatus aktif tetapi tidak membelanjakan anggaran selama ${zeroDays} hari. Cek pembayaran, review iklan, atau jadwal.`
          : `${start} berstatus aktif tetapi berhenti membelanjakan anggaran (biasanya ${money(meanSpend)} per hari).`,
    });
  }

  // FATIGUE — frequency over the last 7 days (anchor included)
  {
    const t = THRESHOLDS.FATIGUE;
    const window = [...baseline.slice(-(t.windowDays - 1)), current];
    const w = sumDays(window);
    const freq = frequency(w.impressions, w.reach);
    if (subject.active && current.spend > 0 && w.impressions >= t.minImpressions && isFatigued(freq)) {
      push({
        rule: "FATIGUE",
        severity: freq > t.criticalFrequency ? "CRITICAL" : "WARNING",
        metric: "frequency",
        current: freq,
        baseline: 3,
        change: null,
        message: `Frekuensi ${mid} mencapai ${formatNumber(freq, 1)} dalam 7 hari. Audiens mulai jenuh, segarkan materi iklan atau perluas target.`,
      });
    }
  }

  return out;
}

/** Run every rule over every subject for one anchor day. Sorted: CRITICAL first. */
export function detectAnomalies(subjects: AnomalySubject[], anchor: string, currency = "IDR"): Anomaly[] {
  const all = subjects.flatMap((s) => detectSubjectAnomalies(s, anchor, currency));
  return all.sort((a, b) => severityRank(b.severity) - severityRank(a.severity) || a.rule.localeCompare(b.rule));
}

export function severityRank(s: AnomalySeverity): number {
  return s === "CRITICAL" ? 2 : s === "WARNING" ? 1 : 0;
}

/** Only WARNING and CRITICAL reach the notification bell. */
export function isNotifiable(s: AnomalySeverity): boolean {
  return s !== "INFO";
}

/** Sum campaign subjects into the account-level subject (campaignId ""). */
export function accountSubject(campaigns: AnomalySubject[]): AnomalySubject {
  const byDate = new Map<string, AdDay[]>();
  for (const c of campaigns)
    for (const d of c.days) {
      // CPR at account level only counts conversion campaigns
      const day = c.isConversion ? d : { ...d, results: 0 };
      byDate.set(d.date, [...(byDate.get(d.date) ?? []), day]);
    }
  return {
    campaignId: "",
    name: "",
    active: campaigns.some((c) => c.active),
    isConversion: campaigns.some((c) => c.isConversion),
    days: [...byDate.entries()].map(([date, ds]) => sumDays(ds, date)),
  };
}
