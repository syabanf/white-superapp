"use client";

import * as React from "react";
import { CheckCircle2, Download, Eraser, Loader2, PenLine } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { formatDateTime } from "@/lib/format";
import { ps } from "../strings";
import type { PublicContract } from "../types";
import { publicClientFetch } from "./client";

const c = ps.contract;

/** Canvas signature pad; exposes the drawing as a PNG data URL. */
function SignaturePad({ onChange }: { onChange: (dataUrl: string | null) => void }) {
  const ref = React.useRef<HTMLCanvasElement>(null);
  const drawing = React.useRef(false);
  const dirty = React.useRef(false);

  React.useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const ratio = window.devicePixelRatio || 1;
    const { width, height } = canvas.getBoundingClientRect();
    canvas.width = width * ratio;
    canvas.height = height * ratio;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.scale(ratio, ratio);
    ctx.lineWidth = 2.2;
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.strokeStyle = "#111";
  }, []);

  const point = (e: React.PointerEvent<HTMLCanvasElement>) => {
    const r = e.currentTarget.getBoundingClientRect();
    return { x: e.clientX - r.left, y: e.clientY - r.top };
  };

  const clear = () => {
    const canvas = ref.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) return;
    ctx.save();
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.restore();
    dirty.current = false;
    onChange(null);
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="relative">
        <canvas
          ref={ref}
          aria-label={c.draw}
          role="img"
          className="h-44 w-full touch-none rounded-lg border border-dashed bg-white"
          onPointerDown={(e) => {
            e.currentTarget.setPointerCapture(e.pointerId);
            drawing.current = true;
            const ctx = e.currentTarget.getContext("2d");
            const p = point(e);
            ctx?.beginPath();
            ctx?.moveTo(p.x, p.y);
          }}
          onPointerMove={(e) => {
            if (!drawing.current) return;
            const ctx = e.currentTarget.getContext("2d");
            const p = point(e);
            ctx?.lineTo(p.x, p.y);
            ctx?.stroke();
            dirty.current = true;
          }}
          onPointerUp={(e) => {
            drawing.current = false;
            if (dirty.current) onChange(e.currentTarget.toDataURL("image/png"));
          }}
          onPointerLeave={(e) => {
            if (drawing.current && dirty.current) onChange(e.currentTarget.toDataURL("image/png"));
            drawing.current = false;
          }}
        />
        <PenLine className="pointer-events-none absolute bottom-3 left-3 size-4 text-black/20" />
      </div>
      <Button type="button" variant="ghost" size="xs" className="self-end" onClick={clear}>
        <Eraser className="size-3.5" /> {c.clear}
      </Button>
    </div>
  );
}

/** Signing card under the contract body; shows the signed state and PDF link afterwards. */
export function SignContract({ token, contract: initial }: { token: string; contract: PublicContract }) {
  const [contract, setContract] = React.useState(initial);
  const [name, setName] = React.useState("");
  const [signature, setSignature] = React.useState<string | null>(null);
  const [agree, setAgree] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [pending, setPending] = React.useState(false);
  const pdf = `/api/public-erp/api/public/sales/contracts/${encodeURIComponent(token)}/pdf`;

  if (contract.status === "cancelled")
    return (
      <Card className="text-sm text-muted-foreground p-(--card-pad)" role="status">
        {c.cancelled}
      </Card>
    );

  if (contract.signedAt)
    return (
      <Card className="flex flex-col items-start gap-3 p-(--card-pad)" role="status">
        <p className="flex items-center gap-2 font-semibold">
          <CheckCircle2 className="size-5 text-positive" /> {c.signed}
        </p>
        <p className="text-sm text-muted-foreground">{c.signedBy(contract.signerName, formatDateTime(contract.signedAt))}</p>
        <p className="text-sm text-muted-foreground">{c.thanks}</p>
        <Button asChild>
          <a href={pdf} download>
            <Download className="size-4" /> {c.download}
          </a>
        </Button>
      </Card>
    );

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!name.trim()) return setError(`${c.name}: ${ps.required}`);
    if (!signature) return setError(c.drawEmpty);
    if (!agree) return setError(c.agreeRequired);
    setPending(true);
    try {
      const res = await publicClientFetch<PublicContract>(`/v1/public/sales/contracts/${encodeURIComponent(token)}:sign`, {
        method: "POST",
        body: JSON.stringify({ signerName: name.trim(), signaturePng: signature, agree }),
      });
      setContract(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : ps.offline);
    } finally {
      setPending(false);
    }
  };

  return (
    <Card className="p-(--card-pad)">
      <form className="flex flex-col gap-4" onSubmit={submit}>
        <div>
          <h2 className="text-base font-semibold">{c.signTitle}</h2>
          <p className="text-sm text-muted-foreground">{c.signHint}</p>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="signer-name">{c.name}</Label>
          <Input id="signer-name" value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>{c.draw}</Label>
          <SignaturePad onChange={setSignature} />
        </div>
        <label className="flex items-start gap-2 text-sm">
          <Checkbox checked={agree} onCheckedChange={(v) => setAgree(v === true)} className="mt-0.5" />
          {c.agree}
        </label>
        {error ? (
          <p className="text-sm text-destructive" role="alert">
            {error}
          </p>
        ) : null}
        <Button type="submit" size="lg" disabled={pending}>
          {pending ? <Loader2 className="size-4 animate-spin" /> : <PenLine className="size-4" />}
          {c.sign}
        </Button>
      </form>
    </Card>
  );
}
