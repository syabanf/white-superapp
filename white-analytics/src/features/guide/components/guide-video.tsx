import { PlayCircle } from "lucide-react";
import videos from "../videos.json";
import { tg } from "../strings";

const durations: Record<string, number> = videos;

const clock = (s: number) => `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;

/**
 * Screen recording of one guide section with Indonesian subtitles, made by
 * scripts/record-tutorials.mjs.
 * Renders nothing for sections without a recording.
 */
export function GuideVideo({ id, title }: { id: string; title: string }) {
  const seconds = durations[id];
  if (!seconds) return null;
  return (
    <figure className="flex flex-col gap-2">
      <video
        controls
        playsInline
        preload="none"
        poster={`/tutorials/${id}.jpg`}
        aria-label={`${tg.video}: ${title}`}
        className="aspect-video w-full rounded-2xl border bg-muted"
      >
        <source src={`/tutorials/${id}.webm`} type="video/webm" />
        <track kind="subtitles" src={`/tutorials/${id}.vtt`} srcLang="id" label="Indonesia" default />
      </video>
      <figcaption className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <PlayCircle className="size-3.5 text-brand" />
        {tg.video} · {clock(seconds)}
      </figcaption>
    </figure>
  );
}
