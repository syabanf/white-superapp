"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { deleteClientAction } from "@/features/clients/actions";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

/** Settings → Zona berbahaya: delete client with type-the-slug confirmation (ADMIN only). */
export function DangerZone({ clientId, slug, name }: { clientId: string; slug: string; name: string }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [confirmText, setConfirmText] = React.useState("");
  const [pending, startTransition] = React.useTransition();
  const matches = confirmText.trim() === slug;

  const confirmDelete = () => {
    if (!matches) return;
    startTransition(async () => {
      const res = await deleteClientAction(clientId);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(t.clients.deleted);
      router.push("/clients");
    });
  };

  return (
    <Card className="border-destructive/30 ring-destructive/20">
      <CardContent className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-medium">{t.settings.deleteClient}</p>
          <p className="text-xs text-muted-foreground">{t.settings.dangerDesc}</p>
        </div>
        <AlertDialog
          open={open}
          onOpenChange={(o) => {
            setOpen(o);
            if (!o) setConfirmText("");
          }}
        >
          <AlertDialogTrigger asChild>
            <Button variant="destructive" size="sm">
              <Trash2 className="size-4" /> {t.settings.deleteClient}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {tc.clients.deleteTitle}: {name}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {tc.settings.dangerConfirmDesc} {tc.clients.typeToConfirm(slug)}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <div className="space-y-2">
              <Label htmlFor="confirm-slug" className="sr-only">
                {t.clients.slug}
              </Label>
              <Input
                id="confirm-slug"
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                placeholder={tc.clients.confirmPlaceholder}
                autoComplete="off"
                className="font-mono text-sm"
              />
              {confirmText.length > 0 && !matches ? <p className="text-xs text-destructive">{tc.clients.confirmMismatch}</p> : null}
            </div>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={pending}>{t.common.cancel}</AlertDialogCancel>
              <Button variant="destructive" disabled={!matches || pending} onClick={confirmDelete}>
                {pending ? <Loader2 className="size-4 animate-spin" /> : null}
                {t.common.delete}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </CardContent>
    </Card>
  );
}
