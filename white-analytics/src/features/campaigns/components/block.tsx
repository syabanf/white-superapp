import type { ReactNode } from "react";

/** Numbered builder step card (same shape as the publishing composer blocks). */
export function Block({ id, index, title, actions, children }: { id?: string; index?: string; title: string; actions?: ReactNode; children: ReactNode }) {
  return (
    <section id={id} className="scroll-mt-24 rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
      <header className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className="flex items-baseline gap-2 text-sm font-semibold tracking-[-0.01em]">
          {index ? <span className="label-mono text-brand">{index}</span> : null} {title}
        </h2>
        {actions}
      </header>
      {children}
    </section>
  );
}

/** Field-level messages, shown after the first save/submit attempt. */
export function FieldIssues({ messages }: { messages: string[] }) {
  if (!messages.length) return null;
  return (
    <ul role="alert" className="mt-2 space-y-1 text-xs text-destructive">
      {messages.map((m) => (
        <li key={m}>{m}</li>
      ))}
    </ul>
  );
}
