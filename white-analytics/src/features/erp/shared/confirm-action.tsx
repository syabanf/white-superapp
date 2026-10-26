"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ss } from "./strings";

/**
 * Confirmation dialog around a trigger. `onConfirm` returns true (or nothing)
 * to close, false to keep the dialog open (e.g. the action failed).
 */
export function ConfirmAction({
  trigger,
  title,
  description,
  confirmLabel,
  destructive,
  onConfirm,
  children,
}: {
  trigger: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  confirmLabel: React.ReactNode;
  destructive?: boolean;
  onConfirm: () => Promise<boolean | void> | boolean | void;
  children?: React.ReactNode;
}) {
  const [open, setOpen] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  return (
    <AlertDialog open={open} onOpenChange={(o) => !busy && setOpen(o)}>
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          {description ? <AlertDialogDescription>{description}</AlertDialogDescription> : null}
        </AlertDialogHeader>
        {children}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy}>{ss.cancel}</AlertDialogCancel>
          <AlertDialogAction
            variant={destructive ? "destructive" : "default"}
            disabled={busy}
            onClick={async (e) => {
              e.preventDefault();
              setBusy(true);
              try {
                const keep = (await onConfirm()) === false;
                if (!keep) setOpen(false);
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? <Loader2 className="size-4 animate-spin" /> : null}
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/** ConfirmAction with a required reason textarea (reject, void). */
export function ReasonAction({
  trigger,
  title,
  description,
  reasonLabel,
  confirmLabel,
  destructive = true,
  required = true,
  onConfirm,
}: {
  trigger: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  reasonLabel: string;
  confirmLabel: React.ReactNode;
  destructive?: boolean;
  required?: boolean;
  onConfirm: (reason: string) => Promise<boolean | void>;
}) {
  const [reason, setReason] = React.useState("");
  const fieldId = React.useId();
  return (
    <ConfirmAction
      trigger={trigger}
      title={title}
      description={description}
      confirmLabel={confirmLabel}
      destructive={destructive}
      onConfirm={async () => {
        if (required && !reason.trim()) return false;
        const res = await onConfirm(reason.trim());
        if (res !== false) setReason("");
        return res;
      }}
    >
      <div className="grid gap-1.5">
        <Label htmlFor={fieldId}>{reasonLabel}</Label>
        <Textarea id={fieldId} value={reason} onChange={(e) => setReason(e.target.value)} rows={3} />
      </div>
    </ConfirmAction>
  );
}
