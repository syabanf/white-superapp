import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatDateShort, formatNumber } from "@/lib/format";

/** Table view of a weekly chart: one row per week, one column per series. */
export function WeekTable({
  rows,
  headers,
  format = formatNumber,
}: {
  rows: { date: string; values: (number | null | undefined)[] }[];
  headers: string[];
  format?: (v: number) => string;
}) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="text-xs">Minggu</TableHead>
          {headers.map((h) => (
            <TableHead key={h} className="text-right text-xs">
              {h}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((r) => (
          <TableRow key={r.date}>
            <TableCell className="text-sm tabular">{formatDateShort(r.date)}</TableCell>
            {r.values.map((v, i) => (
              <TableCell key={headers[i]} className="text-right text-sm tabular">
                {v == null ? "–" : format(v)}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
