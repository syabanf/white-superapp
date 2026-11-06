import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";

export type CompanyProfile = {
  legalName: string;
  address: string;
  email: string;
  phone: string;
  npwp: string;
  bankName: string;
  bankAccountNo: string;
  bankAccountName: string;
  bankLine: string;
};

export type MailStatus = { configured: boolean; source: "DB" | "ENV" | "NONE"; from: string; secretMasked: string; verifiedAt?: string };

export async function getCompanyProfile(): Promise<CompanyProfile> {
  const r = await gatewayFetch<Partial<CompanyProfile>>("/v1/finance/company-profile");
  return {
    legalName: r.legalName ?? "",
    address: r.address ?? "",
    email: r.email ?? "",
    phone: r.phone ?? "",
    npwp: r.npwp ?? "",
    bankName: r.bankName ?? "",
    bankAccountNo: r.bankAccountNo ?? "",
    bankAccountName: r.bankAccountName ?? "",
    bankLine: r.bankLine ?? "",
  };
}

type Status = { key: string; configured?: boolean; source?: MailStatus["source"]; publicId?: string; secretMasked?: string; verifiedAt?: string };

export async function getMailStatus(): Promise<MailStatus> {
  const r = await gatewayFetch<{ statuses?: Status[] }>("/v1/integrations");
  const s = r.statuses?.find((x) => x.key === "RESEND");
  return { configured: s?.configured ?? false, source: s?.source ?? "NONE", from: s?.publicId ?? "", secretMasked: s?.secretMasked ?? "", verifiedAt: s?.verifiedAt };
}
