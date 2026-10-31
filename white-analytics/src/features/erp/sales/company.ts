/**
 * Letterhead printed on quotations and invoices. Mirrors the Go renderer's
 * defaults (COMPANY_* there); override per deployment with NEXT_PUBLIC_COMPANY_*.
 * NPWP stays empty until it is configured, and the line is then hidden.
 */
export const COMPANY = {
  legalName: process.env.NEXT_PUBLIC_COMPANY_LEGAL_NAME || "PT Mixist Digital Kreatif",
  address: process.env.NEXT_PUBLIC_COMPANY_ADDRESS || "Jl. Senopati No. 8, Kebayoran Baru, Jakarta Selatan 12190",
  email: process.env.NEXT_PUBLIC_COMPANY_EMAIL || "hello@mixist.id",
  phone: process.env.NEXT_PUBLIC_COMPANY_PHONE || "+62 21 5000 8800",
  npwp: process.env.NEXT_PUBLIC_COMPANY_NPWP || "",
};
