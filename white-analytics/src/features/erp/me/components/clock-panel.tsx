"use client";

import * as React from "react";
import { Camera, Check, Clock, ImageUp, Loader2, LogIn, LogOut, MapPin, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { toast } from "@/lib/toast";
import { timeWib } from "@/features/erp/hr/format";
import type { AttendanceRecord, GeoLocation } from "@/features/erp/hr/types";
import { myClockInAction, myClockOutAction } from "../actions";
import { tm } from "../strings";

const c = tm.clock;
const MAX_SIDE = 960;
const JPEG_QUALITY = 0.8;

/** Draws a video frame or image onto a canvas no wider than MAX_SIDE and returns raw JPEG base64. */
function toJpegBase64(source: CanvasImageSource, w: number, h: number, mirror = false): string {
  const scale = Math.min(1, MAX_SIDE / Math.max(w, h));
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(w * scale);
  canvas.height = Math.round(h * scale);
  const ctx = canvas.getContext("2d");
  if (!ctx) return "";
  if (mirror) {
    ctx.translate(canvas.width, 0);
    ctx.scale(-1, 1);
  }
  ctx.drawImage(source, 0, 0, canvas.width, canvas.height);
  return canvas.toDataURL("image/jpeg", JPEG_QUALITY).split(",")[1] ?? "";
}

function fileToJpegBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      resolve(toJpegBase64(img, img.naturalWidth, img.naturalHeight));
      URL.revokeObjectURL(url);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error(c.cameraError));
    };
    img.src = url;
  });
}

function getLocation(): Promise<GeoLocation | null> {
  return new Promise((resolve) => {
    if (!navigator.geolocation) return resolve(null);
    navigator.geolocation.getCurrentPosition(
      (pos) =>
        resolve({
          latitude: pos.coords.latitude,
          longitude: pos.coords.longitude,
          accuracy: pos.coords.accuracy,
          address: "",
        }),
      () => resolve(null),
      { enableHighAccuracy: true, timeout: 10000, maximumAge: 0 },
    );
  });
}

/**
 * Clock in/out for today (NüHabit ess-clock-panel). The selfie is optional
 * here: camera, a picked photo, or none; GPS is attached when allowed.
 */
export function ClockPanel({ today, hasShift }: { today?: AttendanceRecord; hasShift: boolean }) {
  const [mode, setMode] = React.useState<"in" | "out" | null>(null);
  const clockedIn = Boolean(today?.clockIn);
  const clockedOut = Boolean(today?.clockOut);

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-2 gap-2">
        <Button size="lg" className="h-14 text-base" disabled={clockedIn} onClick={() => setMode("in")}>
          <LogIn className="size-5" /> {c.in}
        </Button>
        <Button
          size="lg"
          variant="outline"
          className="h-14 text-base"
          disabled={!clockedIn || clockedOut}
          onClick={() => setMode("out")}
        >
          <LogOut className="size-5" /> {c.out}
        </Button>
      </div>
      {today && clockedIn ? (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <Clock className="size-4 text-muted-foreground" />
          <span className="tabular">{c.inAt(timeWib(today.clockIn))}</span>
          {clockedOut ? <span className="tabular">· {c.outAt(timeWib(today.clockOut))}</span> : null}
          {today.isLate ? (
            <Badge variant="destructive">{c.late(today.lateMinutes)}</Badge>
          ) : hasShift ? (
            <Badge variant="secondary">{c.onTime}</Badge>
          ) : null}
          {clockedOut ? <Badge variant="outline">{c.done}</Badge> : null}
        </div>
      ) : (
        <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
          <MapPin className="mt-0.5 size-3.5 shrink-0" /> {c.selfieOptional}
        </p>
      )}
      <Dialog open={mode !== null} onOpenChange={(o) => !o && setMode(null)}>
        <DialogContent className="max-h-[92dvh] overflow-y-auto sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{mode === "out" ? c.selfieTitleOut : c.selfieTitleIn}</DialogTitle>
            <DialogDescription>{c.selfieOptional}</DialogDescription>
          </DialogHeader>
          {mode ? <ClockSession mode={mode} attendanceId={today?.id} close={() => setMode(null)} /> : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ClockSession({ mode, attendanceId, close }: { mode: "in" | "out"; attendanceId?: string; close: () => void }) {
  const videoRef = React.useRef<HTMLVideoElement>(null);
  const streamRef = React.useRef<MediaStream | null>(null);
  const fileRef = React.useRef<HTMLInputElement>(null);
  const [camera, setCamera] = React.useState(false);
  const [photo, setPhoto] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [notes, setNotes] = React.useState("");
  const [stage, setStage] = React.useState<"idle" | "locating" | "sending">("idle");

  const stop = React.useCallback(() => {
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    setCamera(false);
  }, []);
  React.useEffect(() => stop, [stop]);

  const startCamera = async () => {
    setError(null);
    setPhoto(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "user" }, audio: false });
      streamRef.current = stream;
      setCamera(true);
      requestAnimationFrame(() => {
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          void videoRef.current.play().catch(() => undefined);
        }
      });
    } catch {
      setError(c.cameraError);
    }
  };

  const capture = () => {
    const v = videoRef.current;
    if (!v || !v.videoWidth) return;
    setPhoto(toJpegBase64(v, v.videoWidth, v.videoHeight, true));
    stop();
  };

  const submit = async (withPhoto: string | null) => {
    setStage("locating");
    const location = await getLocation();
    setStage("sending");
    const input = { photo: withPhoto ?? undefined, location, notes };
    const res = mode === "in" ? await myClockInAction(input) : await myClockOutAction(attendanceId ?? "", input);
    setStage("idle");
    if (!res.ok) {
      toast.error(res.error);
      return;
    }
    const late = res.data.isLate ? ` · ${c.late(res.data.lateMinutes)}` : "";
    toast.success(mode === "in" ? c.successIn : c.successOut, {
      description: `${timeWib(mode === "in" ? res.data.clockIn : res.data.clockOut)}${late}${location ? ` · ${c.locationSaved}` : ""}`,
    });
    close();
  };

  const busy = stage !== "idle";
  return (
    <div className="flex flex-col gap-3">
      {error ? <p className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p> : null}
      {photo ? (
        // eslint-disable-next-line @next/next/no-img-element -- local data URL preview
        <img src={`data:image/jpeg;base64,${photo}`} alt="" className="w-full rounded-xl" />
      ) : camera ? (
        <video ref={videoRef} playsInline muted className="w-full -scale-x-100 rounded-xl bg-black" />
      ) : (
        <div className="grid grid-cols-2 gap-2">
          <Button type="button" variant="outline" className="h-20 flex-col" onClick={startCamera} disabled={busy}>
            <Camera className="size-5" /> {c.useCamera}
          </Button>
          <Button type="button" variant="outline" className="h-20 flex-col" onClick={() => fileRef.current?.click()} disabled={busy}>
            <ImageUp className="size-5" /> {c.useFile}
          </Button>
        </div>
      )}
      <input
        ref={fileRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        capture="user"
        hidden
        onChange={async (e) => {
          const file = e.target.files?.[0];
          e.target.value = "";
          if (!file) return;
          try {
            setPhoto(await fileToJpegBase64(file));
            setError(null);
          } catch (err) {
            setError(err instanceof Error ? err.message : c.cameraError);
          }
        }}
      />
      <Input placeholder={c.notes} value={notes} onChange={(e) => setNotes(e.target.value)} disabled={busy} />
      <DialogFooter className="gap-2">
        {camera ? (
          <Button onClick={capture}>
            <Camera className="size-4" /> {c.capture}
          </Button>
        ) : photo ? (
          <>
            <Button variant="outline" onClick={() => setPhoto(null)} disabled={busy}>
              <RefreshCw className="size-4" /> {c.retake}
            </Button>
            <Button onClick={() => void submit(photo)} disabled={busy}>
              {busy ? <Loader2 className="size-4 animate-spin" /> : <Check className="size-4" />}
              {stage === "locating" ? c.locating : stage === "sending" ? c.sending : c.usePhoto}
            </Button>
          </>
        ) : (
          <Button variant="secondary" onClick={() => void submit(null)} disabled={busy}>
            {busy ? <Loader2 className="size-4 animate-spin" /> : null}
            {stage === "locating" ? c.locating : stage === "sending" ? c.sending : c.skipPhoto}
          </Button>
        )}
      </DialogFooter>
    </div>
  );
}
