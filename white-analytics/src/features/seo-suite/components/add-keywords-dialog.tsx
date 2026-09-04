"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Plus } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { t } from "@/i18n/id";
import { addTrackedKeywords } from "@/features/seo-suite/actions";
import { parseKeywordLines } from "@/features/seo-suite/lib";
import { s } from "@/features/seo-suite/strings";

export function AddKeywordsDialog({ clientId, propertyId, variant = "default" }: { clientId: string; propertyId: string; variant?: "default" | "outline" }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [text, setText] = React.useState("");
  const [device, setDevice] = React.useState<"MOBILE" | "DESKTOP">("MOBILE");
  const [tags, setTags] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [pending, startTransition] = React.useTransition();
  const count = parseKeywordLines(text).length;

  const submit = () => {
    if (count === 0) {
      setError(s.keywordsInvalid);
      return;
    }
    startTransition(async () => {
      const res = await addTrackedKeywords({ clientId, propertyId, text, device, tags });
      if (res.ok) {
        toast.success(`${res.data.added} ${s.keywordsAdded}`);
        setOpen(false);
        setText("");
        setTags("");
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
        <Button size="sm" variant={variant}>
          <Plus className="size-4" /> {s.addKeywords}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{s.addKeywords}</DialogTitle>
          <DialogDescription>{s.addKeywordsDesc}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            submit();
          }}
          className="space-y-4"
        >
          <div className="space-y-1.5">
            <Label htmlFor="kw-text">{s.keywordsTextarea}</Label>
            <Textarea
              id="kw-text"
              value={text}
              onChange={(e) => {
                setText(e.target.value);
                if (error) setError(null);
              }}
              placeholder={s.keywordsPlaceholder}
              rows={7}
              autoFocus
            />
            <p className="text-xs text-muted-foreground tabular">{count} / 100</p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label>{s.device}</Label>
              <Select value={device} onValueChange={(v) => setDevice(v as "MOBILE" | "DESKTOP")}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="MOBILE">{t.seo.mobile}</SelectItem>
                  <SelectItem value="DESKTOP">{t.seo.desktop}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="kw-tags">{s.tagsLabel}</Label>
              <Input id="kw-tags" value={tags} onChange={(e) => setTags(e.target.value)} placeholder={s.tagsPlaceholder} />
            </div>
          </div>
          {error ? (
            <p className="text-xs text-destructive" role="alert">
              {error}
            </p>
          ) : null}
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {t.common.cancel}
            </Button>
            <Button type="submit" disabled={pending || count === 0}>
              {pending ? <Spinner className="size-4" /> : null}
              {t.common.add}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
