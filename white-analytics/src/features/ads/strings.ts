/**
 * Modul Meta Ads — string UI tambahan (Bahasa Indonesia).
 * Kunci umum tetap dari `t.ads.*` di src/i18n/id.ts; file ini hanya untuk
 * string BARU modul ini (i18n/id.ts tidak diedit agar tidak bentrok antar-agen).
 */
export const s = {
  // Filter
  allCampaigns: "Semua kampanye",
  filterCampaign: "Filter kampanye",

  // KPI captions & hints
  resultsHint: "Jumlah hasil kampanye konversi (klik tautan, pembelian, prospek, percakapan). Kampanye awareness (jangkauan) tidak dihitung agar satuan tidak tercampur.",
  cprHint: "Belanja iklan ÷ hasil konversi. Lebih rendah lebih baik.",
  ctrHint: "Klik tautan ÷ impresi × 100.",
  cpcHint: "Belanja ÷ klik tautan. Lebih rendah lebih baik.",
  cpmHint: "Belanja ÷ impresi × 1.000. Lebih rendah lebih baik.",
  frequencyHint: "Impresi ÷ jangkauan. Di atas 3 mengindikasikan ad fatigue.",
  roasHint: "Nilai konversi pembelian ÷ belanja iklan. Hanya tampil bila ada data pembelian.",
  reachHint: "Perkiraan akun unik yang melihat iklan (penjumlahan harian, mendekati).",
  noConversionResults: "Belum ada hasil konversi",

  // Charts
  shareLabel: "Porsi",
  rowKindLabel: "Tipe baris",
  rowKindDaily: "Harian",
  spendDailyDesc: "Belanja iklan per hari vs periode sebelumnya",
  resultsDailyDesc: "Hasil konversi per hari (kampanye awareness dikecualikan)",
  spendByCampaign: "Belanja per kampanye",
  spendByCampaignDesc: "Porsi belanja tiap kampanye pada periode ini",
  demographicsAgeTitle: "Belanja per usia",
  demographicsGenderTitle: "Belanja per gender",
  demographicsOverlapNote: "Rentang demografi Meta mencakup seluruh masa aktif kampanye yang beririsan dengan periode terpilih.",

  // Breakdown table
  expandRow: "Perluas baris",
  collapseRow: "Ciutkan baris",
  fatigueBadge: "Fatigue",
  sortBySpend: "Urutkan belanja",
  sortByCpr: "Urutkan biaya/hasil",
  breakdownEmpty: "Belum ada kampanye pada periode ini",

  // Top ads
  topAdsEmpty: "Belum ada iklan dengan hasil memadai pada periode ini",

  // CSV import dialog
  chooseFile: "Pilih file",
  fileTooLarge: "Ukuran file melebihi 10MB",
  notCsv: "File harus berformat .csv",
  previewTitle: "Pratinjau (5 baris pertama)",
  uploadAndImport: "Unggah & impor",
  importing: "Mengimpor…",
  parsingPreview: "Membaca file…",
  sampleFormatTitle: "Contoh format",
  sampleFormatDesc:
    "Ekspor dari Meta Ads Manager: Laporan → Ekspor data tabel → CSV. Sertakan kolom Campaign name / Nama kampanye, Ad set name, Ad name, Day / Hari, Amount spent (IDR), Impressions / Tayangan, Reach / Jangkauan, Link clicks / Klik tautan, Results / Hasil, Result indicator. Tambahkan breakdown Age / Usia & Gender untuk data demografi.",
  detectedNone: "tidak terdeteksi",
  columnLabel: "Kolom",
  mappedTo: "Terpetakan ke",
  dailyRows: "baris harian",
  demographicRows: "baris demografi",
  importSummary: (r: { campaigns: number; adSets: number; ads: number; days: number; demographics: number }) =>
    `${r.campaigns} kampanye, ${r.adSets} set iklan, ${r.ads} iklan, ${r.days} hari data, ${r.demographics} baris demografi.`,
  removeFile: "Hapus file",

  // Sync
  syncDone: "Sinkronisasi selesai",
  syncDemoMessage: (n: number) => `Sinkronisasi demo: ${n} kampanye diperbarui.`,
  syncNoAccount: "Belum ada akun iklan untuk disinkronkan.",

  // Errors / validation
  noFile: "Pilih file CSV terlebih dahulu.",
  csvNoRows: "Tidak ada baris data yang bisa dibaca dari file.",
  csvMissingColumns: (cols: string) => `Kolom wajib tidak ditemukan: ${cols}.`,
} as const;

/** Label Indonesia untuk resultType Meta. */
export const RESULT_TYPE_LABEL: Record<string, string> = {
  link_click: "Klik tautan",
  purchase: "Pembelian",
  lead: "Prospek",
  messaging_conversation_started: "Percakapan",
  reach: "Jangkauan",
  impressions: "Impresi",
  video_view: "Tayangan video",
  thruplay: "ThruPlay",
};

export function resultTypeLabel(type: string | null | undefined): string {
  if (!type) return "Hasil";
  return RESULT_TYPE_LABEL[type] ?? type.replace(/_/g, " ");
}

/** Label Indonesia untuk objective kampanye Meta (ODAX). */
export const OBJECTIVE_LABEL: Record<string, string> = {
  OUTCOME_TRAFFIC: "Traffic",
  OUTCOME_SALES: "Penjualan",
  OUTCOME_LEADS: "Prospek",
  OUTCOME_AWARENESS: "Awareness",
  OUTCOME_ENGAGEMENT: "Interaksi",
  OUTCOME_APP_PROMOTION: "Promosi aplikasi",
};

export function objectiveLabel(objective: string | null | undefined): string {
  if (!objective) return "–";
  return OBJECTIVE_LABEL[objective] ?? objective.replace(/^OUTCOME_/, "").replace(/_/g, " ").toLowerCase().replace(/^./, (c) => c.toUpperCase());
}

/** Label gender Indonesia (male/female/unknown). */
export const GENDER_LABEL: Record<string, string> = {
  male: "Pria",
  female: "Wanita",
  unknown: "Tidak diketahui",
};

export function genderLabel(g: string): string {
  return GENDER_LABEL[g] ?? g;
}
