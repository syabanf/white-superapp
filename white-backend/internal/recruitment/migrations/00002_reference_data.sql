-- +goose Up
-- The psikotes instruments NüHabit ships (20260715120000_psikotes_online).
-- Question banks are managed by HR; the seeder adds demo banks.
INSERT INTO recruitment.psikotes_instruments (id, code, name, kind, duration_seconds, question_count, shuffle, instructions, sort_order) VALUES
  ('psi_papi_kostick', 'papi_kostick', 'PAPI Kostick', 'forced_choice', 2400, NULL, false,
   'Pilih satu pernyataan (A atau B) yang paling menggambarkan diri Anda pada setiap pasangan. Tidak ada jawaban benar atau salah.', 10),
  ('psi_baum', 'baum', 'Baum (Tree Test)', 'drawing', 1200, NULL, false,
   'Gambarlah sebuah pohon berkayu pada kertas kosong, lalu foto dan unggah hasil gambar Anda.', 20),
  ('psi_dap', 'dap', 'DAP (Draw a Person)', 'drawing', 1200, NULL, false,
   'Gambarlah seorang manusia secara utuh, lalu foto dan unggah hasil gambar Anda.', 30),
  ('psi_wartegg', 'wartegg', 'Wartegg (WZT)', 'drawing', 1800, NULL, false,
   'Lengkapi 8 kotak stimulus menjadi gambar yang bermakna, lalu foto dan unggah hasil gambar Anda.', 40),
  ('psi_aritmatika', 'aritmatika', 'Aritmatika', 'mcq', 600, 10, true,
   'Kerjakan soal hitungan berikut. Waktu berjalan otomatis dan jawaban tersimpan setiap kali Anda memilih.', 50),
  ('psi_matematika', 'matematika', 'Matematika', 'mcq', 600, 10, true,
   'Kerjakan soal matematika berikut. Waktu berjalan otomatis dan jawaban tersimpan setiap kali Anda memilih.', 60)
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DELETE FROM recruitment.psikotes_instruments WHERE id LIKE 'psi\_%';
