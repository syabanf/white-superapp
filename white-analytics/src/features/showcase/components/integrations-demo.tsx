import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { INTEGRATIONS, type IntegrationState } from "../data";
import { DemoTable, Panel } from "./shared";

const STATE: Record<IntegrationState, { label: string; kind: StatusKind }> = {
  live: { label: "Sudah ada", kind: "good" },
  partial: { label: "Sebagian", kind: "warning" },
  planned: { label: "Rencana", kind: "neutral" },
};

export function IntegrationsDemo() {
  return (
    <Panel title="Layanan dan API" hint="Status ini bukan simulasi: mengikuti kode saat ini">
      <DemoTable head={[{ label: "Layanan" }, { label: "Dipakai untuk" }, { label: "Status" }, { label: "Catatan" }]}>
        {INTEGRATIONS.map((i) => (
          <tr key={i.name}>
            <td className="px-4 py-3 font-medium">{i.name}</td>
            <td className="px-4 py-3 text-muted-foreground">{i.use}</td>
            <td className="px-4 py-3">
              <StatusBadge kind={STATE[i.state].kind}>{STATE[i.state].label}</StatusBadge>
            </td>
            <td className="min-w-56 px-4 py-3 text-muted-foreground">{i.note}</td>
          </tr>
        ))}
      </DemoTable>
    </Panel>
  );
}
