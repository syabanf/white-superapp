/** Bare layout for pages embedded in other websites (no app chrome). */
export default function EmbedLayout({ children }: LayoutProps<"/chat">) {
  return <div className="flex min-h-dvh flex-1 flex-col bg-card">{children}</div>;
}
