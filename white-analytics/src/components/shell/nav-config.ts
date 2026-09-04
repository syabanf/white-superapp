import {
  BarChart3,
  Briefcase,
  CalendarDays,
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
  children?: { label: string; href: string; icon?: LucideIcon }[];
};

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
