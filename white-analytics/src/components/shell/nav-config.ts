import {
  BarChart3,
  Banknote,
  Bot,
  Building2,
  FileSignature,
  HandCoins,
  Landmark,
  Presentation,
  Receipt,
  Rocket,
  Target,
  UserCircle,
  UserPlus,
  Workflow,
  Briefcase,
  CalendarDays,
  ClipboardList,
  FolderOpen,
  Kanban,
  Palette,
  PenSquare,
  Link2,
  Swords,
  TrendingUp,
  Lightbulb,
  Plug,
  FileText,
  Gauge,
  Globe,
  KeyRound,
  BookOpen,
  LayoutDashboard,
  Megaphone,
  Search,
  Settings,
  Share2,
  ShieldCheck,
  Users,
  type LucideIcon,
} from "lucide-react";
import { t } from "@/i18n/id";

export type NavItem = {
  label: string;
  href: string;
  icon: LucideIcon;
  /** exact match vs prefix match for active state */
  exact?: boolean;
  children?: { label: string; href: string; icon?: LucideIcon; perm?: string | string[] }[];
  /** ERP permission key needed to see the item (ADMIN sees everything); a list means any of them */
  perm?: string | string[];
};

export type Workspace = "marketing" | "erp";

/** The ERP workspace lives under /erp; everything else is marketing. */
export function workspaceOf(pathname: string): Workspace {
  return pathname === "/erp" || pathname.startsWith("/erp/") ? "erp" : "marketing";
}

export type ErpAccess = { role: string; permissions: string[] };

/** Not a grantable key (identity rejects it), so only ADMIN passes canAccess. */
const ADMIN_ONLY = "admin.only";

export function canAccess(access: ErpAccess | null, perm?: string | string[]): boolean {
  if (Array.isArray(perm)) return perm.some((p) => canAccess(access, p));
  if (!perm) return true;
  if (!access) return false;
  if (access.role === "ADMIN") return true;
  const moduleKey = perm.split(".")[0];
  return access.permissions.includes(perm) || access.permissions.includes(`${moduleKey}.*`);
}

const e = t.erpNav;

/** ERP menu, grouped. Items and children are filtered by permission. */
export const erpNav: { label: string; items: NavItem[] }[] = [
  {
    label: t.workspace.erp,
    items: [{ label: e.home, href: "/erp", icon: LayoutDashboard, exact: true }],
  },
  {
    label: e.finance,
    items: [
      {
        label: e.finance,
        href: "/erp/finance",
        icon: Landmark,
        perm: "finance.read",
        children: [
          { label: e.financeAnalysis, href: "/erp/finance/analysis" },
          { label: e.journals, href: "/erp/finance/journals" },
          { label: e.accounts, href: "/erp/finance/accounts" },
          { label: e.cashBank, href: "/erp/finance/cash-bank" },
          { label: e.payables, href: "/erp/finance/payables" },
          { label: e.receivables, href: "/erp/finance/receivables" },
          { label: e.financeReports, href: "/erp/finance/reports" },
          { label: e.fiscal, href: "/erp/finance/fiscal" },
        ],
      },
      { label: e.reimbursements, href: "/erp/finance/reimbursements", icon: Receipt, perm: "finance.read" },
    ],
  },
  {
    label: e.hr,
    items: [
      {
        label: e.hr,
        href: "/erp/hr",
        icon: Building2,
        perm: "hr.read",
        children: [
          { label: e.employees, href: "/erp/hr/employees" },
          { label: e.attendance, href: "/erp/hr/attendance" },
          { label: e.leave, href: "/erp/hr/leave" },
          { label: e.overtime, href: "/erp/hr/overtime" },
          { label: e.contracts, href: "/erp/hr/contracts" },
          { label: e.announcements, href: "/erp/hr/announcements" },
        ],
      },
      { label: e.payroll, href: "/erp/hr/payroll", icon: Banknote, perm: "hr.payroll" },
      { label: e.recruitment, href: "/erp/hr/recruitment", icon: UserPlus, perm: "recruitment.read" },
      {
        label: e.performance,
        href: "/erp/performance",
        icon: Target,
        // The overview serves employees too (scope "saya"); team pages need hr.read.
        perm: ["hr.read", "self.service"],
        children: [
          { label: e.tasks, href: "/erp/performance/tasks", perm: "hr.read" },
          { label: e.kpi, href: "/erp/performance/kpi", perm: "hr.read" },
          { label: e.reviews, href: "/erp/performance/reviews", perm: "hr.read" },
          { label: e.feedback360, href: "/erp/performance/feedback", perm: "hr.read" },
          { label: e.logbook, href: "/erp/performance/logbook", perm: "hr.read" },
          { label: e.recaps, href: "/erp/performance/recaps", perm: "hr.read" },
          { label: e.improvements, href: "/erp/performance/improvements", perm: "hr.read" },
        ],
      },
    ],
  },
  {
    label: e.sales,
    items: [
      {
        label: e.sales,
        href: "/erp/sales",
        icon: HandCoins,
        perm: "sales.read",
        children: [
          { label: e.leads, href: "/erp/sales/leads" },
          { label: e.pipeline, href: "/erp/sales/pipeline" },
          { label: e.quotations, href: "/erp/sales/quotations" },
        ],
      },
      { label: e.salesInvoices, href: "/erp/sales/invoices", icon: Receipt, perm: "sales.read" },
      { label: e.salesReports, href: "/erp/sales/reports", icon: BarChart3, perm: "sales.read" },
      { label: e.clientContracts, href: "/erp/sales/contracts", icon: FileSignature, perm: "sales.read" },
      { label: e.pitchDecks, href: "/erp/sales/pitch-decks", icon: Presentation, perm: "sales.read" },
      { label: e.onboarding, href: "/erp/sales/onboarding", icon: Rocket, perm: "sales.read" },
      { label: e.chatbot, href: "/erp/sales/chatbot", icon: Bot, perm: "sales.read" },
      { label: e.leadForms, href: "/erp/sales/forms", icon: ClipboardList, perm: "sales.read" },
      { label: e.salesAutomation, href: "/erp/sales/automation", icon: Workflow, perm: ADMIN_ONLY },
    ],
  },
  {
    label: e.me,
    items: [
      {
        label: e.me,
        href: "/erp/me",
        icon: UserCircle,
        perm: "self.service",
        children: [
          { label: e.myAttendance, href: "/erp/me/attendance" },
          { label: e.myLeave, href: "/erp/me/leave" },
          { label: e.myOvertime, href: "/erp/me/overtime" },
          { label: e.myLoans, href: "/erp/me/loans" },
          { label: e.myReimburse, href: "/erp/me/reimbursements" },
          { label: e.reimburseApprovals, href: "/erp/finance/reimbursements" },
          { label: e.payslips, href: "/erp/me/payslips" },
          { label: e.myTasks, href: "/erp/me/tasks" },
          { label: e.myRecap, href: "/erp/me/recap" },
        ],
      },
    ],
  },
];

/** ERP groups with items and children the user may see; empty groups dropped. */
export function visibleErpNav(access: ErpAccess | null) {
  return erpNav
    .map((g) => ({
      label: g.label,
      items: g.items
        .filter((it) => canAccess(access, it.perm))
        .map((it) => ({ ...it, children: it.children?.filter((c) => canAccess(access, c.perm ?? it.perm)) })),
    }))
    .filter((g) => g.items.length > 0);
}

export const generalNav: NavItem[] = [
  { label: t.nav.portfolio, href: "/", icon: LayoutDashboard, exact: true },
  { label: t.nav.clients, href: "/clients", icon: Briefcase, exact: true },
  { label: t.nav.guide, href: "/panduan", icon: BookOpen, exact: true },
];

export function clientNav(slug: string): NavItem[] {
  const base = `/clients/${slug}`;
  return [
    { label: t.nav.overview, href: base, icon: Gauge, exact: true },
    {
      label: t.nav.social,
      href: `${base}/social`,
      icon: Share2,
      children: [{ label: t.nav.competitors, href: `${base}/social/competitors`, icon: Users }],
    },
    {
      label: t.nav.seo,
      href: `${base}/seo`,
      icon: Search,
      children: [
        { label: t.nav.keywords, href: `${base}/seo/keywords`, icon: KeyRound },
        { label: t.nav.pages, href: `${base}/seo/pages`, icon: Globe },
        { label: t.nav.research, href: `${base}/seo/research`, icon: Lightbulb },
        { label: t.nav.rank, href: `${base}/seo/rank`, icon: TrendingUp },
        { label: t.nav.backlinks, href: `${base}/seo/backlinks`, icon: Link2 },
        { label: t.nav.seoCompetitors, href: `${base}/seo/competitors`, icon: Swords },
        { label: t.nav.audit, href: `${base}/seo/audit`, icon: ShieldCheck },
      ],
    },
    {
      label: t.nav.publish,
      href: `${base}/publish`,
      icon: CalendarDays,
      children: [
        { label: t.nav.posts, href: `${base}/publish/posts`, icon: FileText },
        { label: t.nav.media, href: `${base}/publish/media`, icon: Globe },
      ],
    },
    {
      label: t.nav.content,
      href: `${base}/content`,
      icon: PenSquare,
      children: [
        { label: t.nav.contentPlans, href: `${base}/content/plans`, icon: ClipboardList },
        { label: t.nav.contentBoard, href: `${base}/content/board`, icon: Kanban },
        { label: t.nav.contentDesign, href: `${base}/content/design`, icon: Palette },
        { label: t.nav.contentAssets, href: `${base}/content/assets`, icon: FolderOpen },
        { label: t.nav.contentReports, href: `${base}/content/reports`, icon: FileText },
      ],
    },
    { label: t.nav.ads, href: `${base}/ads`, icon: Megaphone },
    { label: t.nav.reports, href: `${base}/reports`, icon: FileText },
    { label: t.nav.settings, href: `${base}/settings`, icon: Settings },
  ];
}

export const adminNav: NavItem[] = [
  { label: t.nav.setup, href: "/setup", icon: Plug, exact: true },
  { label: t.nav.users, href: "/admin/users", icon: Users },
];

export const moduleIcons = { social: Share2, seo: Search, ads: Megaphone, analytics: BarChart3 };

/** Whether a pathname is "under" a nav href */
export function isActivePath(pathname: string, href: string, exact = false): boolean {
  if (exact) return pathname === href;
  return pathname === href || pathname.startsWith(href + "/");
}
