"use client";

import * as React from "react";
import { Check, ChevronsUpDown, Copy, Loader2 } from "lucide-react";
import type { ActionResult } from "@/lib/action-result";
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
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { useErpAction } from "@/features/erp/shared/client";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { KIND } from "../lib";
import { lbl, ss } from "../strings";

export type Option = { value: string; label: string };
export type FieldErrors = Record<string, string[]> | undefined;

/** Badge for a Sales value set entry ("diterima", "lunas", …). */
export function SalesBadge({ value, map, className }: { value: string; map: Record<string, string>; className?: string }) {
  if (!value) return null;
  return (
    <StatusBadge kind={KIND[value] ?? "neutral"} className={className}>
      {lbl(map, value)}
    </StatusBadge>
  );
}

/** Label, control and the first field error. */
export function Fld({
  label,
  htmlFor,
  error,
  hint,
  className,
  children,
}: {
  label: string;
  htmlFor?: string;
  error?: string[];
  hint?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1.5", className)}>
      <Label htmlFor={htmlFor}>{label}</Label>
      {children}
      {error?.[0] ? (
        <p className="text-xs text-destructive">{error[0]}</p>
      ) : hint ? (
        <p className="text-xs text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  );
}

const NONE = "__none__";

/** Radix select over plain options; "" maps to the optional empty choice. */
export function SimpleSelect({
  value,
  onChange,
  options,
  placeholder = ss.pickOne,
  emptyLabel,
  id,
  disabled,
  className,
  invalid,
  size,
}: {
  value: string;
  onChange: (v: string) => void;
  options: Option[];
  placeholder?: string;
  emptyLabel?: string;
  id?: string;
  disabled?: boolean;
  className?: string;
  invalid?: boolean;
  size?: "sm" | "default";
}) {
  return (
    <Select
      value={value === "" ? (emptyLabel ? NONE : "") : value}
      onValueChange={(v) => onChange(v === NONE ? "" : v)}
      disabled={disabled}
    >
      <SelectTrigger id={id} size={size} className={cn("w-full", className)} aria-invalid={invalid || undefined}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {emptyLabel ? <SelectItem value={NONE}>{emptyLabel}</SelectItem> : null}
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/** Searchable single picker over options. */
export function Picker({
  value,
  onChange,
  options,
  id,
  placeholder = ss.pickOne,
  allowEmpty,
  invalid,
  disabled,
}: {
  value: string;
  onChange: (v: string) => void;
  options: Option[];
  id?: string;
  placeholder?: string;
  allowEmpty?: string;
  invalid?: boolean;
  disabled?: boolean;
}) {
  const [open, setOpen] = React.useState(false);
  const selected = options.find((o) => o.value === value);
  const pick = (v: string) => {
    onChange(v);
    setOpen(false);
  };
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-invalid={invalid || undefined}
          disabled={disabled}
          className={cn("w-full justify-between font-normal shadow-none", !selected && "text-muted-foreground")}
        >
          <span className="truncate">{selected?.label ?? (value === "" && allowEmpty ? allowEmpty : placeholder)}</span>
          <ChevronsUpDown className="size-3.5 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) min-w-64 p-0" align="start">
        <Command>
          <CommandInput placeholder={ss.search} />
          <CommandList>
            <CommandEmpty>{ss.none}</CommandEmpty>
            {allowEmpty ? (
              <CommandItem value={allowEmpty} onSelect={() => pick("")}>
                <Check className={cn("size-3.5", value === "" ? "opacity-100" : "opacity-0")} />
                {allowEmpty}
              </CommandItem>
            ) : null}
            {options.map((o) => (
              <CommandItem key={o.value} value={`${o.label} ${o.value}`} onSelect={() => pick(o.value)}>
                <Check className={cn("size-3.5", o.value === value ? "opacity-100" : "opacity-0")} />
                <span className="truncate">{o.label}</span>
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

/** Toggle chips for a multi-value field. */
export function ChipToggle({
  value,
  onChange,
  options,
}: {
  value: string[];
  onChange: (v: string[]) => void;
  options: Option[];
}) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {options.map((o) => {
        const on = value.includes(o.value);
        return (
          <button
            key={o.value}
            type="button"
            aria-pressed={on}
            onClick={() => onChange(on ? value.filter((x) => x !== o.value) : [...value, o.value])}
            className={cn(
              "rounded-full border px-2.5 py-1 text-xs transition-colors",
              on ? "border-brand bg-brand/10 text-foreground" : "text-muted-foreground hover:bg-accent",
            )}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

/**
 * Dialog with an optional trigger; the body receives `close` and remounts on
 * every open, so forms always start fresh.
 */
export function FormDialog({
  trigger,
  title,
  description,
  size = "md",
  open: openProp,
  onOpenChange,
  children,
}: {
  trigger?: React.ReactNode;
  title: string;
  description?: string;
  size?: "md" | "lg" | "xl";
  open?: boolean;
  onOpenChange?: (o: boolean) => void;
  children: (close: () => void) => React.ReactNode;
}) {
  const [state, setState] = React.useState(false);
  const open = openProp ?? state;
  const setOpen = (o: boolean) => {
    onOpenChange?.(o);
    if (openProp === undefined) setState(o);
  };
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {trigger ? <DialogTrigger asChild>{trigger}</DialogTrigger> : null}
      <DialogContent
        className={cn(
          "max-h-[90dvh] overflow-y-auto",
          size === "md" ? "sm:max-w-lg" : size === "lg" ? "sm:max-w-2xl" : "sm:max-w-[calc(100%-2rem)] lg:max-w-4xl",
        )}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description ? <DialogDescription>{description}</DialogDescription> : null}
        </DialogHeader>
        {open ? children(() => setOpen(false)) : null}
      </DialogContent>
    </Dialog>
  );
}

/** Footer with cancel and a submit button that shows progress. */
export function FormFooter({
  onCancel,
  pending,
  submitLabel = ss.save,
  destructive,
  disabled,
}: {
  onCancel: () => void;
  pending: boolean;
  submitLabel?: string;
  destructive?: boolean;
  disabled?: boolean;
}) {
  return (
    <DialogFooter className="gap-2 pt-2">
      <Button type="button" variant="outline" onClick={onCancel} disabled={pending}>
        {ss.cancel}
      </Button>
      <Button type="submit" disabled={pending || disabled} variant={destructive ? "destructive" : "default"}>
        {pending ? <Loader2 className="size-4 animate-spin" /> : null}
        {submitLabel}
      </Button>
    </DialogFooter>
  );
}

/** Button that asks for confirmation, then runs the action with a toast. */
export function ConfirmAction({
  trigger,
  title,
  description = ss.confirmDeleteBody,
  confirmLabel = ss.confirm,
  success = ss.deleted,
  destructive = true,
  action,
  onDone,
}: {
  trigger: React.ReactNode;
  title: string;
  description?: string;
  confirmLabel?: string;
  success?: string;
  destructive?: boolean;
  action: () => Promise<ActionResult<unknown>>;
  onDone?: () => void;
}) {
  const [open, setOpen] = React.useState(false);
  const { pending, run } = useErpAction();
  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>{ss.cancel}</AlertDialogCancel>
          <AlertDialogAction
            disabled={pending}
            className={destructive ? "bg-destructive text-white hover:bg-destructive/90" : undefined}
            onClick={(e) => {
              e.preventDefault();
              void run(action, {
                success,
                onOk: () => {
                  setOpen(false);
                  onDone?.();
                },
              });
            }}
          >
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/** Dialog that collects one text (required unless `optional`) before running an action. */
export function ReasonDialog({
  trigger,
  title,
  label,
  submitLabel,
  success,
  optional,
  destructive = true,
  onSubmit,
}: {
  trigger: React.ReactNode;
  title: string;
  label: string;
  submitLabel: string;
  success?: string;
  optional?: boolean;
  destructive?: boolean;
  onSubmit: (text: string) => Promise<ActionResult<unknown>>;
}) {
  return (
    <FormDialog trigger={trigger} title={title}>
      {(close) => (
        <ReasonForm
          label={label}
          submitLabel={submitLabel}
          success={success}
          optional={optional}
          destructive={destructive}
          onSubmit={onSubmit}
          close={close}
        />
      )}
    </FormDialog>
  );
}

function ReasonForm({
  label,
  submitLabel,
  success,
  optional,
  destructive,
  onSubmit,
  close,
}: {
  label: string;
  submitLabel: string;
  success?: string;
  optional?: boolean;
  destructive: boolean;
  onSubmit: (text: string) => Promise<ActionResult<unknown>>;
  close: () => void;
}) {
  const [text, setText] = React.useState("");
  const [error, setError] = React.useState<string[]>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (!optional && !text.trim()) return setError(["Wajib diisi"]);
        void run(() => onSubmit(text.trim()), { success, onOk: close });
      }}
    >
      <Fld label={label} htmlFor="reason-text" error={error}>
        <Textarea id="reason-text" value={text} onChange={(e) => setText(e.target.value)} rows={3} autoFocus />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={submitLabel} destructive={destructive} />
    </form>
  );
}

/** Button that runs an action directly with a toast. */
export function ActionButton({
  action,
  success,
  children,
  onOk,
  ...props
}: Omit<React.ComponentProps<typeof Button>, "onClick" | "action"> & {
  action: () => Promise<ActionResult<unknown>>;
  success?: string;
  onOk?: (data: unknown) => void;
}) {
  const { pending, run } = useErpAction();
  return (
    <Button {...props} disabled={pending || props.disabled} onClick={() => void run(action, { success, onOk })}>
      {pending ? <Loader2 className="size-3.5 animate-spin" /> : null}
      {children}
    </Button>
  );
}

/** Copies an absolute URL built from a path on this origin. */
export function CopyLinkButton({ path, label = ss.copy, size = "sm" }: { path: string; label?: string; size?: "sm" | "xs" }) {
  return (
    <Button
      type="button"
      variant="outline"
      size={size}
      onClick={() => {
        const url = new URL(path, window.location.origin).toString();
        void navigator.clipboard.writeText(url).then(
          () => toast.success(ss.copied),
          () => toast.error(url),
        );
      }}
    >
      <Copy className="size-3.5" /> {label}
    </Button>
  );
}

/** Small definition row for detail panels. */
export function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-sm break-words">{children || ss.dash}</dd>
    </div>
  );
}
