"use client";

import * as React from "react";
import { useActionState } from "react";
import { CalendarRange, FileText, Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Section } from "@/components/dashboard/page-header";
import { DateRangePicker } from "@/components/shell/date-range-picker";
import { createReport, type CreateReportState } from "@/features/reports/actions";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

type ModuleKey = "SOCIAL" | "SEO" | "ADS";

const MODULE_OPTIONS: { key: ModuleKey; label: string }[] = [
  { key: "SOCIAL", label: t.overview.socialCard },
  { key: "SEO", label: t.overview.seoCard },
  { key: "ADS", label: t.overview.adsCard },
];

/**
 * Report builder. Module/AI toggles are client state so the preview below reacts live;
 * the server-rendered module previews come in as slots.
 */
export function ReportBuilderForm({
  slug,
  defaultTitle,
  from,
  to,
  rangeLabel,
  compareDefault,
  previews,
  insightPreview,
}: {
  slug: string;
  defaultTitle: string;
  from: string;
  to: string;
  rangeLabel: string;
  compareDefault: boolean;
  previews: Partial<Record<ModuleKey, React.ReactNode>>;
  insightPreview: React.ReactNode;
}) {
  const [state, formAction, pending] = useActionState<CreateReportState, FormData>(
    createReport.bind(null, slug),
    null,
  );
  const [modules, setModules] = React.useState<ModuleKey[]>(["SOCIAL", "SEO", "ADS"]);
  const [includeAi, setIncludeAi] = React.useState(true);
  const [compare, setCompare] = React.useState(compareDefault);
  const [language, setLanguage] = React.useState("id");

  React.useEffect(() => {
    if (state && !state.ok) toast.error(state.error);
  }, [state]);

  const toggleModule = (key: ModuleKey, checked: boolean) => {
    setModules((prev) => (checked ? [...prev, key] : prev.filter((m) => m !== key)));
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <FileText className="size-4 text-muted-foreground" /> {t.reports.builder}
          </CardTitle>
          <CardDescription>{t.reports.builderDesc}</CardDescription>
        </CardHeader>
        <CardContent>
          <form action={formAction} className="space-y-5">
            <input type="hidden" name="from" value={from} />
            <input type="hidden" name="to" value={to} />
            {modules.map((m) => (
              <input key={m} type="hidden" name="modules" value={m} />
            ))}
            <input type="hidden" name="compare" value={compare ? "true" : "false"} />
            <input type="hidden" name="includeAi" value={includeAi ? "true" : "false"} />
            <input type="hidden" name="language" value={language} />

            <div className="grid gap-5 md:grid-cols-2">
              <div className="space-y-2 md:col-span-2">
                <Label htmlFor="report-title">{t.reports.reportTitle}</Label>
                <Input
                  id="report-title"
                  name="title"
                  defaultValue={defaultTitle}
                  placeholder={rs.builder.titlePlaceholder}
                  required
                  maxLength={160}
                />
              </div>

              <div className="space-y-2.5">
                <Label>{t.reports.modules}</Label>
                <div className="space-y-2">
                  {MODULE_OPTIONS.map((m) => (
                    <label key={m.key} className="flex cursor-pointer items-center gap-2 text-sm">
                      <Checkbox
                        checked={modules.includes(m.key)}
                        onCheckedChange={(c) => toggleModule(m.key, c === true)}
                      />
                      {m.label}
                    </label>
                  ))}
                </div>
                {modules.length === 0 ? (
                  <p className="text-xs text-destructive">{rs.builder.noModules}</p>
                ) : null}
              </div>

              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>{t.reports.language}</Label>
                  <Select value={language} onValueChange={setLanguage}>
                    <SelectTrigger className="w-full">
                      <SelectValue>{language === "en" ? rs.builder.langEn : rs.builder.langId}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="id">{rs.builder.langId}</SelectItem>
                      <SelectItem value="en">{rs.builder.langEn}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <label className="flex cursor-pointer items-center justify-between gap-3 text-sm">
                  <span>
                    {rs.builder.includeComparison}
                    <span className="block text-xs text-muted-foreground">{t.common.comparePrevious}</span>
                  </span>
                  <Switch checked={compare} onCheckedChange={setCompare} />
                </label>
                <label className="flex cursor-pointer items-center justify-between gap-3 text-sm">
                  <span>
                    {t.reports.includeAi}
                    <span className="block text-xs text-muted-foreground">{rs.builder.aiNote}</span>
                  </span>
                  <Switch checked={includeAi} onCheckedChange={setIncludeAi} />
                </label>
              </div>
            </div>

            <Separator />
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex flex-wrap items-center gap-2">
                <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <CalendarRange className="size-3.5" />
                  {t.common.period}: <span className="font-medium text-foreground">{rangeLabel}</span>
                </p>
                <DateRangePicker showCompare={false} />
              </div>
              <Button type="submit" disabled={pending || modules.length === 0}>
                {pending ? <Loader2 className="size-4 animate-spin" /> : <FileText className="size-4" />}
                {pending ? t.reports.generating : t.reports.generate}
              </Button>
            </div>
            {state && !state.ok ? <p className="text-sm text-destructive">{state.error}</p> : null}
          </form>
        </CardContent>
      </Card>

      <Section title={rs.builder.previewTitle} description={rs.builder.previewDesc}>
        <div className="space-y-4">
          {MODULE_OPTIONS.filter((m) => modules.includes(m.key) && previews[m.key]).map((m) => (
            <div key={m.key}>{previews[m.key]}</div>
          ))}
          {includeAi ? insightPreview : null}
        </div>
      </Section>
    </div>
  );
}
