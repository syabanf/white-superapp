import { AsyncLocalStorage } from "node:async_hooks";

/**
 * Mode demo per klien. Pekerjaan untuk klien bertanda `isDemo` dijalankan di
 * dalam `runInDemoMode`; selama itu semua pengecekan "sudah dikonfigurasi?"
 * pada provider (Meta, Google, Apify, DataForSEO, OpenRouter, PageSpeed)
 * menjawab false, sehingga adapter mock, DemoBanner, dan koneksi simulasi
 * dipakai otomatis walau kredensial asli terpasang di server.
 *
 * Tanpa konteks (klien biasa, halaman admin) perilaku tidak berubah.
 */
const store = new AsyncLocalStorage<{ demo: boolean }>();

export function inDemoMode(): boolean {
  return store.getStore()?.demo === true;
}

/** Jalankan `fn` dengan mode demo aktif bila `demo` true; selain itu apa adanya. */
export function runInDemoMode<T>(demo: boolean, fn: () => T): T {
  return store.run({ demo }, fn);
}
