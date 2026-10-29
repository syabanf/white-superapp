/**
 * UI and notification strings for the ads automation module (anomaly alerts,
 * weekly ADS insight, recurring reports). Bahasa Indonesia, no em dash.
 */
import type { AnomalyRule } from "@/lib/metrics/ads-anomaly";

export const RULE_LABEL: Record<AnomalyRule, string> = {
  CPR_SPIKE: "Biaya per hasil naik",
  CTR_DROP: "CTR turun",
  ROAS_DROP: "ROAS turun",
  SPEND_SPIKE: "Belanja melonjak",
  SPEND_STALL: "Belanja berhenti",
  FATIGUE: "Ad fatigue",
};

export function ruleLabel(rule: string): string {
  return RULE_LABEL[rule as AnomalyRule] ?? rule;
}

export const aa = {
  alerts: {
    title: "Peringatan otomatis",
    hint: "Dicek otomatis setiap hari setelah sinkronisasi Meta Ads. Hari terakhir yang lengkap dibandingkan dengan rata-rata 7 hari sebelumnya: biaya per hasil naik ≥ 40%, CTR atau ROAS turun ≥ 30%, belanja melonjak ≥ 60% atau berhenti, dan frekuensi di atas 3. Kampanye bervolume kecil dilewati agar tidak berisik. Penyimpangan yang lebih kecil tampil sebagai Info tanpa notifikasi.",
    checkNow: "Periksa sekarang",
    checking: "Memeriksa…",
    checked: (n: number) => (n > 0 ? `${n} peringatan baru ditemukan` : "Pemeriksaan selesai, tidak ada peringatan baru"),
    emptyTitle: "Tidak ada anomali",
    emptyDesc: "Semua kampanye berjalan dalam batas normal 7 hari terakhir.",
    lastChecked: "Terakhir dicek",
    neverChecked: "Belum pernah dicek",
    resolve: "Tandai selesai",
    resolved: "Peringatan ditandai selesai",
    more: (n: number) => `+${n} peringatan lainnya`,
    showLess: "Tampilkan lebih sedikit",
    account: "Seluruh akun",
    severity: { INFO: "Info", WARNING: "Perhatian", CRITICAL: "Kritis" } as Record<string, string>,
    weeklyInsight: "Analisa AI mingguan",
    weeklyInsightOpen: "Lihat analisa",
    weeklyInsightNone: "Analisa AI mingguan dibuat otomatis setiap 7 hari.",
  },
  notif: {
    anomalyTitle: (n: number, critical: number) =>
      critical > 0 ? `${critical} anomali kritis di Meta Ads` : `${n} anomali baru di Meta Ads`,
    anomalyBody: (sample: string, more: number) => (more > 0 ? `${sample} (+${more} lainnya)` : sample),
    reportTitle: (name: string) => `Laporan otomatis siap: ${name}`,
    reportBody: (period: string) => `Laporan periode ${period} sudah dibuat dan bisa diunduh sebagai PDF.`,
    insightTitle: "Analisa AI mingguan Meta Ads siap",
    insightBody: (period: string) => `Ringkasan strategis Meta Ads untuk ${period}.`,
  },
  schedules: {
    title: "Laporan otomatis",
    hint: "Laporan dibuat otomatis oleh job harian. Mingguan terbit setiap Senin untuk minggu sebelumnya (Senin sampai Minggu), bulanan terbit setiap tanggal 1 untuk bulan sebelumnya. Tim klien mendapat notifikasi saat laporan siap.",
    add: "Jadwal baru",
    edit: "Ubah jadwal",
    emptyTitle: "Belum ada laporan otomatis",
    emptyDesc: "Buat jadwal agar laporan mingguan atau bulanan terbit sendiri.",
    name: "Nama laporan",
    namePlaceholder: "Contoh: Laporan mingguan",
    nameRequired: "Nama laporan wajib diisi.",
    cadence: "Frekuensi",
    cadenceLabel: { WEEKLY: "Mingguan", MONTHLY: "Bulanan" } as Record<string, string>,
    cadenceDesc: { WEEKLY: "Setiap Senin, minggu lalu", MONTHLY: "Setiap tanggal 1, bulan lalu" } as Record<string, string>,
    modules: "Modul",
    noModules: "Pilih minimal satu modul.",
    withInsight: "Sertakan ringkasan AI",
    enabled: "Aktif",
    nextRun: "Berikutnya",
    lastRun: "Terakhir",
    never: "Belum pernah",
    paused: "Dijeda",
    save: "Simpan",
    cancel: "Batal",
    saved: "Jadwal disimpan",
    created: "Jadwal dibuat",
    deleted: "Jadwal dihapus",
    deleteTitle: "Hapus jadwal laporan?",
    deleteDesc: "Laporan yang sudah dibuat tetap tersimpan di riwayat.",
    delete: "Hapus",
    runNow: "Jalankan sekarang",
    running: "Membuat laporan…",
    ran: "Laporan dibuat, lihat di riwayat",
    toggledOn: "Jadwal diaktifkan",
    toggledOff: "Jadwal dijeda",
    defaultNameWeekly: "Laporan mingguan",
    defaultNameMonthly: "Laporan bulanan",
  },
};
