import type { Metadata } from "next";
import { EyeOff, Megaphone, Search, SearchX, Share2 } from "lucide-react";
import { EmptyState } from "@/components/dashboard/empty-state";
import { getShareContext } from "@/features/share/access";
import { getShareAds, getShareSeo, getShareSocial } from "@/features/share/queries";
import { SharePinForm } from "@/features/share/components/pin-form";
import { ShareTabs, type ShareTabItem } from "@/features/share/components/share-tabs";
import { ShareAdsSection, ShareSeoSection, ShareSocialSection } from "@/features/share/components/module-sections";
import { previousRange, rangeForPreset, type PresetKey } from "@/lib/dates";
import { formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";

const SHARE_PRESETS: PresetKey[] = ["7d", "28d", "90d"];

export async function generateMetadata(props: PageProps<"/share/[slug]">): Promise<Metadata> {
  const { slug } = await props.params;
  const ctx = await getShareContext(slug);
  const name = ctx.status === "not-found" ? t.share.title : ctx.client.name;
  return { title: `${name} · ${t.share.title}`, robots: { index: false, follow: false } };
}

export default async function SharePage(props: PageProps<"/share/[slug]">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const ctx = await getShareContext(slug);

  if (ctx.status === "not-found") {
    return <EmptyState icon={<SearchX className="size-5" />} title={t.share.notFound} description={t.share.notFoundDesc} className="my-auto" />;
  }
  if (ctx.status === "disabled") {
    return <EmptyState icon={<EyeOff className="size-5" />} title={t.share.disabled} description={t.share.disabledDesc} className="my-auto" />;
  }
  if (ctx.status === "locked") {
    return <SharePinForm slug={slug} clientName={ctx.client.name} />;
  }

  const client = ctx.client;
  const presetRaw = Array.isArray(sp.preset) ? sp.preset[0] : sp.preset;
  const preset: PresetKey = SHARE_PRESETS.includes(presetRaw as PresetKey) ? (presetRaw as PresetKey) : "28d";
  const range = rangeForPreset(preset);
  const previous = previousRange(range);

  const modules = client.shareModules;
  const [social, seo, ads] = await Promise.all([
    modules.includes("SOCIAL") ? getShareSocial(client.id, range, previous) : null,
    modules.includes("SEO") ? getShareSeo(client.id, range, previous) : null,
    modules.includes("ADS") ? getShareAds(client.id, range, previous, client.currency) : null,
  ]);

  const tabLabel = (icon: React.ReactNode, label: string) => (
    <span className="flex items-center gap-1.5">
      {icon} {label}
    </span>
  );
  const items: ShareTabItem[] = [];
  if (social) items.push({ key: "social", label: tabLabel(<Share2 className="size-3.5" />, t.nav.social), content: <ShareSocialSection data={social} /> });
  if (seo) items.push({ key: "seo", label: tabLabel(<Search className="size-3.5" />, t.nav.seo), content: <ShareSeoSection data={seo} /> });
  if (ads) items.push({ key: "ads", label: tabLabel(<Megaphone className="size-3.5" />, t.nav.ads), content: <ShareAdsSection data={ads} /> });

  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{t.share.title}</div>
          <h1 className="text-xl font-semibold tracking-tight">{client.name}</h1>
        </div>
        <p className="text-sm text-muted-foreground">
          {t.common.period}: <span className="font-medium text-foreground">{formatDateRange(range.from, range.to)}</span>
          <span className="ml-2 text-xs">({t.common.vsPrevious} {formatDateRange(previous.from, previous.to)})</span>
        </p>
      </div>
      {items.length === 0 ? (
        <EmptyState title={t.common.noData} description={t.share.disabledDesc} />
      ) : (
        <ShareTabs items={items} />
      )}
    </>
  );
}
