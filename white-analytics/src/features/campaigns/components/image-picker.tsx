"use client";

import * as React from "react";
import { ImagePlus, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import type { BuilderImage } from "@/features/campaigns/queries";
import { c } from "@/features/campaigns/strings";

/* eslint-disable @next/next/no-img-element */

/** Pick one image from the client's media library (publishing → Media). Upload stays in that module. */
export function ImagePicker({ images, onPick, label, disabled }: { images: BuilderImage[]; onPick: (url: string) => void; label: string; disabled?: boolean }) {
  const [open, setOpen] = React.useState(false);
  const [q, setQ] = React.useState("");
  const shown = q.trim() ? images.filter((i) => i.filename.toLowerCase().includes(q.trim().toLowerCase())) : images;
  return (
    <>
      <Button type="button" variant="outline" size="xs" onClick={() => setOpen(true)} disabled={disabled}>
        <ImagePlus /> {label}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{c.imageDialogTitle}</DialogTitle>
            <DialogDescription className="sr-only">{c.imageDialogTitle}</DialogDescription>
          </DialogHeader>
          {images.length ? (
            <>
              <div className="relative">
                <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder={c.imageSearch} className="pl-9" aria-label={c.imageSearch} />
              </div>
              <ul className="grid max-h-[55dvh] grid-cols-3 gap-2 overflow-y-auto sm:grid-cols-4">
                {shown.map((img) => (
                  <li key={img.id}>
                    <button
                      type="button"
                      onClick={() => {
                        onPick(img.url);
                        setOpen(false);
                      }}
                      className="lift block aspect-square w-full overflow-hidden rounded-xl bg-muted outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      title={img.filename}
                    >
                      <img src={img.thumbnailUrl ?? img.url} alt={img.altText ?? img.filename} loading="lazy" className="size-full object-cover" />
                    </button>
                  </li>
                ))}
              </ul>
            </>
          ) : (
            <p className="py-8 text-center text-sm text-muted-foreground">{c.imageDialogEmpty}</p>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
