/**
 * Builder form state (plain module: importable from server pages and client
 * components alike).
 */
import type { DraftDTO } from "@/features/campaigns/queries";
import { CREATIVE_SLOTS, DEFAULT_AUDIENCE, OBJECTIVE_DEFAULTS, emptyCreative, isObjective, type Audience, type Creative, type DraftInput, type Objective, type Placement } from "@/features/campaigns/lib";

export type BuilderState = {
  name: string;
  objective: Objective;
  dailyBudget: number;
  startDate: string;
  /** "" = open-ended */
  endDate: string;
  adAccountId: string | null;
  audience: Audience;
  placements: Placement[];
  creatives: Creative[];
  aiNotes: string;
};

export type SetField = <K extends keyof BuilderState>(key: K, value: BuilderState[K]) => void;

export function initialBuilderState(initial: DraftDTO | null, opts: { today: string; defaultAdAccountId: string | null; defaultBudget: number }): BuilderState {
  if (initial) {
    return {
      name: initial.name,
      objective: isObjective(initial.objective) ? initial.objective : "OUTCOME_TRAFFIC",
      dailyBudget: initial.dailyBudget,
      startDate: initial.startDate,
      endDate: initial.endDate ?? "",
      adAccountId: initial.adAccountId,
      audience: initial.audience,
      placements: initial.placements,
      creatives: initial.creatives,
      aiNotes: initial.aiNotes,
    };
  }
  return {
    name: "",
    objective: "OUTCOME_TRAFFIC",
    dailyBudget: opts.defaultBudget,
    startDate: opts.today,
    endDate: "",
    adAccountId: opts.defaultAdAccountId,
    audience: { ...DEFAULT_AUDIENCE },
    placements: [...OBJECTIVE_DEFAULTS.OUTCOME_TRAFFIC.placements],
    creatives: Array.from({ length: CREATIVE_SLOTS }, emptyCreative),
    aiNotes: "",
  };
}

export const toDraftInput = (s: BuilderState, currency: string): DraftInput => ({
  ...s,
  currency,
  endDate: s.endDate || null,
});
