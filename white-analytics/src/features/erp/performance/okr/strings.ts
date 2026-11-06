export const to = {
  title: "OKR",
  subtitle: "Objective per kuartal dan key result yang mengukurnya. Semua orang bisa melihat papan ini.",
  period: "Kuartal",
  levels: { company: "Perusahaan", department: "Departemen", employee: "Karyawan" } as Record<string, string>,
  statuses: { active: "Berjalan", done: "Selesai", cancelled: "Batal" } as Record<string, string>,
  addObjective: "Tambah objective",
  editObjective: "Ubah objective",
  level: "Level",
  department: "Departemen",
  employee: "Karyawan",
  parent: "Mendukung objective",
  noParent: "Tidak ada",
  titleField: "Objective",
  description: "Deskripsi",
  status: "Status",
  addKr: "Tambah key result",
  editKr: "Ubah key result",
  krTitle: "Key result",
  metric: "Satuan ukur",
  metrics: { number: "Angka", percent: "Persen", currency: "Rupiah" } as Record<string, string>,
  start: "Nilai awal",
  target: "Target",
  unit: "Unit (opsional)",
  checkIn: "Check-in",
  checkInTitle: "Check-in key result",
  value: "Nilai sekarang",
  confidence: "Keyakinan tercapai (1 sampai 10)",
  note: "Catatan",
  saved: "Tersimpan.",
  deleted: "Dihapus.",
  checkedIn: "Check-in tersimpan.",
  deleteObjective: "Hapus objective ini beserta key result-nya?",
  deleteKr: "Hapus key result ini?",
  delete: "Hapus",
  empty: "Belum ada OKR di kuartal ini.",
  noKr: "Belum ada key result.",
  progress: "Progres",
  confidenceShort: (n: number) => `Keyakinan ${n}/10`,
};

/** "2026-Q4" for a date. */
export function quarterOf(d: Date): string {
  return `${d.getFullYear()}-Q${Math.floor(d.getMonth() / 3) + 1}`;
}

export function shiftQuarter(period: string, by: number): string {
  const [y, q] = period.split("-Q").map(Number);
  const n = (y ?? 2026) * 4 + ((q ?? 1) - 1) + by;
  return `${Math.floor(n / 4)}-Q${(n % 4) + 1}`;
}
