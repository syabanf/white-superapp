import { cn } from "@/lib/utils";

/**
 * Platform identity marks — brand colors are used ONLY here (identity), never as chart series.
 */
export function PlatformIcon({ platform, className, mono }: { platform: string; className?: string; mono?: boolean }) {
  const p = platform.toUpperCase();
  const color = mono ? "currentColor" : p === "INSTAGRAM" ? "var(--brand-instagram)" : p === "FACEBOOK" ? "var(--brand-facebook)" : p === "TIKTOK" ? "var(--brand-tiktok)" : "currentColor";
  if (p === "INSTAGRAM") {
    return (
      <svg viewBox="0 0 24 24" className={cn("size-4 shrink-0", className)} fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-label="Instagram">
        <rect x="3" y="3" width="18" height="18" rx="5" />
        <circle cx="12" cy="12" r="4" />
        <circle cx="17.5" cy="6.5" r="1" fill={color} stroke="none" />
      </svg>
    );
  }
  if (p === "FACEBOOK") {
    return (
      <svg viewBox="0 0 24 24" className={cn("size-4 shrink-0", className)} fill={color} aria-label="Facebook">
        <path d="M22 12a10 10 0 1 0-11.56 9.88v-6.99H7.9V12h2.54V9.8c0-2.5 1.49-3.89 3.77-3.89 1.09 0 2.24.2 2.24.2v2.46h-1.26c-1.24 0-1.63.77-1.63 1.56V12h2.78l-.44 2.89h-2.34v6.99A10 10 0 0 0 22 12Z" />
      </svg>
    );
  }
  if (p === "TIKTOK") {
    return (
      <svg viewBox="0 0 24 24" className={cn("size-4 shrink-0", className)} fill={color} aria-label="TikTok">
        <path d="M16.5 3c.4 2.4 1.9 3.9 4.3 4.1v3.1c-1.6.1-3-.4-4.3-1.3v6.5c0 3.5-2.9 6.1-6.3 5.9-3.1-.2-5.6-2.8-5.6-6 0-3.5 3-6.3 6.5-6v3.2c-1.7-.4-3.3.9-3.3 2.7 0 1.6 1.3 2.9 2.9 2.9 1.7 0 2.8-1.3 2.8-3.1V3h3Z" />
      </svg>
    );
  }
  return <span className={cn("inline-block size-4 rounded-full bg-muted", className)} />;
}

export function platformLabel(platform: string): string {
  const p = platform.toUpperCase();
  return p === "INSTAGRAM" ? "Instagram" : p === "FACEBOOK" ? "Facebook" : p === "TIKTOK" ? "TikTok" : platform;
}
