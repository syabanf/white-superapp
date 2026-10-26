import "server-only";
import { listAccounts, listCustomFields, listLeads, listOwners, listPipelines } from "./queries";

const settle = async <T>(p: Promise<T>, fallback: T): Promise<T> => p.catch(() => fallback);

/** Owners as picker options; empty when the call fails. */
export async function ownerOptions() {
  const owners = await settle(listOwners(), []);
  return owners.map((o) => ({ value: o.id, label: o.name || o.email }));
}

/** Pickers and pipelines that the deal form needs. Failures degrade to empty lists. */
export async function dealFormRefs() {
  const [owners, leads, accounts, pipelines, customFields] = await Promise.all([
    ownerOptions(),
    settle(listLeads({ pageSize: 500 }), { leads: [], total: 0 }),
    settle(listAccounts({ limit: 500 }), []),
    settle(listPipelines(), []),
    settle(listCustomFields("deal"), []),
  ]);
  return {
    owners,
    leads: leads.leads.map((l) => ({ value: l.id, label: `${l.orgName} · ${l.picName}` })),
    accounts: accounts.map((a) => ({ value: a.id, label: a.name })),
    pipelines,
    customFields,
  };
}
