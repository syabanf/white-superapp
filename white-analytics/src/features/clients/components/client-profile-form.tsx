"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { updateClientAction, type ClientFormInput } from "@/features/clients/actions";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";
import { ClientFields, type ClientFormValues, type FieldErrors } from "./client-fields";
import type { EditableClient } from "./client-form-dialog";

/** Settings → Profil klien (same fields as the edit dialog, inline card). */
export function ClientProfileForm({ client, canManage }: { client: EditableClient; canManage: boolean }) {
  const router = useRouter();
  const [values, setValues] = React.useState<ClientFormValues>({ ...client });
  const [errors, setErrors] = React.useState<FieldErrors>(undefined);
  const [pending, startTransition] = React.useTransition();

  const setField = <K extends keyof ClientFormValues>(field: K, value: ClientFormValues[K]) => {
    setValues((prev) => ({ ...prev, [field]: value }));
  };

  const submit = () => {
    const payload: ClientFormInput = {
      name: values.name,
      slug: values.slug,
      description: values.description,
      industry: values.industry,
      websiteUrl: values.websiteUrl.trim(),
      currency: values.currency as ClientFormInput["currency"],
      timezone: values.timezone as ClientFormInput["timezone"],
    };
    startTransition(async () => {
      const res = await updateClientAction(client.id, payload);
      if (!res.ok) {
        setErrors(res.fieldErrors);
        toast.error(res.error);
        return;
      }
      setErrors(undefined);
      toast.success(t.clients.updated);
      if (res.data.slug !== client.slug) {
        router.replace(`/clients/${res.data.slug}/settings`);
      }
    });
  };

  return (
    <Card>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
        className="contents"
      >
        <CardContent>
          <ClientFields values={values} onChange={setField} errors={errors} disabled={pending || !canManage} idPrefix="profile" />
        </CardContent>
        <CardFooter className="justify-between gap-2">
          <p className="text-xs text-muted-foreground">{canManage ? tc.settings.profileDesc : tc.settings.readOnly}</p>
          {canManage ? (
            <Button type="submit" size="sm" disabled={pending || values.name.trim().length === 0}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {t.common.save}
            </Button>
          ) : null}
        </CardFooter>
      </form>
    </Card>
  );
}
