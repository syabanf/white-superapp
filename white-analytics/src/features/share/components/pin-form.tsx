"use client";

import * as React from "react";
import { useActionState } from "react";
import { KeyRound, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { verifySharePinForm } from "@/features/share/actions";
import type { ActionResult } from "@/lib/action-result";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

export function SharePinForm({ slug, clientName }: { slug: string; clientName: string }) {
  const action = verifySharePinForm.bind(null, slug);
  const [state, formAction, pending] = useActionState<ActionResult<never> | null, FormData>(action, null);

  return (
    <div className="flex flex-1 items-center justify-center px-4 py-16">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-full bg-muted">
            <KeyRound className="size-5 text-muted-foreground" />
          </div>
          <CardTitle className="text-lg">{t.share.pinTitle}</CardTitle>
          <CardDescription>
            {clientName} · {t.share.pinDesc}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form action={formAction} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="share-pin">{t.reports.sharePin}</Label>
              <Input
                id="share-pin"
                name="pin"
                type="password"
                inputMode="numeric"
                autoComplete="one-time-code"
                pattern="\d{6}"
                maxLength={6}
                required
                placeholder={rs.share.pinPlaceholder}
                className="text-center text-lg tracking-[0.5em]"
                autoFocus
              />
            </div>
            {state && !state.ok ? <p className="text-sm text-destructive">{state.error}</p> : null}
            <Button type="submit" className="w-full" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null} {t.share.unlock}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
