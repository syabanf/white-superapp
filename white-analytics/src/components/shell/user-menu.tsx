"use client";

import * as React from "react";
import { useTheme } from "next-themes";
import { ChevronsUpDown, LogOut, Monitor, Moon, ShieldCheck, Sun, UserRound, Volume2, VolumeX } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenuCheckboxItem,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { logoutAction } from "@/features/auth/actions";
import { isSoundEnabled, playSound, setSoundEnabled, soundDisabledSnapshot, subscribeSound } from "@/lib/sound";
import { initials } from "@/lib/format";
import { t } from "@/i18n/id";

export type ShellUser = { id: string; name: string; email: string; role: "ADMIN" | "MEMBER" };

export function UserMenu({ user, collapsed }: { user: ShellUser; collapsed?: boolean }) {
  const { theme, setTheme } = useTheme();
  // Read straight from the store — no effect, and other tabs stay in sync.
  const sound = React.useSyncExternalStore(subscribeSound, isSoundEnabled, soundDisabledSnapshot);
  const toggleSound = (on: boolean) => {
    setSoundEnabled(on);
    if (on) playSound("notify"); // confirm audibly that it is back on
  };
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton size="lg" className="data-[state=open]:bg-sidebar-accent">
              <Avatar className="size-8 rounded-lg">
                <AvatarFallback className="rounded-lg bg-primary text-xs font-semibold text-primary-foreground">
                  {initials(user.name || user.email)}
                </AvatarFallback>
              </Avatar>
              {!collapsed ? (
                <>
                  <span className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{user.name}</span>
                    <span className="truncate text-xs text-muted-foreground">{user.email}</span>
                  </span>
                  <ChevronsUpDown className="ml-auto size-4" />
                </>
              ) : null}
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="top" align="start" className="w-64">
            <DropdownMenuLabel className="flex items-center gap-2 font-normal">
              <Avatar className="size-8 rounded-lg">
                <AvatarFallback className="rounded-lg text-xs font-semibold">{initials(user.name || user.email)}</AvatarFallback>
              </Avatar>
              <span className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{user.name}</span>
                <span className="truncate text-xs text-muted-foreground">{user.email}</span>
              </span>
              <span className="inline-flex items-center gap-1 rounded-full border px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                {user.role === "ADMIN" ? <ShieldCheck className="size-3" /> : <UserRound className="size-3" />}
                {user.role === "ADMIN" ? t.admin.roleAdmin : t.admin.roleMember}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuLabel className="text-xs text-muted-foreground">{t.common.theme}</DropdownMenuLabel>
              <DropdownMenuRadioGroup value={theme ?? "system"} onValueChange={setTheme}>
                <DropdownMenuRadioItem value="light">
                  <Sun className="size-4" /> {t.common.themeLight}
                </DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="dark">
                  <Moon className="size-4" /> {t.common.themeDark}
                </DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="system">
                  <Monitor className="size-4" /> {t.common.themeSystem}
                </DropdownMenuRadioItem>
              </DropdownMenuRadioGroup>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuCheckboxItem checked={sound} onCheckedChange={toggleSound}>
              {sound ? <Volume2 className="size-4" /> : <VolumeX className="size-4" />} {t.common.sound}
            </DropdownMenuCheckboxItem>
            <DropdownMenuSeparator />
            <form action={logoutAction}>
              <DropdownMenuItem asChild variant="destructive">
                <button type="submit" className="w-full">
                  <LogOut className="size-4" /> {t.common.logout}
                </button>
              </DropdownMenuItem>
            </form>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
