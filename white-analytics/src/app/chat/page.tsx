import type { Metadata } from "next";
import { ChatWidget } from "@/features/erp/sales/public/chat-widget";
import { ps } from "@/features/erp/sales/strings";

export const metadata: Metadata = { title: ps.chat.title, robots: { index: false, follow: false } };

/** Embeddable chat widget: fills its iframe (see the embed snippet in chatbot settings). */
export default function ChatEmbedPage() {
  return (
    <div className="flex h-dvh flex-col">
      <ChatWidget className="flex-1" />
    </div>
  );
}
