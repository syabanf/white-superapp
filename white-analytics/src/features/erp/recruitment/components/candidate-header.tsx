"use client";

import { useRouter } from "next/navigation";
import { ArrowRightLeft, Loader2, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ConfirmAction } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { deleteCandidateAction, moveStageAction } from "../actions";
import { CANDIDATE_STATUS, STAGE_ORDER, rs } from "../strings";
import type { Candidate } from "../types";

/** Stage picker; also used on pipeline cards. */
export function StageMenu({ candidate, size = "default" }: { candidate: Pick<Candidate, "id" | "status">; size?: "default" | "xs" }) {
  const { run, pending } = useErpAction();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button size={size} variant="outline" disabled={pending}>
          {pending ? <Loader2 className="animate-spin" /> : <ArrowRightLeft />}
          {rs.candidates.moveStage}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>{rs.candidates.moveStage}</DropdownMenuLabel>
        {STAGE_ORDER.filter((s) => s !== candidate.status).map((s) => (
          <DropdownMenuItem key={s} onSelect={() => void run(() => moveStageAction(candidate.id, s), { success: rs.candidates.moved })}>
            {CANDIDATE_STATUS[s]}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function DeleteCandidateButton({ id }: { id: string }) {
  const router = useRouter();
  return (
    <ConfirmAction
      title={rs.candidates.deleteTitle}
      description={rs.candidates.deleteBody}
      confirmLabel={rs.delete}
      success={rs.candidates.deleted}
      action={async () => {
        const res = await deleteCandidateAction(id);
        if (res.ok) router.push("/erp/hr/recruitment/candidates");
        return res;
      }}
      trigger={
        <Button size="icon" variant="ghost" className="text-destructive" aria-label={rs.delete}>
          <Trash2 className="size-4" />
        </Button>
      }
    />
  );
}
