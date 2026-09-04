"use client";

import * as React from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export type ShareTabItem = { key: string; label: React.ReactNode; content: React.ReactNode };

/** Client tabs shell — module content is server-rendered and passed in as slots. */
export function ShareTabs({ items }: { items: ShareTabItem[] }) {
  if (items.length === 0) return null;
  if (items.length === 1) return <div>{items[0]!.content}</div>;
  return (
    <Tabs defaultValue={items[0]!.key}>
      <TabsList className="mb-4">
        {items.map((it) => (
          <TabsTrigger key={it.key} value={it.key}>
            {it.label}
          </TabsTrigger>
        ))}
      </TabsList>
      {items.map((it) => (
        <TabsContent key={it.key} value={it.key}>
          {it.content}
        </TabsContent>
      ))}
    </Tabs>
  );
}
