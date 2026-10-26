"use client";

import * as React from "react";
import { Check, ChevronsUpDown, Loader2 } from "lucide-react";
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
import { Switch } from "@/components/ui/switch";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { cn } from "@/lib/utils";
import { th } from "../strings";

export type Option = { value: string; label: string };
export type EmployeeOption = { id: string; label: string };
export type FieldErrors = Record<string, string[]> | undefined;

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
  placeholder,
  emptyLabel,
  id,
  disabled,
  className,
  invalid,
}: {
  value: string;
  onChange: (v: string) => void;
  options: Option[];
  placeholder?: string;
  /** when set, an extra first item that clears the value */
  emptyLabel?: string;
  id?: string;
  disabled?: boolean;
  className?: string;
  invalid?: boolean;
}) {
  return (
    <Select
      value={value === "" ? (emptyLabel ? NONE : "") : value}
      onValueChange={(v) => onChange(v === NONE ? "" : v)}
      disabled={disabled}
    >
      <SelectTrigger id={id} className={cn("w-full", className)} aria-invalid={invalid || undefined}>
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

/** Searchable employee picker. */
export function EmployeePicker({
  value,
  onChange,
  options,
  id,
  placeholder = th.pickEmployee,
  allowEmpty,
  invalid,
}: {
  value: string;
  onChange: (id: string) => void;
  options: EmployeeOption[];
  id?: string;
  placeholder?: string;
  allowEmpty?: string;
  invalid?: boolean;
}) {
  const [open, setOpen] = React.useState(false);
  const selected = options.find((o) => o.id === value);
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
          className={cn("w-full justify-between font-normal shadow-none", !selected && "text-muted-foreground")}
        >
          <span className="truncate">{selected?.label ?? (value === "" && allowEmpty ? allowEmpty : placeholder)}</span>
          <ChevronsUpDown className="size-3.5 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) min-w-64 p-0" align="start">
        <Command>
          <CommandInput placeholder={th.pickEmployee} />
          <CommandList>
            <CommandEmpty>{th.none}</CommandEmpty>
            {allowEmpty ? (
              <CommandItem
                value={allowEmpty}
                onSelect={() => {
                  onChange("");
                  setOpen(false);
                }}
              >
                <Check className={cn("size-3.5", value === "" ? "opacity-100" : "opacity-0")} />
                {allowEmpty}
              </CommandItem>
            ) : null}
            {options.map((o) => (
              <CommandItem
                key={o.id}
                value={o.label}
                onSelect={() => {
                  onChange(o.id);
                  setOpen(false);
                }}
              >
                <Check className={cn("size-3.5", o.id === value ? "opacity-100" : "opacity-0")} />
                <span className="truncate">{o.label}</span>
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

/**
 * Dialog with a trigger; the body receives `close` and remounts on every
 * open, so forms always start fresh.
 */
export function FormDialog({
  trigger,
  title,
  description,
  wide,
  open: openProp,
  onOpenChange,
  children,
}: {
  trigger?: React.ReactNode;
  title: string;
  description?: string;
  wide?: boolean;
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
      <DialogContent className={cn("max-h-[90dvh] overflow-y-auto", wide ? "sm:max-w-2xl" : "sm:max-w-lg")}>
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
  submitLabel = th.save,
  destructive,
}: {
  onCancel: () => void;
  pending: boolean;
  submitLabel?: string;
  destructive?: boolean;
}) {
  return (
    <DialogFooter className="gap-2 pt-2">
      <Button type="button" variant="outline" onClick={onCancel} disabled={pending}>
        {th.cancel}
      </Button>
      <Button type="submit" disabled={pending} variant={destructive ? "destructive" : "default"}>
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
  description = th.confirmDeleteBody,
  confirmLabel = th.confirm,
  success,
  destructive = true,
  action,
}: {
  trigger: React.ReactNode;
  title: string;
  description?: string;
  confirmLabel?: string;
  success?: string;
  destructive?: boolean;
  action: () => Promise<ActionResult<unknown>>;
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
          <AlertDialogCancel disabled={pending}>{th.cancel}</AlertDialogCancel>
          <AlertDialogAction
            disabled={pending}
            className={destructive ? "bg-destructive text-white hover:bg-destructive/90" : undefined}
            onClick={(e) => {
              e.preventDefault();
              void run(action, { success, onOk: () => setOpen(false) });
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

/** Approve and reject buttons; reject asks for a reason. */
export function DecisionButtons({
  onApprove,
  onReject,
  size = "xs",
}: {
  onApprove: () => Promise<ActionResult<unknown>>;
  onReject: (reason: string) => Promise<ActionResult<unknown>>;
  size?: "xs" | "sm";
}) {
  const { pending, run } = useErpAction();
  return (
    <div className="flex items-center justify-end gap-1.5">
      <ReasonDialog
        title={th.reject}
        label={th.rejectReason}
        submitLabel={th.reject}
        onSubmit={onReject}
        success={th.rejected}
        trigger={
          <Button size={size} variant="outline" disabled={pending}>
            {th.reject}
          </Button>
        }
      />
      <Button size={size} disabled={pending} onClick={() => void run(onApprove, { success: th.approved })}>
        {pending ? <Loader2 className="size-3.5 animate-spin" /> : null}
        {th.approve}
      </Button>
    </div>
  );
}

/** Dialog that collects one required text before running an action. */
export function ReasonDialog({
  trigger,
  title,
  label,
  submitLabel,
  success,
  onSubmit,
}: {
  trigger: React.ReactNode;
  title: string;
  label: string;
  submitLabel: string;
  success?: string;
  onSubmit: (reason: string) => Promise<ActionResult<unknown>>;
}) {
  return (
    <FormDialog trigger={trigger} title={title}>
      {(close) => <ReasonForm label={label} submitLabel={submitLabel} success={success} onSubmit={onSubmit} close={close} />}
    </FormDialog>
  );
}

function ReasonForm({
  label,
  submitLabel,
  success,
  onSubmit,
  close,
}: {
  label: string;
  submitLabel: string;
  success?: string;
  onSubmit: (reason: string) => Promise<ActionResult<unknown>>;
  close: () => void;
}) {
  const [reason, setReason] = React.useState("");
  const [error, setError] = React.useState<string[]>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (!reason.trim()) return setError([th.rejectReasonRequired]);
        void run(() => onSubmit(reason), { success, onOk: close });
      }}
    >
      <Fld label={label} htmlFor="reason" error={error}>
        <Textarea id="reason" value={reason} onChange={(e) => setReason(e.target.value)} rows={3} autoFocus />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={submitLabel} destructive />
    </form>
  );
}

/** Tabs bound to the `tab` URL param, so each tab is linkable and server-rendered. */
export function UrlTabs({ tabs, value }: { tabs: Option[]; value: string }) {
  const { set, isPending } = useUrlFilters(["tab"]);
  return (
    <Tabs value={value} onValueChange={(v) => set({ tab: v === tabs[0]?.value ? null : v })}>
      <div className="-mx-1 overflow-x-auto px-1 scrollbar-thin">
        <TabsList className={cn(isPending && "opacity-70")}>
          {tabs.map((t) => (
            <TabsTrigger key={t.value} value={t.value}>
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </div>
    </Tabs>
  );
}

/** Applies server field errors to local state after a failed action. */
export function useFieldErrors() {
  const [errors, setErrors] = React.useState<FieldErrors>();
  const onFail = React.useCallback((res: { fieldErrors?: Record<string, string[]> }) => setErrors(res.fieldErrors), []);
  return { errors, setErrors, onFail };
}

/** Switch bound to a "1"/absent URL param. */
export function UrlSwitch({ param, label }: { param: string; label: string }) {
  const { values, set } = useUrlFilters([param]);
  const id = `sw-${param}`;
  return (
    <div className="flex items-center gap-2 px-1">
      <Switch id={id} checked={values[param] === "1"} onCheckedChange={(v) => set({ [param]: v ? "1" : null })} />
      <Label htmlFor={id} className="text-xs whitespace-nowrap text-muted-foreground">
        {label}
      </Label>
    </div>
  );
}

/** Single date bound to a URL param ("YYYY-MM-DD"); empty means `fallback`. */
export function UrlDate({ param = "date", fallback, className }: { param?: string; fallback: string; className?: string }) {
  const { values, set } = useUrlFilters([param]);
  return (
    <DatePicker
      value={values[param] ?? fallback}
      onChange={(v) => set({ [param]: v && v !== fallback ? v : null })}
      className={cn("h-8 w-auto", className)}
    />
  );
}
