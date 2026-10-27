"use client";

import * as React from "react";
import { Building2, Check, CircleDashed, SkipForward } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { slugify } from "@/lib/format";
import { cn } from "@/lib/utils";
import { createMarketingClientAction, updateOnboardingItemAction } from "../actions";
import { ONBOARDING_KIND, lbl, ss } from "../strings";
import type { Onboarding, OnboardingItem } from "../types";
import { Fld, FormDialog, FormFooter, type Option, Picker } from "./ui";

const s = ss.onboarding;

/** Checklist with status, owner and due date per item. */
export function OnboardingChecklist({ items, owners, canWrite }: { items: OnboardingItem[]; owners: Option[]; canWrite: boolean }) {
  return (
    <ul className="divide-y">
      {items.map((it) => (
        <ChecklistRow key={it.id} item={it} owners={owners} canWrite={canWrite} />
      ))}
    </ul>
  );
}

function ChecklistRow({ item, owners, canWrite }: { item: OnboardingItem; owners: Option[]; canWrite: boolean }) {
  const { pending, run } = useErpAction();
  const update = (patch: { status?: string; ownerUserId?: string; dueDate?: string }) =>
    void run(() => updateOnboardingItemAction(item.id, patch), { success: s.itemUpdated });
  const done = item.status === "done";
  const skipped = item.status === "skipped";
  return (
    <li className={cn("flex flex-col gap-2 px-(--card-pad) py-3 sm:flex-row sm:items-start", pending && "opacity-60")}>
      <div className="flex min-w-0 flex-1 items-start gap-3">
        <span
          className={cn(
            "mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border",
            done && "border-positive bg-positive text-white",
            skipped && "border-muted-foreground/40 text-muted-foreground",
          )}
        >
          {done ? <Check className="size-3" /> : skipped ? <SkipForward className="size-3" /> : <CircleDashed className="size-3 text-muted-foreground" />}
        </span>
        <div className="min-w-0">
          <p className={cn("text-sm font-medium", (done || skipped) && "text-muted-foreground", skipped && "line-through")}>{item.title}</p>
          <p className="text-xs text-muted-foreground">
            {lbl(ONBOARDING_KIND, item.kind)}
            {item.description ? ` · ${item.description}` : ""}
          </p>
        </div>
      </div>
      {canWrite ? (
        <div className="flex flex-wrap items-center gap-2 sm:justify-end">
          <div className="w-40">
            <Picker value={item.ownerUserId} onChange={(v) => update({ ownerUserId: v })} options={owners} allowEmpty={ss.noOwner} disabled={pending} />
          </div>
          <div className="w-40">
            <DatePicker value={item.dueDate} onChange={(v) => update({ dueDate: v })} placeholder={s.due} disabled={pending} />
          </div>
          {done || skipped ? (
            <Button size="xs" variant="outline" disabled={pending} onClick={() => update({ status: "todo" })}>
              {s.todo}
            </Button>
          ) : (
            <>
              <Button size="xs" variant="ghost" disabled={pending} onClick={() => update({ status: "skipped" })}>
                {s.skip}
              </Button>
              <Button size="xs" disabled={pending} onClick={() => update({ status: "done" })}>
                <Check className="size-3.5" /> {s.done}
              </Button>
            </>
          )}
        </div>
      ) : null}
    </li>
  );
}

/** ADMIN: creates the client in the marketing platform. */
export function CreateClientDialog({ onboarding }: { onboarding: Onboarding }) {
  return (
    <FormDialog
      title={s.createClientTitle}
      trigger={
        <Button>
          <Building2 /> {s.createClient}
        </Button>
      }
    >
      {(close) => <CreateClientForm onboarding={onboarding} close={close} />}
    </FormDialog>
  );
}

function CreateClientForm({ onboarding, close }: { onboarding: Onboarding; close: () => void }) {
  const [name, setName] = React.useState(onboarding.brief.brand_name || onboarding.clientName);
  const [slug, setSlug] = React.useState(slugify(onboarding.brief.brand_name || onboarding.clientName));
  const [websiteUrl, setWebsiteUrl] = React.useState(onboarding.brief.website ?? "");
  const [industry, setIndustry] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createMarketingClientAction(onboarding.id, { name, slug, websiteUrl, industry }), {
          success: s.clientCreated,
          onOk: close,
        });
      }}
    >
      <Fld label={`${s.clientName} *`} htmlFor="mc-name">
        <Input id="mc-name" value={name} onChange={(e) => setName(e.target.value)} />
      </Fld>
      <Fld label={s.slug} htmlFor="mc-slug">
        <Input id="mc-slug" value={slug} onChange={(e) => setSlug(slugify(e.target.value))} />
      </Fld>
      <Fld label={s.website} htmlFor="mc-web">
        <Input id="mc-web" type="url" value={websiteUrl} onChange={(e) => setWebsiteUrl(e.target.value)} placeholder="https://" />
      </Fld>
      <Fld label={s.industry} htmlFor="mc-industry">
        <Input id="mc-industry" value={industry} onChange={(e) => setIndustry(e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} submitLabel={ss.create} />
      </div>
    </form>
  );
}
