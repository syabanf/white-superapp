"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { createClientAction, updateClientAction, type ClientFormInput } from "@/features/clients/actions";
import type { ActionResult } from "@/lib/action-result";
import { slugify } from "@/lib/format";
import { t } from "@/i18n/id";
import { ClientFields, emptyClientForm, type ClientFormValues, type FieldErrors } from "./client-fields";

export type EditableClient = ClientFormValues & { id: string };

/**
 * Create / edit client dialog. Controlled (open/onOpenChange) or uncontrolled
 * with a trigger. On create the server action redirects to the new dashboard.
 * Form state lives in the body component, which unmounts on close — no stale state.
 */
export function ClientFormDialog({
  mode,
  client,
  trigger,
  open: openProp,
  onOpenChange,
  defaultOpen = false,
  clearQueryOnClose = false,
}: {
  mode: "create" | "edit";
  client?: EditableClient;
  trigger?: React.ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  defaultOpen?: boolean;
  /** replace URL (drop ?new=1) when the auto-opened dialog closes */
  clearQueryOnClose?: boolean;
}) {
  const router = useRouter();
  const [openState, setOpenState] = React.useState(defaultOpen);
  const open = openProp ?? openState;
  const setOpen = React.useCallback(
    (next: boolean) => {
      onOpenChange?.(next);
      if (openProp === undefined) setOpenState(next);
      if (!next && clearQueryOnClose) router.replace("/clients");
    },
    [onOpenChange, openProp, clearQueryOnClose, router],
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {trigger ? <DialogTrigger asChild>{trigger}</DialogTrigger> : null}
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{mode === "create" ? t.clients.new : t.clients.edit}</DialogTitle>
          <DialogDescription>{t.clients.subtitle}</DialogDescription>
        </DialogHeader>
        <ClientFormBody key={client?.id ?? "new"} mode={mode} client={client} onClose={() => setOpen(false)} />
      </DialogContent>
    </Dialog>
  );
}

function ClientFormBody({ mode, client, onClose }: { mode: "create" | "edit"; client?: EditableClient; onClose: () => void }) {
  const router = useRouter();
  const [values, setValues] = React.useState<ClientFormValues>(() => (client ? { ...client } : emptyClientForm()));
  const [slugTouched, setSlugTouched] = React.useState(mode === "edit");
  const [errors, setErrors] = React.useState<FieldErrors>(undefined);
  const [pending, startTransition] = React.useTransition();

  const setField = <K extends keyof ClientFormValues>(field: K, value: ClientFormValues[K]) => {
    setValues((prev) => {
      const next = { ...prev, [field]: value };
      if (field === "name" && !slugTouched) next.slug = slugify(String(value));
      return next;
    });
    if (field === "slug") setSlugTouched(true);
  };

  const submit = () => {
    const payload: ClientFormInput = {
      name: values.name,
      slug: values.slug || slugify(values.name),
      description: values.description,
      industry: values.industry,
      websiteUrl: values.websiteUrl.trim(),
      currency: values.currency as ClientFormInput["currency"],
      timezone: values.timezone as ClientFormInput["timezone"],
    };
    startTransition(async () => {
      const res = (await (mode === "create"
        ? createClientAction(payload)
        : updateClientAction(client!.id, payload))) as ActionResult<{ slug: string }> | undefined;
      if (!res) {
        // create redirected server-side
        toast.success(t.clients.created);
        return;
      }
      if (!res.ok) {
        setErrors(res.fieldErrors);
        toast.error(res.error);
        return;
      }
      toast.success(mode === "create" ? t.clients.created : t.clients.updated);
      onClose();
      router.refresh();
    });
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
      className="space-y-4"
    >
      <ClientFields values={values} onChange={setField} errors={errors} disabled={pending} idPrefix={`client-${mode}`} />
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          {t.common.cancel}
        </Button>
        <Button type="submit" disabled={pending || values.name.trim().length === 0}>
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {mode === "create" ? t.common.create : t.common.save}
        </Button>
      </DialogFooter>
    </form>
  );
}
