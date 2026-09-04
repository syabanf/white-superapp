/**
 * UI sound feedback, synthesised with the Web Audio API.
 *
 * No audio files and no dependency: each cue is a short shaped tone, so it costs
 * nothing to load and can never be blocked by an asset request. Sounds only ever
 * fire in response to something the user did, which also keeps us on the right
 * side of browser autoplay rules (the context is created on that first gesture).
 */

export type SoundKind = "success" | "error" | "notify";

const STORAGE_KEY = "white-analytics-sound";

/** Two-note motifs: up = good, down = bad, single blip = neutral. */
const CUES: Record<SoundKind, { freq: number; at: number; dur: number; type: OscillatorType }[]> = {
  success: [
    { freq: 659.25, at: 0, dur: 0.09, type: "sine" }, // E5
    { freq: 987.77, at: 0.075, dur: 0.13, type: "sine" }, // B5
  ],
  error: [
    { freq: 415.3, at: 0, dur: 0.11, type: "triangle" }, // G#4
    { freq: 311.13, at: 0.095, dur: 0.17, type: "triangle" }, // D#4
  ],
  notify: [{ freq: 880, at: 0, dur: 0.08, type: "sine" }],
};

const PEAK_GAIN = 0.05; // deliberately quiet — this is feedback, not an alert

let ctx: AudioContext | null = null;

function audioContext(): AudioContext | null {
  if (typeof window === "undefined") return null;
  const Ctor = window.AudioContext ?? (window as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctor) return null;
  if (!ctx) {
    try {
      ctx = new Ctor();
    } catch {
      return null;
    }
  }
  return ctx;
}

export function isSoundEnabled(): boolean {
  if (typeof window === "undefined") return false;
  try {
    // Opt-out, not opt-in: the cues are quiet and tied to explicit actions.
    return window.localStorage.getItem(STORAGE_KEY) !== "off";
  } catch {
    return false;
  }
}

export function setSoundEnabled(on: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, on ? "on" : "off");
  } catch {
    /* private mode — sound just stays at its default for this session */
  }
  for (const l of listeners) l();
}

// ── Subscription, so React can read the setting with useSyncExternalStore
// (no effect + setState dance, and other tabs stay in sync).
const listeners = new Set<() => void>();

export function subscribeSound(onChange: () => void): () => void {
  listeners.add(onChange);
  const onStorage = (e: StorageEvent) => {
    if (e.key === STORAGE_KEY) onChange();
  };
  if (typeof window !== "undefined") window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(onChange);
    if (typeof window !== "undefined") window.removeEventListener("storage", onStorage);
  };
}

/** Server snapshot for useSyncExternalStore — no audio during SSR. */
export function soundDisabledSnapshot(): boolean {
  return false;
}

export function playSound(kind: SoundKind): void {
  if (!isSoundEnabled()) return;
  const audio = audioContext();
  if (!audio) return;
  // Browsers park the context until a gesture; our cues always follow one.
  if (audio.state === "suspended") void audio.resume().catch(() => {});

  const now = audio.currentTime;
  for (const note of CUES[kind]) {
    const osc = audio.createOscillator();
    const gain = audio.createGain();
    osc.type = note.type;
    osc.frequency.value = note.freq;

    const start = now + note.at;
    const end = start + note.dur;
    // Fast attack, exponential tail — a click-free, soft blip.
    gain.gain.setValueAtTime(0.0001, start);
    gain.gain.exponentialRampToValueAtTime(PEAK_GAIN, start + 0.012);
    gain.gain.exponentialRampToValueAtTime(0.0001, end);

    osc.connect(gain).connect(audio.destination);
    osc.start(start);
    osc.stop(end + 0.02);
  }
}
