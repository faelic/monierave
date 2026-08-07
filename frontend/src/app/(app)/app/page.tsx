import type { Metadata } from "next";

import { DashboardOverview } from "@/features/dashboard/dashboard-overview";

export const metadata: Metadata = {
  title: "Overview",
  description: "Review your Monierave accounts and recent posted activity.",
};

export default function DashboardPage() {
  return <DashboardOverview />;
}
