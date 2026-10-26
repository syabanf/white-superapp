"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { saveDealAction, type DealValues } from "../actions";
import { FORECAST_CATEGORY, SERVICE_LINE, opts, ss } from "../strings";
import type { CustomField, Deal, Pipeline } from "../types";
import { CustomFieldInputs } from "./custom-fields";
import { ChipToggle, type FieldErrors, Fld, FormDialog, FormFooter, type Option, Picker, SimpleSelect } from "./ui";

const s = ss.pipeline;

export type DealFormRefs = {
  owners: Option[];
  leads: Option[];
  accounts: Option[];
  pipelines: Pipeline[];
  customFields: CustomField[];
};

export function DealFormDialog({
  deal,
  refs,
  defaults,
  trigger,
}: {
  deal: Deal | null;
  refs: DealFormRefs;
  defaults?: Partial<DealValues>;
  trigger: React.ReactNode;
}) {
  return (
    <FormDialog trigger={trigger} title={deal ? s.editTitle : s.addTitle} size="lg">
      {(close) => <DealForm deal={deal} refs={refs} defaults={defaults} close={close} />}
    </FormDialog>
  );
}

function DealForm({
  deal,
  refs,
  defaults,
  close,
}: {
  deal: Deal | null;
  refs: DealFormRefs;
  defaults?: Partial<DealValues>;
  close: () => void;
}) {
  const router = useRouter();
  const defaultPipeline = refs.pipelines.find((p) => p.isDefault) ?? refs.pipelines[0];
  const [v, setV] = React.useState<DealValues>({
    leadId: deal?.leadId ?? "",
    accountId: deal?.accountId ?? "",
    title: deal?.title ?? "",
    serviceLines: deal?.serviceLines ?? [],
    pipelineId: deal?.pipelineId ?? defaultPipeline?.id ?? "",
    stageId: deal?.stageId ?? "",
    valueEstimate: deal?.valueEstimate ?? "",
    ownerUserId: deal?.ownerUserId ?? "",
    expectedCloseDate: deal?.expectedCloseDate ?? "",
    notes: deal?.notes ?? "",
    forecastCategory: deal && ["pipeline", "best_case", "commit"].includes(deal.forecastCategory) ? deal.forecastCategory : "",
    custom: deal?.custom ?? {},
    ...defaults,
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof DealValues>(k: K, val: DealValues[K]) => setV((p) => ({ ...p, [k]: val }));
  const pipeline = refs.pipelines.find((p) => p.id === v.pipelineId);
  const openStages = (pipeline?.stages ?? []).filter((st) => st.isActive && !st.isWon && !st.isLost);
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveDealAction(deal?.id ?? null, v), {
          success: ss.saved,
          onOk: (saved) => {
            close();
            if (!deal) router.push(`/erp/sales/pipeline/${saved.id}`);
          },
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <Fld label={`${s.dealTitle} *`} htmlFor="deal-title" error={errors?.title} className="sm:col-span-2">
        <Input id="deal-title" value={v.title} onChange={(e) => set("title", e.target.value)} />
      </Fld>
      <Fld label={s.lead} htmlFor="deal-lead">
        <Picker id="deal-lead" value={v.leadId ?? ""} onChange={(x) => set("leadId", x)} options={refs.leads} allowEmpty={ss.dash} />
      </Fld>
      <Fld label={s.account} htmlFor="deal-account">
        <Picker id="deal-account" value={v.accountId ?? ""} onChange={(x) => set("accountId", x)} options={refs.accounts} allowEmpty={ss.dash} />
      </Fld>
      <Fld label={s.serviceLines} className="sm:col-span-2">
        <ChipToggle value={v.serviceLines ?? []} onChange={(x) => set("serviceLines", x)} options={opts(SERVICE_LINE)} />
      </Fld>
      {!deal ? (
        <>
          {refs.pipelines.length > 1 ? (
            <Fld label="Pipeline" htmlFor="deal-pipeline">
              <SimpleSelect
                id="deal-pipeline"
                value={v.pipelineId ?? ""}
                onChange={(x) => setV((p) => ({ ...p, pipelineId: x, stageId: "" }))}
                options={refs.pipelines.map((p) => ({ value: p.id, label: p.name }))}
              />
            </Fld>
          ) : null}
          <Fld label={s.stage} htmlFor="deal-stage">
            <SimpleSelect
              id="deal-stage"
              value={v.stageId ?? ""}
              onChange={(x) => set("stageId", x)}
              options={openStages.map((st) => ({ value: st.id, label: st.name }))}
              placeholder={openStages[0]?.name}
            />
          </Fld>
        </>
      ) : null}
      <Fld label={s.valueEstimate} htmlFor="deal-value" error={errors?.valueEstimate}>
        <MoneyInput id="deal-value" value={v.valueEstimate ?? ""} onChange={(x) => set("valueEstimate", x)} />
      </Fld>
      <Fld label={s.expectedClose} htmlFor="deal-close" error={errors?.expectedCloseDate}>
        <DatePicker id="deal-close" value={v.expectedCloseDate ?? ""} onChange={(x) => set("expectedCloseDate", x)} clearable />
      </Fld>
      <Fld label={ss.owner} htmlFor="deal-owner">
        <Picker id="deal-owner" value={v.ownerUserId ?? ""} onChange={(x) => set("ownerUserId", x)} options={refs.owners} allowEmpty={ss.noOwner} />
      </Fld>
      <Fld label={s.forecastCategory} htmlFor="deal-fc">
        <SimpleSelect
          id="deal-fc"
          value={v.forecastCategory ?? ""}
          onChange={(x) => set("forecastCategory", x)}
          options={opts(FORECAST_CATEGORY).filter((o) => !o.value.startsWith("closed"))}
          emptyLabel={s.forecastAuto}
        />
      </Fld>
      <Fld label={ss.notes} htmlFor="deal-notes" className="sm:col-span-2">
        <Textarea id="deal-notes" rows={3} value={v.notes ?? ""} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <CustomFieldInputs fields={refs.customFields} value={v.custom ?? {}} onChange={(c) => set("custom", c)} />
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
