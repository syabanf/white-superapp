import { Clapperboard, FileBarChart, Megaphone, MessageCircle, PlugZap, Store, UserCheck, Users, type LucideIcon } from "lucide-react";

export type ShowcaseTopic = {
  id: string;
  title: string;
  summary: string;
  icon: LucideIcon;
  /** Client questions this demo answers. */
  questions: string[];
  /** What already works in the product today, shown beside the simulation. */
  live: string;
};

export const SHOWCASE_TOPICS: ShowcaseTopic[] = [
  {
    id: "whatsapp",
    title: "AI di WhatsApp",
    summary: "Reminder customer, edit quotation lewat chat, notulen grup, dan revisi klien.",
    icon: MessageCircle,
    questions: ["Chatbot reminder ke WhatsApp", "Edit quotation lewat chat", "Notulen dan auto reminder tim", "Revisi dari grup klien"],
    live: "Chatbot lead di widget website sudah berjalan. Pesan keluar ke WhatsApp menunggu gateway aktif.",
  },
  {
    id: "leads",
    title: "Lead WA dari iklan",
    summary: "Chat dari iklan traffic masuk otomatis sebagai lead dengan skor kualitas.",
    icon: UserCheck,
    questions: ["Connect ke nomor WA customer", "Rekap tanpa menunggu klien", "Kualitas lead traffic dan awareness"],
    live: "Modul Leads, skor lead, dan pencatatan lead oleh chatbot sudah ada. Asal iklan pada chat WA belum tercatat.",
  },
  {
    id: "iklan",
    title: "Rekomendasi iklan AI",
    summary: "AI mengusulkan tindakan, tim menyetujui, sistem mengeksekusi.",
    icon: Megaphone,
    questions: ["AI merekomendasikan dan mengeksekusi iklan", "Review setelah iklan berjalan"],
    live: "Campaign Builder AI, peringatan harian, dan analisa mingguan sudah ada. Eksekusi ke Meta menunggu izin ads_management.",
  },
  {
    id: "marketplace",
    title: "Kompetitor marketplace",
    summary: "Bandingkan harga, penjualan, dan konten dengan kompetitor per marketplace.",
    icon: Store,
    questions: ["Koneksi ke Kalodata atau alternatif lebih murah", "Data pembanding marketplace"],
    live: "Pembanding kompetitor Instagram dan domain SEO sudah ada. Data marketplace belum terhubung.",
  },
  {
    id: "desain",
    title: "Desain, revisi, dan video",
    summary: "Rekap file sesuai jumlah dan deadline, antrean revisi, serta alur edit video.",
    icon: Clapperboard,
    questions: ["Rekap file desain per quantity dan deadline", "Revisi klien masuk dashboard", "Auto edit ke Adobe Premiere", "Biaya token edit video"],
    live: "Papan konten, tugas desain, dan Studio Desain AI sudah ada. Edit video AI belum ada.",
  },
  {
    id: "tim",
    title: "Aktivitas dan kualitas tim",
    summary: "Lihat tim remote sedang mengerjakan apa, dan nilai kualitas selain ketepatan waktu.",
    icon: Users,
    questions: ["Tim Mixist lagi ngapain", "Penilaian kualitas, bukan hanya on-time"],
    live: "Task, logbook, rekap kerja, KPI, dan review kinerja sudah ada di ERP. Skor kualitas gabungan ini masih contoh.",
  },
  {
    id: "report",
    title: "Report konten dan buzzer",
    summary: "Coba buat report otomatis, lalu pantau performa akun buzzer.",
    icon: FileBarChart,
    questions: ["Auto buat report", "Fitur buzzer"],
    live: "Laporan PDF dan laporan otomatis sudah ada di menu Laporan. Pemantauan buzzer belum ada.",
  },
  {
    id: "integrasi",
    title: "Daftar integrasi",
    summary: "Layanan yang sudah terhubung, sebagian siap, dan masih rencana.",
    icon: PlugZap,
    questions: ["List fitur atau API yang bisa dikoneksikan"],
    live: "Status pada daftar ini mengikuti kode saat ini.",
  },
];
