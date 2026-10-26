/**
 * Cache an AI insight per client and module for the presets people open first
 * (28 and 30 days, this month, 90 days), so the overview and report cards are not empty
 * in a demo. Uses the mock writer on purpose: no client data leaves the machine
 * and no OpenRouter credit is spent, even when a key is configured. Skips ranges
 * that already have an insight. Run: pnpm db:seed-insights
 */
import { buildInsightPayload, toRangeInfo } from "@/features/reports/insights";
import type { InsightModule } from "@/features/reports/queries";
import { db } from "@/lib/db";
import { previousRange, rangeForPreset, type PresetKey } from "@/lib/dates";
import { mockInsightProvider } from "@/lib/providers/openrouter/mock";

const MODULES: InsightModule[] = ["OVERVIEW", "SOCIAL", "SEO", "ADS"];
const PRESETS: PresetKey[] = ["28d", "30d", "thisMonth", "90d"];

async function main() {
  const clients = await db.client.findMany({ select: { id: true, name: true } });
  let created = 0;
  for (const client of clients) {
    for (const preset of PRESETS) {
      const range = rangeForPreset(preset);
      const previous = previousRange(range);
      for (const mod of MODULES) {
        const exists = await db.aiInsight.findFirst({ where: { clientId: client.id, module: mod, dateFrom: range.from, dateTo: range.to } });
        if (exists) continue;
        const data = await buildInsightPayload(client.id, mod, range, previous);
        const content = await mockInsightProvider.generateInsight({
          module: mod,
          language: "id",
          clientName: client.name,
          range: toRangeInfo(range, previous),
          data,
        });
        await db.aiInsight.create({
          data: { clientId: client.id, module: mod, dateFrom: range.from, dateTo: range.to, language: "id", model: mockInsightProvider.model, content },
        });
        created++;
      }
    }
  }
  console.log(`✓ ${created} insight dibuat untuk ${clients.length} klien.`);
}

main()
  .catch((e) => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await db.$disconnect();
  });
