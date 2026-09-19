"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * Login hero: a 3D marketing funnel drawn on a 2D canvas with hand-rolled
 * perspective (no WebGL, no dependency). Audience particles enter from three
 * channel arms (Social, SEO, Ads), spiral down the funnel and speed up as the
 * radius shrinks, then convert at the core with a pulse and a ripple. The
 * pointer tilts the scene. Colours come from the theme tokens.
 */
const CHANNELS = ["SOCIAL", "SEO", "ADS"] as const;
const PARTICLES = 420;
const RINGS = [0, 0.22, 0.45, 0.7, 1];
const FOCAL = 2.6;
const TAU = Math.PI * 2;

type Particle = { ch: number; t: number; a: number; speed: number; size: number; px: number; py: number };
type Projected = { x: number; y: number; scale: number; depth: number };

const funnelRadius = (t: number) => 0.1 + 0.9 * Math.pow(1 - t, 2.2);
const funnelY = (t: number) => -0.58 + 1.12 * Math.pow(t, 0.9);

function spawn(p: Particle, stagger: boolean) {
  p.ch = Math.floor(Math.random() * CHANNELS.length);
  p.t = stagger ? Math.random() : 0;
  p.a = (p.ch * TAU) / CHANNELS.length + (Math.random() - 0.5) * 0.9 + p.t * 9;
  p.speed = 0.1 + Math.random() * 0.12;
  p.size = 1.2 + Math.random() * 1.9;
  p.px = Number.NaN;
  p.py = Number.NaN;
}

export function MarketingFunnel3D({ className }: { className?: string }) {
  const ref = React.useRef<HTMLCanvasElement>(null);

  React.useEffect(() => {
    const canvas = ref.current;
    const ctx = canvas?.getContext("2d");
    if (!canvas || !ctx) return;

    const css = getComputedStyle(canvas);
    const brand = css.getPropertyValue("--brand").trim() || "#1b4de4";
    const ink = css.getPropertyValue("--muted-foreground").trim() || "#6b7280";

    let w = 0;
    let h = 0;
    const resize = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      w = canvas.clientWidth;
      h = canvas.clientHeight;
      canvas.width = Math.round(w * dpr);
      canvas.height = Math.round(h * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    };
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);
    resize();

    const particles: Particle[] = Array.from({ length: PARTICLES }, () => {
      const p = { ch: 0, t: 0, a: 0, speed: 0, size: 0, px: 0, py: 0 };
      spawn(p, true);
      return p;
    });
    const ripples: number[] = [];
    let pulse = 0;
    let yaw = 0;
    let tiltX = 0;
    let tiltY = 0;
    let targetX = 0;
    let targetY = 0;

    const onPointer = (e: PointerEvent) => {
      const r = canvas.getBoundingClientRect();
      targetX = ((e.clientX - r.left) / r.width - 0.5) * 2;
      targetY = ((e.clientY - r.top) / r.height - 0.5) * 2;
    };
    window.addEventListener("pointermove", onPointer, { passive: true });

    const project = (x: number, y: number, z: number): Projected => {
      const cosY = Math.cos(yaw + tiltX * 0.6);
      const sinY = Math.sin(yaw + tiltX * 0.6);
      const x1 = x * cosY - z * sinY;
      const z1 = x * sinY + z * cosY;
      const pitch = 0.42 + tiltY * 0.22;
      const y2 = y * Math.cos(pitch) - z1 * Math.sin(pitch);
      const z2 = y * Math.sin(pitch) + z1 * Math.cos(pitch);
      const scale = FOCAL / (FOCAL + z2);
      const unit = Math.min(w, h * 1.15) * 0.4;
      return { x: w / 2 + x1 * unit * scale, y: h / 2 + y2 * unit * scale, scale, depth: z2 };
    };

    const ring = (t: number, radius: number, alpha: number) => {
      ctx.beginPath();
      for (let i = 0; i <= 56; i++) {
        const a = (i / 56) * TAU;
        const p = project(Math.cos(a) * radius, funnelY(t), Math.sin(a) * radius);
        if (i === 0) ctx.moveTo(p.x, p.y);
        else ctx.lineTo(p.x, p.y);
      }
      ctx.globalAlpha = alpha;
      ctx.stroke();
    };

    let last = performance.now();
    let frame = 0;
    const tick = (now: number) => {
      const dt = Math.min(0.05, (now - last) / 1000);
      last = now;
      // Hidden instance (the other breakpoint's copy): keep the loop alive, skip the work.
      if (w === 0 || h === 0) {
        raf = requestAnimationFrame(tick);
        return;
      }
      yaw += dt * 0.12;
      tiltX += (targetX - tiltX) * 0.06;
      tiltY += (targetY - tiltY) * 0.06;
      ctx.clearRect(0, 0, w, h);

      // Funnel wireframe and channel labels
      ctx.strokeStyle = brand;
      ctx.lineWidth = 1;
      for (const t of RINGS) ring(t, funnelRadius(t), 0.22);
      ctx.font = "500 10px ui-monospace, SFMono-Regular, Menlo, monospace";
      ctx.textAlign = "center";
      ctx.fillStyle = ink;
      CHANNELS.forEach((label, i) => {
        const a = (i * TAU) / CHANNELS.length;
        const p = project(Math.cos(a) * 1.02, funnelY(0) - 0.09, Math.sin(a) * 1.02);
        ctx.globalAlpha = 0.35 + 0.65 * Math.max(0, Math.min(1, (1.2 - p.depth) / 2.4));
        ctx.fillText(label.split("").join(" "), p.x, p.y);
      });

      // Particles: advance, then draw far-to-near so near ones sit on top
      let arrivals = 0;
      for (const p of particles) {
        p.t += p.speed * dt;
        p.a += (0.55 / funnelRadius(p.t)) * dt;
        if (p.t >= 1) {
          arrivals++;
          spawn(p, false);
        }
      }
      const drawn = particles
        .map((p) => {
          const r = funnelRadius(p.t);
          return { p, at: project(Math.cos(p.a) * r, funnelY(p.t), Math.sin(p.a) * r) };
        })
        .sort((a, b) => b.at.depth - a.at.depth);
      ctx.fillStyle = brand;
      ctx.lineCap = "round";
      for (const { p, at } of drawn) {
        const nearness = Math.max(0, Math.min(1, (1.3 - at.depth) / 2.6));
        ctx.globalAlpha = (0.18 + 0.72 * nearness) * Math.min(1, p.t * 8);
        const size = p.size * at.scale;
        if (!Number.isNaN(p.px)) {
          ctx.lineWidth = size;
          ctx.beginPath();
          ctx.moveTo(p.px, p.py);
          ctx.lineTo(at.x, at.y);
          ctx.stroke();
        }
        ctx.beginPath();
        ctx.arc(at.x, at.y, size * 0.75, 0, TAU);
        ctx.fill();
        p.px = at.x;
        p.py = at.y;
      }

      // Conversion core: glow that swells with arrivals, plus expanding ripples
      pulse = Math.min(1.6, pulse * 0.92 + arrivals * 0.22);
      if (arrivals > 0 && ripples.length < 4 && frame % 18 === 0) ripples.push(0);
      const core = project(0, funnelY(1), 0);
      const glow = (16 + pulse * 16) * core.scale;
      const gradient = ctx.createRadialGradient(core.x, core.y, 0, core.x, core.y, glow);
      gradient.addColorStop(0, brand);
      gradient.addColorStop(1, /^#[0-9a-f]{6}$/i.test(brand) ? `${brand}00` : "transparent");
      ctx.globalAlpha = 0.55;
      ctx.fillStyle = gradient;
      ctx.beginPath();
      ctx.arc(core.x, core.y, glow, 0, TAU);
      ctx.fill();
      ctx.lineWidth = 1.25;
      for (let i = ripples.length - 1; i >= 0; i--) {
        ripples[i]! += dt * 0.55;
        const age = ripples[i]!;
        if (age >= 1) ripples.splice(i, 1);
        else ring(1, 0.1 + age * 0.62, (1 - age) * 0.5);
      }

      ctx.globalAlpha = 1;
      frame++;
      raf = requestAnimationFrame(tick);
    };
    let raf = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(raf);
      observer.disconnect();
      window.removeEventListener("pointermove", onPointer);
    };
  }, []);

  return <canvas ref={ref} aria-hidden className={cn("block h-full w-full", className)} />;
}
