/**
 * Mock data for the /showcase demos. Nothing here touches a database, WhatsApp
 * or a third-party service; every name and number is invented.
 */

// ── WhatsApp simulation ──────────────────────────────────────────────────────

export type ChatStep = {
  from: "them" | "bot" | "me";
  name: string;
  text: string;
  /** What the dashboard records once this message lands. */
  effect?: string;
};

export type ChatScenario = {
  id: string;
  label: string;
  room: string;
  intro: string;
  /** Who the viewer plays; their typed text is sent under this name. */
  you: Pick<ChatStep, "from" | "name">;
  /** Messages already in the room before the viewer types. */
  opening: ChatStep[];
  placeholder: string;
  /** Facts and duties for the AI. Invented data, as everything else in this file. */
  context: string;
};

export const CHAT_SCENARIOS: ChatScenario[] = [
  {
    id: "reminder",
    label: "Reminder customer",
    room: "Chat: Bu Ratna, Kopi Nusantara",
    intro: "Invoice jatuh tempo besok. Anda berperan sebagai Bu Ratna, lalu AI menjawab dan mencatat jawaban Anda.",
    you: { from: "them", name: "Bu Ratna" },
    opening: [{ from: "bot", name: "Asisten Mixist", text: "Halo Bu Ratna, invoice INV-2611-0018 senilai Rp 11.988.000 jatuh tempo besok. Bukti transfer bisa dikirim di chat ini ya.", effect: "Reminder H-1 terkirim untuk INV-2611-0018" }],
    placeholder: "Contoh: Oke, tapi baru bisa transfer hari Jumat.",
    context: "Kamu menagih invoice INV-2611-0018 senilai Rp 11.988.000 milik Bu Ratna dari Kopi Nusantara, jatuh tempo besok. Catat janji bayar beserta tanggalnya, keluhan, atau permintaan perpanjangan. Beri tahu bahwa kamu meneruskan hal penting ke tim Finance.",
  },
  {
    id: "quotation",
    label: "Edit quotation",
    room: "Chat internal: Sales dan AI",
    intro: "Anda berperan sebagai Sales dan mengubah quotation lewat chat. AI menghitung ulang dan meminta konfirmasi sebelum menyimpan.",
    you: { from: "me", name: "Sales" },
    opening: [{ from: "bot", name: "Asisten Mixist", text: "Q-2611-0042 untuk Kopi Nusantara:\nSubtotal Rp 12.000.000\nDiskon 10% Rp 1.200.000\nPPN 11% Rp 1.188.000\nTotal Rp 11.988.000\n\nMau diubah apa?" }],
    placeholder: "Contoh: tambah Produksi Konten 4 juta, diskon tetap 10%.",
    context: "Kamu mengedit quotation Q-2611-0042 versi 1. Subtotal Rp 12.000.000, diskon 10%, PPN 11% dihitung dari subtotal setelah diskon, total Rp 11.988.000. Hitung ulang dengan teliti setiap ada perubahan, tampilkan subtotal, diskon, PPN, dan total, lalu minta konfirmasi. Simpan sebagai versi baru hanya setelah pengguna mengonfirmasi.",
  },
  {
    id: "notulen",
    label: "Notulen grup",
    room: "Grup: Tim Kopi Nusantara",
    intro: "Anda ikut di grup tim. Minta notulen atau tambahkan keputusan, lalu AI membuat tugas dan reminder per orang.",
    you: { from: "me", name: "Anda" },
    opening: [
      { from: "them", name: "Rangga", text: "Feed minggu depan temanya promo kopi susu ya, klien minta 6 post." },
      { from: "them", name: "Lala", text: "Aku bikin desainnya, paling lambat Kamis." },
      { from: "them", name: "Dimas", text: "Caption aku kerjakan Rabu." },
    ],
    placeholder: "Contoh: @Asisten notulen dong",
    context: "Kamu asisten di grup kerja. Rangkum obrolan grup menjadi keputusan dan daftar tugas per orang beserta tenggatnya hanya dari isi obrolan. Sebutkan reminder H-1 pukul 09.00 untuk tiap tugas. Kalau pengguna mengubah tugas atau tenggat, perbarui daftarnya.",
  },
  {
    id: "revisi",
    label: "Revisi klien",
    room: "Grup: Kopi Nusantara dan Mixist",
    intro: "Anda berperan sebagai Bu Ratna dan menulis revisi di grup. AI mengubahnya menjadi tugas pada file desain yang tepat.",
    you: { from: "them", name: "Bu Ratna" },
    opening: [{ from: "bot", name: "Asisten Mixist", text: "Selamat pagi, Bu. Feed Promo 01, 02, dan 03 sudah dikirim untuk direview. Kuota revisi Feed Promo 02 tersisa 2 dari 3." }],
    placeholder: "Contoh: Feed promo yang kedua logonya kurang besar, warnanya terlalu gelap.",
    context: "Kamu mencatat revisi klien untuk Feed Promo 01, 02, dan 03. Desainernya Lala. Feed Promo 02 sudah memakai 1 dari 3 kuota revisi, dua feed lain belum memakai kuota. Tautkan revisi ke file yang tepat, tulis poinnya bernomor, dan minta konfirmasi. Setelah klien mengonfirmasi, teruskan ke Lala dengan target besok pukul 15.00 dan sebutkan sisa kuota.",
  },
];

// ── Leads from WhatsApp ads ──────────────────────────────────────────────────

export type LeadQuality = "qualified" | "review" | "notTarget";

export type WaLead = { name: string; phone: string; campaign: string; message: string; score: number };

export const WA_FUNNEL = { spend: 3_700_000, clicks: 1240, chats: 186 };

export const WA_LEADS: WaLead[] = [
  { name: "Sinta", phone: "0812-xxxx-4471", campaign: "Traffic Promo Kopi Susu", message: "Kak, minimal order untuk kantor berapa? Butuh 200 cup per minggu.", score: 92 },
  { name: "Budi Hartono", phone: "0857-xxxx-2210", campaign: "Traffic Promo Kopi Susu", message: "Bisa kirim ke Bandung? Saya punya 3 cabang cafe.", score: 88 },
  { name: "Maya", phone: "0813-xxxx-9034", campaign: "Traffic Paket Kantor", message: "Tolong kirim price list paket kantor.", score: 74 },
  { name: "Rizal", phone: "0821-xxxx-1187", campaign: "Traffic Paket Kantor", message: "Ada promo apa bulan ini?", score: 55 },
  { name: "Dewi", phone: "0878-xxxx-6602", campaign: "Traffic Promo Kopi Susu", message: "Ini jual biji kopi juga?", score: 48 },
  { name: "Tanpa nama", phone: "0896-xxxx-3345", campaign: "Traffic Promo Kopi Susu", message: "Halo", score: 22 },
  { name: "Agus", phone: "0838-xxxx-7719", campaign: "Traffic Paket Kantor", message: "Ada lowongan kerja barista?", score: 8 },
];

export function leadQuality(score: number): LeadQuality {
  if (score >= 70) return "qualified";
  if (score >= 40) return "review";
  return "notTarget";
}

// ── AI ad recommendations ────────────────────────────────────────────────────

export type AdRecommendation = { id: string; campaign: string; action: string; why: string; impact: string; confidence: number };

export const AD_RECOMMENDATIONS: AdRecommendation[] = [
  { id: "pause", campaign: "Traffic Promo Kopi Susu · Materi B", action: "Pause materi B", why: "Biaya per hasil naik 42% dalam 3 hari, frekuensi sudah 3,8.", impact: "Hemat sekitar Rp 180.000 per hari", confidence: 91 },
  { id: "budget", campaign: "Traffic Paket Kantor", action: "Naikkan budget harian dari Rp 150.000 ke Rp 200.000", why: "Biaya per chat 28% di bawah rata-rata selama 7 hari.", impact: "Tambahan sekitar 6 chat per hari", confidence: 78 },
  { id: "audience", campaign: "Awareness Brand Q4", action: "Kecualikan audiens yang sudah melihat iklan 5 kali", why: "22% tayangan jatuh ke orang yang sudah jenuh.", impact: "Jangkauan unik naik sekitar 15%", confidence: 66 },
  { id: "creative", campaign: "Traffic Promo Kopi Susu", action: "Buat 2 variasi baru dari materi A", why: "Materi A memimpin dengan CTR 2,4%, kampanye butuh penyegaran.", impact: "Menahan kenaikan biaya minggu depan", confidence: 59 },
];

// ── Marketplace competitors ──────────────────────────────────────────────────

export const MARKETPLACES = ["shopee", "tokopedia", "tiktok"] as const;
export type Marketplace = (typeof MARKETPLACES)[number];

export type MarketplaceBrand = { name: string; own?: boolean; price: number; rating: number; videos: number; units: Record<Marketplace, number> };

export const MARKETPLACE_BRANDS: MarketplaceBrand[] = [
  { name: "Kopi Nusantara", own: true, price: 89_000, rating: 4.8, videos: 6, units: { shopee: 620, tokopedia: 280, tiktok: 340 } },
  { name: "Kopi Lereng", price: 79_000, rating: 4.7, videos: 48, units: { shopee: 1240, tokopedia: 420, tiktok: 1460 } },
  { name: "Roastery Sawah", price: 99_000, rating: 4.9, videos: 19, units: { shopee: 900, tokopedia: 520, tiktok: 440 } },
  { name: "Kopi Bukit Raya", price: 69_000, rating: 4.5, videos: 22, units: { shopee: 1150, tokopedia: 600, tiktok: 700 } },
];

export function brandSales(brand: MarketplaceBrand, market: Marketplace | "all") {
  const units = market === "all" ? MARKETPLACES.reduce((sum, m) => sum + brand.units[m], 0) : brand.units[market];
  return { units, revenue: units * brand.price };
}

export const MARKETPLACE_INSIGHTS = [
  "Kopi Lereng menjual 2,5 kali lebih banyak dengan harga 11% lebih murah.",
  "Selisih terbesar ada di TikTok Shop: Kopi Lereng 1.460 unit, Kopi Nusantara 340 unit. Kopi Lereng juga memproduksi 48 video, Kopi Nusantara 6.",
  "Rating Kopi Nusantara 4,8 sudah di atas Kopi Lereng. Saran AI: pertahankan harga dan tambah video ulasan di TikTok Shop sebelum memberi diskon.",
];

// ── Design recap ─────────────────────────────────────────────────────────────

export type DesignStatus = "done" | "onTrack" | "atRisk" | "late";
export type DesignOrder = { client: string; item: string; target: number; uploaded: number; approved: number; daysLeft: number; owner: string };

/** Files one designer can realistically finish per day; used for the at-risk check. */
export const FILES_PER_DAY = 2;

export const DESIGN_ORDERS: DesignOrder[] = [
  { client: "Kopi Nusantara", item: "Feed November", target: 12, uploaded: 12, approved: 12, daysLeft: 3, owner: "Lala" },
  { client: "Kopi Nusantara", item: "Story Promo", target: 8, uploaded: 3, approved: 2, daysLeft: 5, owner: "Lala" },
  { client: "Adiharjo Property", item: "Feed Unit Baru", target: 10, uploaded: 2, approved: 1, daysLeft: 2, owner: "Bima" },
  { client: "Adiharjo Property", item: "Banner Pameran", target: 4, uploaded: 3, approved: 3, daysLeft: -1, owner: "Bima" },
  { client: "Bali Villa Escapes", item: "Carousel Villa", target: 6, uploaded: 5, approved: 4, daysLeft: 4, owner: "Citra" },
];

export function designStatus(o: Pick<DesignOrder, "target" | "uploaded" | "approved" | "daysLeft">): DesignStatus {
  if (o.approved >= o.target) return "done";
  if (o.daysLeft < 0) return "late";
  return o.target - o.uploaded > o.daysLeft * FILES_PER_DAY ? "atRisk" : "onTrack";
}

export type RevisionRow = { file: string; client: string; note: string; source: string; round: string; owner: string; state: string };

export const REVISIONS: RevisionRow[] = [
  { file: "Feed Promo 02", client: "Kopi Nusantara", note: "Perbesar logo, cerahkan warna", source: "Grup WhatsApp", round: "2 dari 3", owner: "Lala", state: "Dikerjakan" },
  { file: "Feed Unit Baru 01", client: "Adiharjo Property", note: "Ganti harga menjadi mulai 850 juta", source: "Grup WhatsApp", round: "1 dari 3", owner: "Bima", state: "Baru" },
  { file: "Carousel Villa 03", client: "Bali Villa Escapes", note: "Pakai foto kolam dari folder aset", source: "Dashboard", round: "3 dari 3", owner: "Citra", state: "Kuota habis" },
];

export const VIDEO_STEPS = [
  { title: "Brief masuk", tool: "Model teks, misalnya DeepSeek", note: "AI menyusun naskah dan daftar potongan dari brief.", credits: 2 },
  { title: "Rough cut", tool: "Layanan pengedit video", note: "Sistem memotong klip mentah mengikuti naskah.", credits: 18 },
  { title: "Subtitle", tool: "Transkripsi suara", note: "Subtitle Indonesia dibuat dari audio klip.", credits: 4 },
  { title: "Selesaikan di Premiere", tool: "Adobe Premiere", note: "Editor membuka timeline hasil ekspor, lalu menyelesaikan warna dan audio.", credits: 0 },
];

// ── Integrations ─────────────────────────────────────────────────────────────

export type IntegrationState = "live" | "partial" | "planned";
export type Integration = { name: string; use: string; state: IntegrationState; note: string };

/** `state` reflects the codebase today, so this list stays honest during a demo. */
export const INTEGRATIONS: Integration[] = [
  { name: "Meta (Instagram, Facebook, Ads)", use: "Data akun dan iklan", state: "live", note: "Membuat kampanye butuh izin ads_management dari Meta." },
  { name: "Google Search Console dan GA4", use: "Data SEO dan trafik", state: "live", note: "Terhubung per properti klien." },
  { name: "DataForSEO", use: "Riset kata kunci dan backlink", state: "live", note: "Dipakai SEO Suite." },
  { name: "OpenRouter", use: "Akses model teks dan gambar AI", state: "live", note: "Satu kunci untuk banyak model." },
  { name: "ChatGPT dan DeepSeek", use: "Naskah, caption, analisa", state: "partial", note: "Bisa lewat OpenRouter, biaya mengikuti pemakaian." },
  { name: "WhatsApp gateway", use: "Chatbot dan inbox", state: "partial", note: "Kode siap, layanan gateway belum dijalankan." },
  { name: "WhatsApp Cloud API", use: "Asal iklan dan nomor per klien", state: "planned", note: "Perlu nomor WA Business tiap klien." },
  { name: "Data marketplace, misalnya Kalodata", use: "Pembanding kompetitor", state: "planned", note: "Cakupan, harga, dan akses API perlu dibandingkan." },
  { name: "Freepik", use: "Aset dan generate gambar", state: "planned", note: "Perlu dicek akses API pada paket yang dipakai." },
  { name: "Adobe Premiere", use: "Finishing video", state: "planned", note: "Rencana lewat ekspor timeline ke editor." },
];

// ── Team ─────────────────────────────────────────────────────────────────────

export type TeamState = "working" | "review" | "meeting" | "offline";
export type TeamMember = { name: string; role: string; state: TeamState; doing: string; client: string; updated: string; doneToday: number; onTime: number; firstPass: number; rating: number };

export const TEAM: TeamMember[] = [
  { name: "Lala", role: "Desainer", state: "working", doing: "Story Promo 04", client: "Kopi Nusantara", updated: "12 menit lalu", doneToday: 3, onTime: 96, firstPass: 82, rating: 4.7 },
  { name: "Bima", role: "Desainer", state: "working", doing: "Feed Unit Baru 03", client: "Adiharjo Property", updated: "25 menit lalu", doneToday: 1, onTime: 98, firstPass: 41, rating: 3.8 },
  { name: "Citra", role: "Desainer", state: "review", doing: "Carousel Villa 05", client: "Bali Villa Escapes", updated: "1 jam lalu", doneToday: 2, onTime: 84, firstPass: 88, rating: 4.9 },
  { name: "Dimas", role: "Copywriter", state: "working", doing: "Caption promo kopi susu", client: "Kopi Nusantara", updated: "8 menit lalu", doneToday: 5, onTime: 92, firstPass: 75, rating: 4.4 },
  { name: "Rangga", role: "Account", state: "meeting", doing: "Review bulanan klien", client: "Adiharjo Property", updated: "40 menit lalu", doneToday: 2, onTime: 90, firstPass: 70, rating: 4.5 },
  { name: "Nadia", role: "Editor video", state: "offline", doing: "Cuti sampai besok", client: "-", updated: "Kemarin", doneToday: 0, onTime: 88, firstPass: 64, rating: 4.2 },
];

/** Example weighting: revisions and client rating count more than punctuality. */
export function qualityScore(m: Pick<TeamMember, "onTime" | "firstPass" | "rating">): number {
  return Math.round(m.firstPass * 0.5 + (m.rating / 5) * 100 * 0.3 + m.onTime * 0.2);
}

// ── Report and buzzer ────────────────────────────────────────────────────────

export const REPORT_STEPS = ["Mengambil data konten", "Menghitung performa", "Menulis analisa AI", "Menyusun PDF"];

export const REPORT_PREVIEW = {
  title: "Laporan Konten Oktober, Kopi Nusantara",
  stats: [
    { label: "Konten terbit", value: "24 dari 24" },
    { label: "Jangkauan", value: "184 rb" },
    { label: "Engagement rate", value: "4,6%" },
    { label: "Follower baru", value: "+1.208" },
  ],
  best: "Reels resep kopi susu: 42 rb jangkauan, 3,1 rb simpan.",
  worst: "Feed info jam buka: 2,1 rb jangkauan, 18 interaksi.",
  next: "Tambah Reels resep menjadi 2 per minggu dan kurangi feed informasi.",
};

export type BuzzerAccount = { handle: string; tier: string; followers: number; posts: number; reach: number; engagement: number; suspicious: boolean; cost: number };

export const BUZZER_ACCOUNTS: BuzzerAccount[] = [
  { handle: "@kulinerjaksel.demo", tier: "Micro", followers: 48_000, posts: 3, reach: 31_000, engagement: 6.2, suspicious: false, cost: 1_500_000 },
  { handle: "@ngopiterus.demo", tier: "Nano", followers: 9_400, posts: 5, reach: 12_500, engagement: 8.4, suspicious: false, cost: 500_000 },
  { handle: "@infopromo.demo", tier: "Micro", followers: 86_000, posts: 2, reach: 6_200, engagement: 0.4, suspicious: true, cost: 1_200_000 },
  { handle: "@ceritakopi.demo", tier: "Macro", followers: 210_000, posts: 1, reach: 74_000, engagement: 3.1, suspicious: false, cost: 4_500_000 },
];

/** Rupiah per 1,000 people reached. */
export function costPerThousand(a: Pick<BuzzerAccount, "cost" | "reach">): number {
  return Math.round((a.cost / a.reach) * 1000);
}
