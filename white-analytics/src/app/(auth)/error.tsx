"use client";

import { ErrorPanel, type ErrorBoundaryProps } from "@/components/dashboard/error-panel";

export default function RouteError(props: ErrorBoundaryProps) {
  return <ErrorPanel {...props} fullPage />;
}
