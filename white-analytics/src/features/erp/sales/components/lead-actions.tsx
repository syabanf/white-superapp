"use client";

import { Pencil, Plus, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { deleteAccountAction, deleteLeadAction } from "../actions";
import { ss } from "../strings";
import type { Account, CustomField, Lead } from "../types";
import { AccountFormDialog } from "./accounts";
import { DealFormDialog, type DealFormRefs } from "./deal-form";
import { LeadFormDialog } from "./lead-form";
import { ConfirmAction } from "./ui";

/** Edit, delete and "create deal" buttons on the lead page. */
export function LeadActions({
  lead,
  refs,
  leadFields,
}: {
  lead: Lead;
  refs: DealFormRefs;
  leadFields: CustomField[];
}) {
  const router = useRouter();
  return (
    <>
      <ConfirmAction
        title={ss.leads.deleteTitle}
        action={() => deleteLeadAction(lead.id)}
        onDone={() => router.push("/erp/sales/leads")}
        trigger={
          <Button variant="ghost" size="sm">
            <Trash2 className="size-3.5" /> {ss.delete}
          </Button>
        }
      />
      <LeadFormDialog
        lead={lead}
        owners={refs.owners}
        accounts={refs.accounts}
        customFields={leadFields}
        trigger={
          <Button variant="outline" size="sm">
            <Pencil className="size-3.5" /> {ss.edit}
          </Button>
        }
      />
      <DealFormDialog
        deal={null}
        refs={refs}
        defaults={{
          leadId: lead.id,
          accountId: lead.accountId,
          title: lead.orgName,
          ownerUserId: lead.ownerUserId,
        }}
        trigger={
          <Button size="sm">
            <Plus className="size-3.5" /> {ss.leads.createDeal}
          </Button>
        }
      />
    </>
  );
}

/** Edit and delete buttons on the account page. */
export function AccountActions({ account, owners }: { account: Account; owners: DealFormRefs["owners"] }) {
  const router = useRouter();
  return (
    <>
      <ConfirmAction
        title={ss.accounts.deleteTitle}
        action={() => deleteAccountAction(account.id)}
        onDone={() => router.push("/erp/sales/leads?tab=accounts")}
        trigger={
          <Button variant="ghost" size="sm">
            <Trash2 className="size-3.5" /> {ss.delete}
          </Button>
        }
      />
      <AccountFormDialog
        account={account}
        owners={owners}
        trigger={
          <Button variant="outline" size="sm">
            <Pencil className="size-3.5" /> {ss.edit}
          </Button>
        }
      />
    </>
  );
}
