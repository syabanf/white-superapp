"use client";

import { ErrorPanel, type ErrorBoundaryProps } from "@/components/dashboard/error-panel";

export default function AppError(props: ErrorBoundaryProps) {
  return <ErrorPanel {...props} />;
}
