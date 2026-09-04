import Link from "next/link";
import { Button } from "@/components/ui/button";
import { WhiteLogo } from "@/components/brand/logo";

export default function NotFound() {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center gap-4 bg-background px-6 text-center">
      <WhiteLogo size={28} />
      <h1 className="text-2xl font-semibold tracking-tight">Halaman tidak ditemukan</h1>
      <p className="max-w-md text-sm text-muted-foreground">
        Alamat yang Anda buka tidak tersedia. Periksa kembali tautannya atau kembali ke portofolio.
      </p>
      <Button asChild>
        <Link href="/">Kembali ke portofolio</Link>
      </Button>
    </main>
  );
}
