import { describe, it, expect, afterEach, vi } from "vitest";

/** Minimal localStorage stand-in; the module reads it through `window`. */
function fakeWindow(store: Record<string, string> = {}) {
  return {
    localStorage: {
      getItem: (k: string) => (k in store ? store[k]! : null),
      setItem: (k: string, v: string) => {
        store[k] = v;
      },
    },
  };
}

async function load() {
  vi.resetModules();
  return import("@/lib/sound");
}

afterEach(() => {
  // @ts-expect-error test shim
  delete globalThis.window;
});

describe("sound preferences", () => {
  it("is off during SSR (no window) and never throws", async () => {
    const { isSoundEnabled, playSound } = await load();
    expect(isSoundEnabled()).toBe(false);
    expect(() => playSound("success")).not.toThrow();
  });

  it("defaults to on in the browser", async () => {
    // @ts-expect-error test shim
    globalThis.window = fakeWindow();
    const { isSoundEnabled } = await load();
    expect(isSoundEnabled()).toBe(true);
  });

  it("remembers an explicit mute", async () => {
    const store: Record<string, string> = {};
    // @ts-expect-error test shim
    globalThis.window = fakeWindow(store);
    const { isSoundEnabled, setSoundEnabled } = await load();
    setSoundEnabled(false);
    expect(store["white-analytics-sound"]).toBe("off");
    expect(isSoundEnabled()).toBe(false);
    setSoundEnabled(true);
    expect(isSoundEnabled()).toBe(true);
  });

  it("stays silent when muted, even if an AudioContext exists", async () => {
    const store: Record<string, string> = { "white-analytics-sound": "off" };
    const AudioCtor = vi.fn();
    // @ts-expect-error test shim
    globalThis.window = { ...fakeWindow(store), AudioContext: AudioCtor };
    const { playSound } = await load();
    playSound("success");
    expect(AudioCtor).not.toHaveBeenCalled();
  });

  it("degrades quietly when the browser has no Web Audio", async () => {
    // @ts-expect-error test shim
    globalThis.window = fakeWindow();
    const { playSound } = await load();
    expect(() => playSound("error")).not.toThrow();
  });
});
