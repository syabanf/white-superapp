/** Shared option lists for client forms — importable from server actions and client components. */

export const CLIENT_CURRENCIES = ["IDR", "USD", "SGD", "AUD", "EUR"] as const;
export type ClientCurrency = (typeof CLIENT_CURRENCIES)[number];

export const CLIENT_TIMEZONES = [
  "Asia/Jakarta",
  "Asia/Makassar",
  "Asia/Jayapura",
  "Asia/Singapore",
  "Asia/Kuala_Lumpur",
  "Australia/Sydney",
] as const;
export type ClientTimezone = (typeof CLIENT_TIMEZONES)[number];

export const SHARE_MODULE_KEYS = ["SOCIAL", "SEO", "ADS"] as const;
export type ShareModuleKey = (typeof SHARE_MODULE_KEYS)[number];

export const SOCIAL_PLATFORMS = ["INSTAGRAM", "FACEBOOK", "TIKTOK"] as const;
export type SocialPlatformKey = (typeof SOCIAL_PLATFORMS)[number];

export const MEMBER_ROLES = ["MANAGER", "VIEWER"] as const;
export type MemberRoleKey = (typeof MEMBER_ROLES)[number];
