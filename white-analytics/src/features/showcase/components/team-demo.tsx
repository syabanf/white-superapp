import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { formatDecimal } from "@/lib/format";
import { TEAM, qualityScore, type TeamState } from "../data";
import { DemoTable, Panel } from "./shared";

const STATE: Record<TeamState, { label: string; kind: StatusKind }> = {
  working: { label: "Mengerjakan", kind: "good" },
  review: { label: "Menunggu review", kind: "warning" },
  meeting: { label: "Meeting", kind: "info" },
  offline: { label: "Tidak aktif", kind: "neutral" },
};

export function TeamDemo() {
  const ranked = [...TEAM].sort((a, b) => qualityScore(b) - qualityScore(a));
  return (
    <>
      <Panel title="Tim hari ini" hint="Diambil dari update tugas dan logbook. Layar dan aktivitas pribadi tidak dipantau.">
        <DemoTable head={[{ label: "Nama" }, { label: "Status" }, { label: "Sedang dikerjakan" }, { label: "Update terakhir" }, { label: "Selesai hari ini", right: true }]}>
          {TEAM.map((m) => (
            <tr key={m.name}>
              <td className="px-4 py-3 whitespace-nowrap">
                <div className="font-medium">{m.name}</div>
                <div className="text-xs text-muted-foreground">{m.role}</div>
              </td>
              <td className="px-4 py-3">
                <StatusBadge kind={STATE[m.state].kind}>{STATE[m.state].label}</StatusBadge>
              </td>
              <td className="min-w-52 px-4 py-3">
                <div>{m.doing}</div>
                <div className="text-xs text-muted-foreground">{m.client}</div>
              </td>
              <td className="px-4 py-3 whitespace-nowrap text-muted-foreground">{m.updated}</td>
              <td className="px-4 py-3 text-right tabular">{m.doneToday}</td>
            </tr>
          ))}
        </DemoTable>
      </Panel>

      <Panel title="Kualitas, selain ketepatan waktu" hint="Contoh bobot: lolos tanpa revisi 50%, rating klien 30%, tepat waktu 20%">
        <DemoTable head={[{ label: "Nama" }, { label: "Tepat waktu", right: true }, { label: "Lolos tanpa revisi", right: true }, { label: "Rating klien", right: true }, { label: "Skor kualitas", right: true }]}>
          {ranked.map((m) => (
            <tr key={m.name}>
              <td className="px-4 py-3 font-medium whitespace-nowrap">{m.name}</td>
              <td className="px-4 py-3 text-right tabular">{m.onTime}%</td>
              <td className="px-4 py-3 text-right tabular">{m.firstPass}%</td>
              <td className="px-4 py-3 text-right tabular">{formatDecimal(m.rating)}</td>
              <td className="px-4 py-3 text-right font-semibold tabular">{qualityScore(m)}</td>
            </tr>
          ))}
        </DemoTable>
        <p className="border-t px-(--card-pad) py-3 text-sm text-muted-foreground">
          Bima paling tepat waktu (98%), tapi hanya 41% karyanya lolos tanpa revisi. Citra lebih sering telat (84%), namun 88% karyanya langsung diterima klien. Penilaian on-time saja tidak akan menangkap perbedaan ini.
        </p>
      </Panel>
    </>
  );
}
