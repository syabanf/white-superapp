"use client";

import { toast as sonner } from "sonner";
import { playSound } from "@/lib/sound";

type Msg = Parameters<typeof sonner.success>[0];
type Opts = Parameters<typeof sonner.success>[1];

/**
 * Drop-in replacement for sonner's `toast` that also plays the matching cue.
 * Import this everywhere instead of sonner, so a notification can never appear
 * without its sound (or vice versa).
 */
export const toast = {
  success: (message: Msg, opts?: Opts) => {
    playSound("success");
    return sonner.success(message, opts);
  },
  error: (message: Msg, opts?: Opts) => {
    playSound("error");
    return sonner.error(message, opts);
  },
  warning: (message: Msg, opts?: Opts) => {
    playSound("error");
    return sonner.warning(message, opts);
  },
  info: (message: Msg, opts?: Opts) => {
    playSound("notify");
    return sonner.info(message, opts);
  },
  message: (message: Msg, opts?: Opts) => {
    playSound("notify");
    return sonner.message(message, opts);
  },
  dismiss: sonner.dismiss,
  loading: sonner.loading,
  promise: sonner.promise,
};
