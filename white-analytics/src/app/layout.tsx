import type { Metadata, Viewport } from "next";
import { DM_Sans, Geist_Mono } from "next/font/google";
import { NuqsAdapter } from "nuqs/adapters/next/app";
import { ThemeProvider } from "next-themes";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/sonner";
import { ChartGradientDefs } from "@/components/dashboard/charts/chart-gradients";
import "./globals.css";

const dmSans = DM_Sans({
  variable: "--font-dm-sans",
  subsets: ["latin"],
  display: "swap",
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "Mixist",
    template: "%s · Mixist",
  },
  description:
    "Dashboard analisa & reporting Social Media, SEO, dan Meta Ads untuk klien agency WHITE.",
  applicationName: "Mixist",
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f8eef3" },
    { media: "(prefers-color-scheme: dark)", color: "#120c27" },
  ],
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="id"
      suppressHydrationWarning
      className={`${dmSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <ChartGradientDefs />
        <ThemeProvider
          attribute="class"
          defaultTheme="light"
          enableSystem
          disableTransitionOnChange
          // The theme script runs from the server HTML. On the client React 19 warns about
          // any <script> it renders, so mark it inert there (the tag hydrates with
          // suppressHydrationWarning).
          scriptProps={typeof window === "undefined" ? undefined : { type: "application/json" }}
        >
          <NuqsAdapter>
            <TooltipProvider delayDuration={200}>{children}</TooltipProvider>
          </NuqsAdapter>
          <Toaster richColors position="bottom-right" />
        </ThemeProvider>
      </body>
    </html>
  );
}
