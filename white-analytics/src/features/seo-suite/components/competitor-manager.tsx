"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Plus, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { formatCompact, formatNumber } from "@/lib/format";
import { MAX_COMPETITOR_DOMAINS, normalizeDomain } from "@/lib/metrics/seo-suite";
import type { CompetitorSuggestion } from "@/lib/providers/dataforseo/types";
import { t } from "@/i18n/id";
import { addCompetitorDomain, removeCompetitorDomain } from "@/features/seo-suite/actions-domains";
import { s } from "@/features/seo-suite/strings";

export function AddDomainDialog({ clientId, propertyId, disabled }: { clientId: string; propertyId: string; disabled?: boolean }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [value, setValue] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [pending, startTransition] = React.useTransition();

  const submit = () => {
    const domain = normalizeDomain(value);
    if (!domain) {
      setError(s.domainInvalid);
      return;
    }
    startTransition(async () => {
      const res = await addCompetitorDomain({ clientId, propertyId, domain });
      if (res.ok) {
        toast.success(s.domainAdded, { description: res.data.domain });
        setOpen(false);
        setValue("");
        setError(null);
        router.refresh();
      } else setError(res.error);
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setError(null);
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm" disabled={disabled}>
          <Plus className="size-4" /> {s.addDomain}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{s.addDomain}</DialogTitle>
          <DialogDescription>{s.addDomainDesc}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            submit();
          }}
          className="space-y-4"
        >
          <div className="space-y-1.5">
            <Label htmlFor="comp-domain">{s.domainLabel}</Label>
            <Input
              id="comp-domain"
              value={value}
              onChange={(e) => {
                setValue(e.target.value);
                if (error) setError(null);
              }}
              placeholder={s.domainPlaceholder}
              autoComplete="off"
              autoFocus
            />
            {error ? (
              <p className="text-xs text-destructive" role="alert">
                {error}
              </p>
            ) : null}
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {t.common.cancel}
            </Button>
            <Button type="submit" disabled={pending || value.trim().length === 0}>
              {pending ? <Spinner className="size-4" /> : null}
              {t.common.add}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/** Own domain + competitor chips (remove) and the provider's suggestions (one-click add). */
export function CompetitorManager({
  clientId,
  propertyId,
  ownDomain,
  competitors,
  suggestions,
  canManage,
}: {
  clientId: string;
  propertyId: string;
  ownDomain: string;
  competitors: { id: string; domain: string }[];
  suggestions: CompetitorSuggestion[];
  canManage: boolean;
}) {
  const router = useRouter();
  const [pendingId, setPendingId] = React.useState<string | null>(null);
  const [, startTransition] = React.useTransition();
  const full = competitors.length >= MAX_COMPETITOR_DOMAINS;

  const remove = (c: { id: string; domain: string }) => {
    setPendingId(c.id);
    startTransition(async () => {
      const res = await removeCompetitorDomain({ clientId, id: c.id });
      setPendingId(null);
      if (res.ok) {
        toast.success(s.domainRemoved, { description: c.domain });
        router.refresh();
      } else toast.error(res.error);
    });
  };
  const add = (domain: string) => {
    setPendingId(domain);
    startTransition(async () => {
      const res = await addCompetitorDomain({ clientId, propertyId, domain });
      setPendingId(null);
      if (res.ok) {
        toast.success(s.domainAdded, { description: domain });
        router.refresh();
      } else toast.error(res.error);
    });
  };

  return (
    <div className="grid gap-5 lg:grid-cols-2">
      <Card className="gap-3 py-0">
        <CardHeader className="px-5 pt-4 pb-0">
          <CardTitle className="text-sm font-semibold">{s.competitorsCount}</CardTitle>
          <CardDescription className="text-xs">
            {formatNumber(competitors.length)} / {MAX_COMPETITOR_DOMAINS}
          </CardDescription>
        </CardHeader>
        <CardContent className="px-5 pb-5">
          <ul className="divide-y">
            <li className="flex items-center justify-between gap-2 py-2 text-sm">
              <span className="font-medium">{ownDomain}</span>
              <Badge variant="secondary" className="text-[10px]">
                {s.yourDomain}
              </Badge>
            </li>
            {competitors.map((c) => (
              <li key={c.id} className="flex items-center justify-between gap-2 py-2 text-sm">
                <span>{c.domain}</span>
                {canManage ? (
                  <Button variant="ghost" size="icon-xs" aria-label={s.removeDomain} onClick={() => remove(c)} disabled={pendingId === c.id}>
                    {pendingId === c.id ? <Spinner className="size-3.5" /> : <Trash2 className="size-3.5" />}
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
          {canManage ? (
            <div className="pt-3">
              <AddDomainDialog clientId={clientId} propertyId={propertyId} disabled={full} />
              {full ? <p className="mt-1.5 text-xs text-muted-foreground">{s.domainMax}</p> : null}
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card className="gap-3 py-0">
        <CardHeader className="px-5 pt-4 pb-0">
          <CardTitle className="text-sm font-semibold">{s.suggestedTitle}</CardTitle>
          <CardDescription className="text-xs">{s.suggestedDesc}</CardDescription>
        </CardHeader>
        <CardContent className="px-5 pb-5">
          {suggestions.length === 0 ? (
            <p className="py-4 text-sm text-muted-foreground">{t.common.noData}</p>
          ) : (
            <ul className="divide-y">
              {suggestions.map((sg) => (
                <li key={sg.domain} className="flex items-center justify-between gap-3 py-2 text-sm">
                  <div className="min-w-0">
                    <p className="truncate font-medium">{sg.domain}</p>
                    <p className="text-xs text-muted-foreground">
                      {formatNumber(sg.intersections)} {s.intersections.toLowerCase()} · {formatCompact(sg.organicTraffic)} {s.organicTraffic.toLowerCase()}
                    </p>
                  </div>
                  {canManage ? (
                    <Button variant="outline" size="xs" onClick={() => add(sg.domain)} disabled={full || pendingId === sg.domain}>
                      {pendingId === sg.domain ? <Spinner className="size-3" /> : <Plus className="size-3" />} {s.addSuggested}
                    </Button>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
