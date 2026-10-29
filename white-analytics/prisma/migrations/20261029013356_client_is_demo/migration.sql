-- AlterTable
ALTER TABLE "Client" ADD COLUMN     "isDemo" BOOLEAN NOT NULL DEFAULT false;

-- Klien contoh dari seed tetap memakai data demo setelah kredensial asli dipasang.
UPDATE "Client" SET "isDemo" = true WHERE "slug" IN ('kopi-nusantara', 'adiharjo-property', 'bali-villa-escapes');
