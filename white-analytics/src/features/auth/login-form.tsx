"use client";

import { useActionState, useRef } from "react";
import { Loader2, LockKeyhole, Mail, UserRound } from "lucide-react";
import { loginAction, type LoginState } from "@/features/auth/actions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { t } from "@/i18n/id";

export type DemoAccount = { label: string; email: string; password: string };

export function LoginForm({ callbackUrl, demoAccounts }: { callbackUrl?: string; demoAccounts?: DemoAccount[] }) {
  const [state, action, pending] = useActionState<LoginState, FormData>(loginAction, undefined);
  const emailRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  const fill = (account: DemoAccount) => {
    if (emailRef.current) emailRef.current.value = account.email;
    if (passwordRef.current) passwordRef.current.value = account.password;
    passwordRef.current?.focus();
  };

  return (
    <form action={action} className="space-y-4">
      {callbackUrl ? <input type="hidden" name="callbackUrl" value={callbackUrl} /> : null}
      <div className="space-y-2">
        <Label htmlFor="email">{t.auth.email}</Label>
        <div className="relative">
          <Mail className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={emailRef}
            id="email"
            name="email"
            type="email"
            autoComplete="email"
            required
            placeholder="nama@agency.id"
            className="h-10 pl-9"
            defaultValue=""
          />
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="password">{t.auth.password}</Label>
        <div className="relative">
          <LockKeyhole className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={passwordRef}
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            required
            placeholder="••••••••"
            className="h-10 pl-9"
          />
        </div>
      </div>
      {state?.error ? (
        <p role="alert" className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {state.error}
        </p>
      ) : null}
      <Button type="submit" className="h-10 w-full" disabled={pending}>
        {pending ? (
          <>
            <Loader2 className="size-4 animate-spin" /> {t.auth.signingIn}
          </>
        ) : (
          t.auth.signIn
        )}
      </Button>
      <p className="text-center text-xs text-muted-foreground">{t.auth.forgot}</p>
      {demoAccounts?.length ? (
        <div className="border-t pt-5">
          <p className="label-mono text-muted-foreground">{t.auth.demoHint}</p>
          <div className="mt-2.5 grid grid-cols-2 gap-2">
            {demoAccounts.map((account) => (
              <Button
                key={account.email}
                type="button"
                variant="outline"
                onClick={() => fill(account)}
                disabled={pending}
                title={account.email}
              >
                <UserRound /> {account.label}
              </Button>
            ))}
          </div>
        </div>
      ) : null}
    </form>
  );
}
