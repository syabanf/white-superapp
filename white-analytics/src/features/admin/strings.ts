/**
 * Extra UI strings for the admin feature (manajemen user).
 * Kept out of src/i18n/id.ts so parallel modules don't collide.
 */
export const ta = {
  usersCount: "user",
  noClients: "Tidak ada akses klien",
  clientsTooltip: "Klien yang diakses",
  managerBadge: "Manajer",
  deleteTitle: "Hapus user",
  deleteDesc: (name: string) => `Hapus ${name}? Akses ke semua klien akan dicabut. Tindakan ini tidak dapat dibatalkan.`,
  nameRequired: "Nama wajib diisi",
  emailInvalid: "Email tidak valid",
  passwordMin: "Kata sandi minimal 8 karakter",
  cannotEditSelf: "Anda tidak dapat menonaktifkan atau menurunkan peran akun sendiri.",
  statusUpdated: "Status diperbarui",
  you: "Anda",
  makeManager: "Manajer",
  accessHint: "Centang klien yang boleh diakses. Aktifkan “Manajer” agar user dapat mengelola klien tsb.",
  adminAccessAll: "Peran Admin otomatis punya akses ke semua klien.",
  emptyUsers: "Belum ada user.",
} as const;
