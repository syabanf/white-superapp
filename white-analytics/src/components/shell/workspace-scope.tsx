"use client";

import * as React from "react";
import { usePathname } from "next/navigation";
import { workspaceOf } from "@/components/shell/nav-config";

/**
 * Sets data-workspace so the ERP accent tokens apply under /erp. The wrapper
 * covers the first paint; mirroring it on <html> reaches portals (sheets,
 * dropdowns, dialogs) that render outside the wrapper.
 */
export function WorkspaceScope({ children }: { children: React.ReactNode }) {
  const workspace = workspaceOf(usePathname());
  React.useEffect(() => {
    document.documentElement.dataset.workspace = workspace;
    return () => {
      delete document.documentElement.dataset.workspace;
    };
  }, [workspace]);
  return (
    <div data-workspace={workspace} className="contents">
      {children}
    </div>
  );
}
