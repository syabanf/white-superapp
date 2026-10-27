"use client";

import { Section } from "@/components/dashboard/page-header";
import * as React from "react";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { cn } from "@/lib/utils";
import { regenerateMyDailyRecapAction, updateMyDailyNotesAction } from "../actions";
import { sp } from "../strings";
import type { DailyRecap } from "../types";
import { DailyRecapView, RecapHistoryList } from "./recaps";

const m = sp.myRecap;

export function MyRecap({ recap, history }: { recap: DailyRecap; history: DailyRecap[] }) {
  const { pending, run } = useErpAction();
  const [notes, setNotes] = React.useState(recap.notes);
  return (
    <div className="grid gap-(--gap) lg:grid-cols-[minmax(0,1fr)_360px]">
      <Card className="gap-4 p-(--card-pad)">
        <div className="flex justify-end">
          <Button
            size="sm"
            variant="outline"
            disabled={pending}
            onClick={() => run(() => regenerateMyDailyRecapAction(recap.recapDate), { success: m.regenerated })}
          >
            <RefreshCw className={cn("size-3.5", pending && "animate-spin")} /> {m.regenerate}
          </Button>
        </div>
        <DailyRecapView recap={recap} hideNotes />
        <section className="flex flex-col gap-2 border-t pt-4">
          <label htmlFor="my-notes" className="text-sm font-semibold">
            {sp.recaps.notes}
          </label>
          <Textarea
            id="my-notes"
            rows={4}
            className="text-base sm:text-sm"
            value={notes}
            placeholder={m.notesPlaceholder}
            onChange={(e) => setNotes(e.target.value)}
          />
          <Button
            size="sm"
            className="self-start"
            disabled={pending || notes === recap.notes}
            onClick={() => run(() => updateMyDailyNotesAction(recap.recapDate, notes), { success: m.notesSaved })}
          >
            {m.saveNotes}
          </Button>
        </section>
      </Card>
      <Section title={sp.recaps.history}>
        <RecapHistoryList recaps={history.filter((h) => h.recapDate !== recap.recapDate)} />
      </Section>
    </div>
  );
}
