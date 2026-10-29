"use client";

import * as React from "react";
import { toast } from "@/lib/toast";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { setClientDemoAction } from "@/features/clients/actions";
import { tc } from "@/features/clients/strings";

/** Settings → Mode data: sakelar klien demo (ADMIN only). */
export function DemoModeCard({ clientId, isDemo }: { clientId: string; isDemo: boolean }) {
  const [checked, setChecked] = React.useState(isDemo);
  const [pending, startTransition] = React.useTransition();

  const change = (next: boolean) => {
    setChecked(next);
    startTransition(async () => {
      const res = await setClientDemoAction(clientId, next);
      if (!res.ok) {
        setChecked(!next);
        toast.error(res.error);
        return;
      }
      toast.success(next ? tc.settings.demoOn : tc.settings.demoOff);
    });
  };

  return (
    <Card>
      <CardContent className="flex items-center justify-between gap-4">
        <Label htmlFor="client-demo" className="text-sm font-medium">
          {tc.settings.demoLabel}
        </Label>
        <Switch id="client-demo" checked={checked} onCheckedChange={change} disabled={pending} />
      </CardContent>
    </Card>
  );
}
