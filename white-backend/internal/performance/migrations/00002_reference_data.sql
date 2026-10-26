-- +goose Up
-- Reference data the engines need on a fresh database. Inserts skip rows
-- that already exist so admin edits are never overwritten.

-- KPI indicators with an automatic collector, plus the supervisor rubric.
INSERT INTO performance.kpi_indicators (id, code, name, description, unit, direction, source_kind, default_target) VALUES
  ('ind_att_ontime', 'att_ontime', 'Kehadiran tepat waktu', 'Hari hadir tanpa terlambat dibagi hari kerja (dikurangi cuti).', '%', 'higher_better', 'auto', 0.95),
  ('ind_att_late_ratio', 'att_late_ratio', 'Rasio menit terlambat', 'Menit terlambat dibagi menit kerja terjadwal (8 jam per hari kerja).', '%', 'lower_better', 'auto', 0.01),
  ('ind_team_ontime', 'team_ontime', 'Ketepatan waktu tim', 'Rata-rata kehadiran tepat waktu anggota departemen.', '%', 'higher_better', 'auto', 0.9),
  ('ind_logbook_compliance', 'logbook_compliance', 'Kepatuhan logbook', 'Item checklist logbook yang dicentang dibagi total item.', '%', 'higher_better', 'auto', 0.9),
  ('ind_task_completion', 'task_completion', 'Penyelesaian tugas', 'Tugas disetujui dibagi tugas jatuh tempo sampai hari ini.', '%', 'higher_better', 'auto', 0.9),
  ('ind_supervisor_rubric', 'supervisor_rubric', 'Penilaian atasan', 'Rubrik atasan langsung skala 1 sampai 5.', 'skor', 'higher_better', 'manual', 5)
ON CONFLICT (code) DO NOTHING;

INSERT INTO performance.behavioral_standards
  (id, value_name, competency_name, standard_description, score_1_description, score_3_description, score_5_description, weight, sort_order) VALUES
  ('bs_disiplin', 'Disiplin', 'Kedisiplinan & Kehadiran', 'Hadir tepat waktu, menepati tenggat, patuh SOP agensi.',
   'Sering terlambat atau melewatkan tenggat', 'Umumnya disiplin, sesekali lalai', 'Selalu tepat waktu dan menepati tenggat', 20, 1),
  ('bs_inisiatif', 'Inisiatif', 'Inisiatif & Tanggung Jawab', 'Mengerjakan tugas tanpa menunggu perintah, menuntaskan sampai selesai.',
   'Pasif, harus selalu diarahkan', 'Bekerja baik dengan arahan normal', 'Proaktif, bisa diandalkan tanpa pengawasan', 20, 2),
  ('bs_kolaborasi', 'Kerja Sama', 'Kerja Sama Tim', 'Membantu rekan, komunikasi baik lintas tim kreatif, akun, dan media.',
   'Sulit bekerja sama', 'Kooperatif dalam tim', 'Perekat tim, membantu lintas departemen', 20, 3),
  ('bs_klien', 'Orientasi Klien', 'Orientasi Klien', 'Memahami kebutuhan klien, responsif, menjaga kualitas deliverable.',
   'Sering dikomplain klien', 'Layanan standar, jarang dikomplain', 'Dipuji klien, jadi contoh pelayanan', 20, 4),
  ('bs_integritas', 'Integritas', 'Integritas & Kejujuran', 'Jujur dalam laporan, transparan soal kendala dan angka performa.',
   'Pernah terbukti tidak jujur', 'Tidak ada catatan negatif', 'Sangat dipercaya memegang tanggung jawab sensitif', 20, 5)
ON CONFLICT (id) DO NOTHING;

INSERT INTO performance.performance_categories (id, category_name, min_score, max_score, description) VALUES
  ('pc_istimewa', 'Istimewa', 90, 100, 'Kandidat promosi / kenaikan istimewa'),
  ('pc_baik', 'Baik', 75, 89.99, 'Kinerja sehat, layak kenaikan normal'),
  ('pc_cukup', 'Cukup', 60, 74.99, 'Perlu peningkatan pada beberapa area'),
  ('pc_pembinaan', 'Perlu Pembinaan', 0, 59.99, 'Butuh rencana pembinaan (coaching plan)')
ON CONFLICT (category_name) DO NOTHING;

INSERT INTO performance.feedback_categories (id, code, name, description, weight, display_order) VALUES
  ('fc_leadership', 'leadership', 'Kepemimpinan', 'Memberi arah, mengambil keputusan, membimbing rekan.', 20, 1),
  ('fc_communication', 'communication', 'Komunikasi', 'Menyampaikan ide dengan jelas dan mendengarkan.', 20, 2),
  ('fc_collaboration', 'collaboration', 'Kolaborasi', 'Bekerja sama lintas tim dan berbagi pengetahuan.', 20, 3),
  ('fc_accountability', 'accountability', 'Akuntabilitas', 'Menepati komitmen dan bertanggung jawab atas hasil.', 20, 4),
  ('fc_problem_solving', 'problem_solving', 'Pemecahan Masalah', 'Menganalisa masalah dan menemukan solusi praktis.', 20, 5)
ON CONFLICT (code) DO NOTHING;

INSERT INTO performance.feedback_criteria (id, category_id, name, description, display_order) VALUES
  ('fcr_lead_1', 'fc_leadership', 'Memberi arahan yang jelas', 'Tujuan dan prioritas pekerjaan mudah dipahami.', 1),
  ('fcr_lead_2', 'fc_leadership', 'Membimbing rekan', 'Mau mengajari dan memberi masukan yang membangun.', 2),
  ('fcr_comm_1', 'fc_communication', 'Jelas dan ringkas', 'Pesan tertulis maupun lisan mudah dimengerti.', 1),
  ('fcr_comm_2', 'fc_communication', 'Mendengarkan', 'Memahami sudut pandang orang lain sebelum menjawab.', 2),
  ('fcr_coll_1', 'fc_collaboration', 'Membantu tim', 'Ringan tangan saat rekan membutuhkan bantuan.', 1),
  ('fcr_coll_2', 'fc_collaboration', 'Berbagi pengetahuan', 'Membagikan template, insight, dan pembelajaran.', 2),
  ('fcr_acc_1', 'fc_accountability', 'Menepati tenggat', 'Deliverable selesai sesuai jadwal yang disepakati.', 1),
  ('fcr_acc_2', 'fc_accountability', 'Mengakui kesalahan', 'Terbuka saat ada kendala dan segera memperbaiki.', 2),
  ('fcr_ps_1', 'fc_problem_solving', 'Analitis', 'Memakai data untuk memahami akar masalah.', 1),
  ('fcr_ps_2', 'fc_problem_solving', 'Solutif', 'Datang dengan usulan solusi, bukan hanya masalah.', 2)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM performance.feedback_criteria WHERE id LIKE 'fcr_%';
DELETE FROM performance.feedback_categories WHERE id LIKE 'fc_%';
DELETE FROM performance.performance_categories WHERE id LIKE 'pc_%';
DELETE FROM performance.behavioral_standards WHERE id LIKE 'bs_%';
DELETE FROM performance.kpi_indicators WHERE id LIKE 'ind_%' AND source_kind = 'auto' OR id = 'ind_supervisor_rubric';
