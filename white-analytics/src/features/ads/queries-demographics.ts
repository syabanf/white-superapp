/** Age and gender tables of the ads dashboard. */
import "server-only";

const AGE_ORDER = ["13-17", "18-24", "25-34", "35-44", "45-54", "55-64", "65+", "unknown"];
const GENDER_ORDER = ["female", "male", "unknown"];

export type AdsDemographicRow = { age: string; gender: string; spend: number; results: number };

export type AdsDemographics = {
  hasData: boolean;
  byAge: AdsDemographicRow[];
  byGender: AdsDemographicRow[];
  matrix: AdsDemographicRow[];
};

/** Demografi (baris AdDemographic yang beririsan dengan rentang). */
export function buildDemographics(demoRows: AdsDemographicRow[]): AdsDemographics {
  const aggDemo = (key: (r: AdsDemographicRow) => string) => {
    const m = new Map<string, { spend: number; results: number }>();
    for (const r of demoRows) {
      const k = key(r);
      const acc = m.get(k) ?? { spend: 0, results: 0 };
      acc.spend += r.spend;
      acc.results += r.results;
      m.set(k, acc);
    }
    return m;
  };
  const orderIndex = (order: string[], v: string) => {
    const i = order.indexOf(v);
    return i === -1 ? order.length : i;
  };
  const byAge: AdsDemographicRow[] = [...aggDemo((r) => r.age)]
    .map(([age, v]) => ({ age, gender: "", ...v }))
    .sort((a, b) => orderIndex(AGE_ORDER, a.age) - orderIndex(AGE_ORDER, b.age) || a.age.localeCompare(b.age));
  const byGender: AdsDemographicRow[] = [...aggDemo((r) => r.gender)]
    .map(([gender, v]) => ({ age: "", gender, ...v }))
    .sort((a, b) => orderIndex(GENDER_ORDER, a.gender) - orderIndex(GENDER_ORDER, b.gender));
  const matrixMap = aggDemo((r) => `${r.age}|${r.gender}`);
  const matrix: AdsDemographicRow[] = [...matrixMap]
    .map(([k, v]) => {
      const [age, gender] = k.split("|") as [string, string];
      return { age, gender, ...v };
    })
    .sort(
      (a, b) =>
        orderIndex(AGE_ORDER, a.age) - orderIndex(AGE_ORDER, b.age) ||
        orderIndex(GENDER_ORDER, a.gender) - orderIndex(GENDER_ORDER, b.gender),
    );

  return { hasData: demoRows.length > 0, byAge, byGender, matrix };
}
