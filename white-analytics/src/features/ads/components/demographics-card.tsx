"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatCompact, formatCurrency, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import { genderLabel, s } from "@/features/ads/strings";

export type DemographicRow = { age: string; gender: string; spend: number; results: number };

/**
 * (d) Demografi — dua HBarChart (Usia; Gender) untuk belanja, hasil sebagai
 * teks sekunder, plus tabel silang usia×gender sebagai kembaran tabel.
 */
export function DemographicsCard({
  byAge,
  byGender,
  matrix,
  currency,
  hasData,
}: {
  byAge: DemographicRow[];
  byGender: DemographicRow[];
  matrix: DemographicRow[];
  currency: string;
  hasData: boolean;
}) {
  const money = (v: number) => formatCurrency(v, currency, { compact: true });
  const genders = [...new Set(matrix.map((m) => m.gender))];
  const ages = [...new Set(matrix.map((m) => m.age))];
  const cell = new Map(matrix.map((m) => [`${m.age}|${m.gender}`, m]));

  const table = (
    <Table className="[&_td]:py-1.5 [&_th]:h-8">
      <TableHeader>
        <TableRow className="hover:bg-transparent">
          <TableHead className="text-xs">{t.ads.age}</TableHead>
          {genders.map((g) => (
            <TableHead key={g} className="text-right text-xs">
              {genderLabel(g)}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {ages.map((age) => (
          <TableRow key={age}>
            <TableCell className="text-xs font-medium">{age}</TableCell>
            {genders.map((g) => {
              const m = cell.get(`${age}|${g}`);
              return (
                <TableCell key={g} className="text-right text-xs tabular">
                  {m ? (
                    <>
                      {money(m.spend)}
                      <span className="ml-1 text-muted-foreground">· {formatCompact(m.results)}</span>
                    </>
                  ) : (
                    "–"
                  )}
                </TableCell>
              );
            })}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );

  return (
    <ChartCard
      title={t.ads.demographics}
      description={t.ads.demographicsDesc}
      table={hasData ? table : undefined}
      footer={hasData ? s.demographicsOverlapNote : undefined}
    >
      {!hasData ? (
        <p className="py-8 text-center text-sm text-muted-foreground">{t.common.noData}</p>
      ) : (
        <div className="grid gap-5 sm:grid-cols-2">
          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">{s.demographicsAgeTitle}</p>
            <HBarChart
              items={byAge.map((r) => ({
                label: r.age,
                value: r.spend,
                secondary: `${formatNumber(r.results)} ${t.ads.results.toLowerCase()}`,
              }))}
              format={money}
            />
          </div>
          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">{s.demographicsGenderTitle}</p>
            <HBarChart
              items={byGender.map((r) => ({
                label: genderLabel(r.gender),
                value: r.spend,
                secondary: `${formatNumber(r.results)} ${t.ads.results.toLowerCase()}`,
              }))}
              format={money}
            />
          </div>
        </div>
      )}
    </ChartCard>
  );
}
