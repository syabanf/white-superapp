/* eslint-disable @next/next/no-img-element */
import { FileText, FileWarning } from "lucide-react";
import { mediaKind } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import { cn } from "@/lib/utils";

/** Inline preview of a file served through the gateway proxy. */
export function FilePreview({ src, mime, name, className, controls = true }: { src: string; mime?: string; name: string; className?: string; controls?: boolean }) {
  const kind = mediaKind(mime);
  if (kind === "image") return <img src={src} alt={name} loading="lazy" className={cn("size-full bg-muted object-contain", className)} />;
  if (kind === "video") return <video src={src} controls={controls} muted={!controls} preload="metadata" className={cn("size-full bg-black object-contain", className)} />;
  const Icon = kind === "pdf" ? FileText : FileWarning;
  return (
    <div className={cn("flex size-full flex-col items-center justify-center gap-1.5 bg-muted p-3 text-center text-muted-foreground", className)}>
      <Icon className="size-6" />
      <span className="line-clamp-2 text-xs break-all">{name}</span>
      {kind === "other" ? <span className="text-[10px]">{c.noPreview}</span> : null}
    </div>
  );
}
