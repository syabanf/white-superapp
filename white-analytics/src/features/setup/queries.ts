import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { getIntegrationStatuses, getSetupState, type IntegrationStatus, type SetupState } from "@/lib/integrations";
import { parseMetaReview, type IntegrationKey, type MetaReviewState } from "@/features/setup/integrations";

export type SetupOverview = {
  statuses: Record<IntegrationKey, IntegrationStatus>;
  metaReview: MetaReviewState;
  setup: SetupState;
  appUrl: string;
  counts: { trackedKeywords: number; competitorDomains: number; clients: number; socialAccounts: number };
};

export const getSetupOverview = cache(async (): Promise<SetupOverview> => {
  const [statuses, setup, trackedKeywords, competitorDomains, clients, socialAccounts] = await Promise.all([
    getIntegrationStatuses(),
    getSetupState(),
    db.trackedKeyword.count(),
    db.competitorDomain.count(),
    db.client.count(),
    db.socialAccount.count(),
  ]);
  return {
    statuses,
    metaReview: parseMetaReview(statuses.META.meta),
    setup,
    appUrl: (process.env.NEXT_PUBLIC_APP_URL ?? "").replace(/\/$/, ""),
    counts: { trackedKeywords, competitorDomains, clients, socialAccounts },
  };
});
