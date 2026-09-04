"use client";

import * as React from "react";
import { useTransition } from "react";
import { UserPlus } from "lucide-react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { addCompetitor } from "@/features/social/actions";
import { normalizeIgUsername } from "@/features/social/lib";
import { s } from "@/features/social/strings";
import { t } from "@/i18n/id";

/** Dialog "Tambah kompetitor" — validasi username (@ dibuang, lowercase) + server action. */
export function AddCompetitorDialog({ clientId }: { clientId: string }) {
  const [open, setOpen] = React.useState(false);
  const [value, setValue] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [pending, startTransition] = useTransition();

  const submit = () => {
    const username = normalizeIgUsername(value);
    if (!username) {
      setError(s.usernameInvalid);
      return;
    }
    startTransition(async () => {
      const res = await addCompetitor({ clientId, username });
      if (res.ok) {
        toast.success(s.competitorAdded, { description: `@${res.data.username}` });
        setOpen(false);
        setValue("");
        setError(null);
      } else {
        setError(res.error);
      }
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
        <Button size="sm">
          <UserPlus className="size-4" /> {t.social.addCompetitor}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t.social.addCompetitor}</DialogTitle>
          <DialogDescription>{s.addCompetitorDesc}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            submit();
          }}
          className="space-y-4"
        >
          <div className="space-y-1.5">
            <Label htmlFor="competitor-username">{t.social.competitorUsername}</Label>
            <Input
              id="competitor-username"
              value={value}
              onChange={(e) => {
                setValue(e.target.value);
                if (error) setError(null);
              }}
              placeholder="@kopikenangan"
              autoComplete="off"
              autoFocus
            />
            <p className="text-xs text-muted-foreground">{t.social.competitorHint}</p>
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
