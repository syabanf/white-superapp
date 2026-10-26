NüHabit Finance and HR inventory (read-only; repo /Users/syabanf/Documents/works/new-nuhabit)

Platform facts that apply to every module below:
- **Go stack:** Go 1.26, stdlib `net/http` ServeMux with method patterns (`"GET /api/x/{id}"`, `{path...}`). There is no router library. pgx/v5 talks to raw SQL with no ORM. Other libraries: go-pdf/fpdf, pdfcpu and ledongthuc/pdf (behind `platform/pdfgen`), excelize (behind `platform/xlsx`), x/crypto.
- **Module contract** (`backend/internal/platform/module`): each module implements `Name()` and `Routes() []module.Route{Pattern, Handler}`. Dependencies arrive as `Deps{DB *pgxpool.Pool, Auth, Log, Now, Config, Events *outbox.Bus, Metrics}`. Modules are registered in `backend/internal/app/module_<name>.go`, and cross-module ports are wired in `backend/internal/app/adapters_<name>.go`.
- **Background jobs:** registered with `app.RegisterJob(module, name, every, svcFactory, method)`. Each job runs on an interval, and the per-day send log `platform.reminder_sends` (via `jobs.ClaimSend`) keeps a rerun from sending twice.
- **Layout pattern:** each module is a flat package plus a pure `domain/` subpackage. There is no domain/app/adapters split. The package holds the handlers (`http*.go`, `routes.go`), services (`service*.go`), SQL stores (`postgres_*.go`, `pg.go`), `ports.go` (interfaces to other modules) and `input.go` (zod-like validation through `platform/validate`). Almost every file is commented as a port of a TypeScript lib under `frontend/src/lib/**`.
- **Auth** (`platform/auth`) uses a session cookie, or a Bearer API token with scopes.
  - `auth.User{ID, Email, Role, FullName, BrandID, CompanyID, BranchID}`.
  - Guards: `RequireUser`, `RequireMenuPrefix(r, iam.X...)`, `RequireMenuAction(action, prefixes)` and `auth.Staff(guard, fn, iam.Can("read", iam.Tasks), …)`.
  - Permissions are IAM menu codes in `iam.menus` + `iam.role_menu_permissions` (each with JSON `permission_context.actions` such as read/create/update/delete/import/export), reached through `iam.user_roles`. Prefix constants are in `backend/internal/platform/iam/prefixes_gen.go`: `accounting`, `hris`, `hris.kepegawaian`, `hris.workforce`, `hris.compensation`, `hris.performance`, `hris.recruitment`, `hris.master`, `hris.insights`, `hris.performance.kpi-scorecard|kpi-config`, `tasks`, `tasks.team`, `tasks.kpi`, `settings.users`.
  - On top of IAM, the code also checks hard-coded role strings (see each module).
- **Tenant scope** (`platform/scope`): `Scope{Role, BusinessScope: holding|company|branch, HoldingID, CompanyID, BranchID, Unscoped}`, loaded from `configuration.users`. A user is unscoped if they are super_admin or have no business_scope. Helpers: `EffectiveCompanyID`, `CompanyFilter`, `BranchFilter`, `RowInScope`.
- **Audit:** `platform/audit.Write` appends to `audit.audit_log` (actor, action, entity, before/after, reason, IP, UA) inside the caller's transaction.
- **Outbox:** `platform/outbox` Bus provides `Subscribe(topic, subscriberName, handler)` for after-commit work (journals, WhatsApp).
- **Migrations** live in `backend/database/migrations/` as `bootstrap/` (functions, triggers, FKs), `schemas/<schema>/000…_table_*.sql` (baseline) and `deltas/<timestamp>_*.sql` (415 files). There is no up/down pairing. Schemas used here: `accounting`, `hris`, `performance`, `recruitment`, `configuration`, `iam`, `audit`, `public`.

========================================================
## 1. ACCOUNTING — `backend/internal/modules/accounting/`
========================================================
**Size:** 45 files, 12,714 Go lines including tests. Largest files: `service_fiscal.go` 695, `service_setup.go` 656, `service_ap.go` 649, `journal.go` 624, `service_reports.go` 551, `http_setup.go` 533, `http_ops.go` 520.

**Layout:**
- `domain/`: journal, fiscal, reports, receivables, mapping, operations, accountcode, coaimport, scope, num.
- Services: `service_{setup,fiscal,opening,cashbank,ap,ar,reports,subledger,finance,payroll,production,gym,wallet,operations}.go`.
- Posting and events: `posting.go`, `events.go`.
- Imports: `coa_import.go` (XLSX).
- `faktur_pajak.go` handles Faktur Pajak file upload and download.

**Routes** (`http.go` `Routes()`). Every `/api/accounting` route goes through `h.staff` = `RequireMenuPrefix(iam.Accounting)`. Company comes from `domain.RequireCompany(scope)`: the user's `configuration.users.company_id` is required, otherwise a 400 with "Akun Anda belum terikat company…". Branch is never used. Global records (`company_id NULL`) may only be mutated by unscoped users.
- account-types: GET/POST `/api/accounting/account-types`, PUT/DELETE `/account-types/{id}`
- chart-of-accounts: GET/POST `/chart-of-accounts`, PUT/DELETE `/chart-of-accounts/{id}`, POST `/chart-of-accounts/import`
- journal-mappings: GET/POST `/journal-mappings`, GET/PUT/DELETE `/journal-mappings/{id}`
- fiscal-years: GET/POST `/fiscal-years`, GET/PUT/DELETE `/fiscal-years/{id}`, GET/POST `/fiscal-years/{id}/beginning-balance`
- fiscal-periods: GET `/fiscal-periods`, POST `/fiscal-periods/{id}/open`, GET `/fiscal-periods/{id}/close` (preview), POST `/fiscal-periods/{id}/close`
- journal-entries: GET/POST `/journal-entries`, GET/PUT/DELETE `/journal-entries/{id}`, POST `/journal-entries/{id}/post`
- cash-bank: GET `/cash-bank`, GET `/cash-bank/accounts-options`, GET `/cash-bank/{accountId}/ledger`, GET/POST `/cash-bank/cash-in`, GET/POST `/cash-bank/cash-out`, GET/POST `/cash-bank/transfer`
- reports: GET `/reports/{report}` (trial-balance, balance-sheet, income-statement, cash-flow, general-ledger), GET `/dashboard`, GET `/ledger/subsidiary`
- AP: GET `/ap/invoices`, GET `/ap/invoices/{id}`, GET `/ap/payable`, GET `/ap/aging`, GET/POST `/ap/payments`, POST `/ap/payments/{id}/void`
- AR: GET `/ar/invoices`, GET `/ar/receivable`, GET `/ar/aging`, GET `/ar/by-sales-invoice/{salesInvoiceId}`, GET/POST `/ar/receipts`
- `/api/finance` (guard `h.finance`, CRM-deal scoped through `FinanceUser.hasCompanyScope`/`dealForbidden`):
  - GET `/invoices`, GET/PATCH `/invoices/{id}`
  - GET/POST `/invoices/{id}/payments` (POST answers "gone")
  - DELETE `/payments/{paymentId}`
  - GET/POST/DELETE `/invoices/{id}/faktur-pajak` (GET uses the `financeViewer` guard)

**Tables:**
- From `deltas/20260720120000_accounting_chart_of_accounts.sql`:
  - `accounting.account_types(id, code UNIQUE varchar30, name, normal_balance DEBIT|CREDIT, sort_order, is_active, audit cols)`
  - `accounting.chart_of_accounts(id, company_id→configuration.companies, code varchar20, name, parent_id self-FK, account_type_id, level 1..4, is_postable, is_contra, cash_flow_category NULL|OPERATING|INVESTING|FINANCING|NON_CASH, description, is_active, created/updated/deleted_at/by)`
  - Later delta adds `is_cash_bank bool`.
  - Unique `(company_id, code)` where not deleted; global unique `code` where `company_id IS NULL`.
- From `20260803130000_accounting_journal_mappings.sql`:
  - `accounting.journal_mappings(id, company_id, event_code varchar60, name, description, module, is_active, soft delete)`. The module CHECK grew to `POS|PURCHASING|PAYROLL|PINJAMAN|SALES|INVENTORY`. Unique `(company_id, event_code)`.
  - `accounting.journal_mapping_lines(id, mapping_id, entry_side DEBIT|CREDIT, line_role varchar40, account_id, amount_source TOTAL|SUBTOTAL|TAX|COGS|PAID|DISCOUNT|SERVICE_CHARGE, sort_order, is_required)`.
- From `20260803160000_accounting_fiscal_journal_entries.sql`:
  - `accounting.fiscal_years(id, company_id, code, name, start_date, end_date, is_active, soft delete)`.
  - `accounting.fiscal_periods(id, fiscal_year_id, period_no 1..12, name, start_date, end_date, status OPEN|CLOSED, UNIQUE(fiscal_year_id, period_no))`.
  - `accounting.journal_entries(id, company_id, entry_no, entry_date, description, fiscal_period_id NOT NULL, status DRAFT|POSTED, is_recon, posted_at/by, soft delete)`. Later deltas add `entry_type MANUAL|OPENING|AUTO`, `source_module`, `source_event_code`, `source_document_type`, `source_document_id` and `source_fiscal_year_id`.
  - `accounting.journal_entry_lines(id, entry_id, account_id, entry_side, amount numeric(18,2) >0, memo, sort_order)`.
- From `20260812120000_accounting_ap_documents.sql`:
  - `accounting.ap_invoices(id, company_id, invoice_no, invoice_date, due_date, vendor_id|supplier_id (one required), purchase_order_id, grn_id UNIQUE, currency IDR, subtotal, tax_amount, total_amount, status DRAFT|POSTED|VOID, description, posted_*)`.
  - `accounting.ap_payments(id, company_id, payment_no, payment_date, amount>0, method cash|bank_transfer|giro|qris|other, reference_number, notes, status, vendor_id, supplier_id, purchase_order_id, vendor_payment_id, posted_*)`. Adds `void_reason`, `voided_at`, `voided_by`.
  - `accounting.ap_payment_allocations(payment_id, invoice_id, amount, UNIQUE pair)`.
- From `20260812140000_accounting_ar_documents.sql`:
  - `accounting.ar_invoices(…, customer_name, sales_invoice_id→crm.crm_sales_invoices UNIQUE, deal_id→crm.crm_sales_deals, …status DRAFT|POSTED|VOID)`.
  - `accounting.ar_receipts(…, method cash|transfer|qris|edc|lainnya|bank_transfer|other, deal_payment_id)`.
  - `accounting.ar_receipt_allocations`.
- Other menu and backfill deltas: `20260803*` (coa_is_cash_bank, journal mapping backfills, payroll/pinjaman mappings), `20260803200000_accounting_beginning_balance.sql`, `20260808080000_journal_entries_auto_source.sql`, `20261005180100_payroll_journal_mappings_ready.sql`, `20261005201000_payroll_bpjs_employee_tapera_journals.sql`, and the gym/production/inventory mapping deltas.

**Business rules to port verbatim:**
- **COA code structure** (`domain/accountcode.go`):
  - A 7-digit compact code with display form `"1 1 01 001"`: class(1) group(1) sub(2) detail(3).
  - Level is inferred from trailing zeros: `X000000`=1, `XX00000`=2, `XXXX000`=3, otherwise 4.
  - Separators `- _ .` are accepted on input.
- **Journal validation** (`domain/journal.go`):
  - At least 2 lines, with at least one DEBIT and one CREDIT.
  - Every amount >0, and `Round2(debit)==Round2(credit)`.
  - Statuses DRAFT→POSTED (VOID only for AP/AR payments). Entry types MANUAL/OPENING/AUTO.
  - Editable only when `status==DRAFT && !is_recon && entry_type==MANUAL`.
  - POSTED, OPENING, AUTO and recon entries are immutable.
  - Accounts must exist, be undeleted, have `is_postable` and be in the same company.
  - Numbering: `JE-YYYYMM-NNNN` and `OB-YYYY-NNNN`, taken with `FOR UPDATE` on the last number. AP/AR documents use `<PREFIX>-YYYYMMDD-NNNN`.
- **Fiscal rules** (`domain/fiscal.go`):
  - A year has 1–12 periods with no duplicate `period_no`, and periods must fall inside the year.
  - Period N may be OPEN only if every earlier period is CLOSED.
  - A year with an OPEN period requires all earlier years fully CLOSED.
  - Close blockers: the period is already CLOSED, the year is inactive, or the period still has DRAFT journals.
- **Beginning balance** (`service_opening.go`): one OPENING entry per fiscal year, editable while DRAFT. It is suggested from the prior year's closing balance sheet, and period 1 is forced OPEN.
- **Balance math** (`domain/reports.go`):
  - `Balance = (debit-credit)` for normal-DEBIT accounts, else `(credit-debit)`; contra accounts flip the sign.
  - Trial balance puts the net in a single column.
  - P&L types are REVENUE, COGS, EXPENSE, OTHER_INCOME and OTHER_EXPENSE.
- **Income statement:**
  - `GrossProfit = Revenue − COGS`
  - `OperatingIncome = GrossProfit − Expenses`
  - `NetIncome = OperatingIncome + OtherIncome − OtherExpenses`
- **Balance sheet:** as-of balances, with `current_year_earnings` = Σ P&L contributions from the covering fiscal year's start (else Jan 1). Revenue and other income count +, the rest −. `TotalEquity = equity + earnings`.
- **Cash flow** (indirect-lite):
  - Accounts are grouped by `cash_flow_category`.
  - Each non-cash account contributes +balance if REVENUE, OTHER_INCOME, LIABILITY or EQUITY, otherwise −balance.
  - Cash opening and closing = Σ of `is_cash_bank` account balances at the day before `dateFrom` and at `dateTo`.
- **Dashboard** (`GET /dashboard`, `service_reports.go:445`):
  - KPIs: total_assets, total_liabilities, total_equity, net_income, cash_balance, cash_opening, cash_change, trial_balance_diff, posted_entries_ytd, draft_entries.
  - Charts: balance_composition, pnl_breakdown, cash_flow_breakdown, monthly_trend (revenue/expense/net per month) and top 8 expense accounts.
  - It computes no ratios.
- **AR/AP** (`domain/receivables.go`):
  - `outstanding = round2(total) − round2(allocated)`.
  - Status is paid / partial / overdue / unpaid, comparing due_date with today in Asia/Jakarta.
  - Aging buckets: `current`, `1_30`, `31_60`, `61_90`, `90_plus`.
  - AP payment void requires a reason of at least 5 characters and only applies to POSTED payments. A draft PURCHASE_PAYMENT journal is deleted; a posted one gets a reversing AUTO entry.
  - Finance invoice summary statuses: lunas / sebagian / belum.
- **Auto-posting** (`domain/mapping.go`, `posting.go`):
  - Event codes include POS_SALE_*, PURCHASE_GRN, PURCHASE_AP_INVOICE, PURCHASE_PAYMENT, PURCHASE_RETURN, landed cost, PAYROLL_*, PINJAMAN_DISBURSEMENT/REPAYMENT, SALE_AR_INVOICE/RECEIPT, STOCK_*, PRODUCTION_*, GYM_*.
  - Each mapping line takes its amount from an amount bag. A required account missing or a 0 amount makes the result DRAFT or skipped, and the reason is appended to the response message.
  - Posting is idempotent per source document (`already_exists`).
  - Outbox subscribers (`events.go`): `accounting.journal-pos-sale`, `-member-bill-sale`, `-member-deposit`, `-wallet-bonus`, `-grn`, `-purchase-return`, `-stock-opname`, `-stock-adjustment`, `-stock-transfer`, `-landed-cost`, `-production-issue|return|receipt`, `-gym-credits`, `-gym-commission`, `-payroll-run`.
- **Payroll journal** (`service_payroll.go`), on `payroll.TopicRunPaid`, posted per company:
  - PAYROLL_ACCRUAL (TOTAL = gross)
  - PAYROLL_PPH21_WITHHOLDING (TAX)
  - PAYROLL_LOAN_DEDUCTION (PAID)
  - PAYROLL_BPJS_TK_EMPLOYEE, PAYROLL_BPJS_KES_EMPLOYEE, PAYROLL_TAPERA_EMPLOYEE
  - PAYROLL_PAYMENT (PAID = net)
  - PAYROLL_BPJS_TK_EMPLOYER, PAYROLL_BPJS_KES_EMPLOYER, PAYROLL_TAPERA_EMPLOYER
  - Entry date is paid_at in Jakarta. Description: `"Payroll MM/YYYY — <label>"`.
- **Exports and files:** COA XLSX import (`coa_import.go`, sheet "coa"). Faktur Pajak upload goes to `storage.FromEnv()`. There are no PDF or XLSX report exports in Go; budget-vs-actual, changes-in-equity and bank reconciliation pages are "coming soon".

**Dependencies:**
- `Ports{Audit, Purchasing, PosOrders, Materials, Sales}` (`ports.go`).
- Event contracts from possales, storedvalue, procurement, inventory, production, gymcredits, gym and payroll.
- Platform: xlsx, storage, outbox.

========================================================
## 2. HRIS — `backend/internal/modules/hris/`
========================================================
**Size:** 60 files, 15,213 lines including tests. Largest files: `http_people.go` 801, `http_employees.go` 528, `domain/insights.go` 526, `leaves.go` 519, `logbook.go` 518, `contracts.go` 499, `selfservice.go` 477.

**Ownership and ports:** owns people, attendance, leave, overtime, shifts, holidays, contracts, on/offboarding, ESS, announcements, notifications, nav badges, department logbook, monthly HRIS report and HR master data. Payroll, loan and KPI data come through ports. `Ports{Payroll PayrollReads, Salary SalaryPort, Recruitment, Directory, WhatsApp WhatsAppSender, HolidayCalendar (ICS feed), Company CompanyProfile, Files *storage.Store, CoachLeaves}`.

**Auth model:**
- Guards: `withUser` (session), `withMenu(prefixes)`, and `withActor`, which builds `Actor{UserID, Role, FullName, EmployeeID, DepartmentID, IsHR}` from `hris.employees.user_id`.
- `hrRoles` = super_admin, admin, hrd, hiring_manager.
- `employeeRecordManagers` = hris.kepegawaian + settings.users. `employeeDirectoryReaders` adds hris.workforce and others.
- There is no company or branch scoping on employees. Departments, positions and sections carry `brand_id`.
- The logbook is locked to the actor's own department, except for super_admin/admin/hrd.

**Routes** (`routes.go`, `http_files.go`, `import.go`):
- Employees:
  - GET/POST `/api/hris/employees`, GET/PUT/DELETE `/api/hris/employees/{id}`
  - GET/POST `/api/hris/employees/documents`, PATCH/DELETE `/api/hris/employees/documents/{doc_id}`
  - `/api/hris/employees/{id}/{contracts|lifecycle|recruitment-documents|shifts}`
  - GET `/api/hris/employees/import/template`, POST `/api/hris/employees/import`
- Attendance:
  - POST `/api/hris/attendance` (clock in/out, selfie required)
  - GET `/api/hris/attendance`, `/daily-roster`, `/schedule`, `/stats`, `/export` (XLSX "Rekap Absensi"), `/photo/{path...}`
  - GET/PUT/DELETE `/api/hris/attendance/{id}`
- Leaves:
  - GET/POST `/api/hris/leaves`, POST `/leaves/approve`, GET `/leaves/export` (CSV)
  - GET/PUT/DELETE `/leaves/{id}`
  - POST `/leaves/attachment`, GET `/leaves/attachment/{path...}`
  - GET/PUT `/api/hris/leave-balances/{employee_id}`
- Shifts and overtime: GET/POST `/api/hris/shifts`, PATCH/DELETE `/shifts/{id}`; GET/POST `/api/hris/overtime`, POST `/overtime/decide`.
- Holidays: GET/POST `/api/hris/holidays`, GET/POST `/holidays/import`, PATCH/DELETE `/holidays/{id}`.
- Contracts:
  - GET `/api/hris/contracts`, GET `/contracts/expiring`, PATCH/DELETE `/contracts/{id}`
  - GET `/contracts/{id}/document` (PDF)
  - GET/POST/DELETE `/contracts/{id}/signed-document`
- Records: GET/POST `/api/hris/employment-history`, GET `/api/hris/departments`, GET/POST/PUT `/api/hris/onboarding/{employee_id}`, GET/POST/PUT `/api/hris/offboarding/{employee_id}`.
- ESS: GET `/api/hris/me`, `/me/beranda`, `/me/team`.
- Announcements:
  - GET/POST `/api/hris/announcements`, GET `/announcements/feed`, GET `/announcements/unread-count`
  - GET/PATCH/DELETE `/announcements/{id}`, POST `/announcements/{id}/read`
  - POST `/announcements/cover`, GET `/announcements/cover/{path...}`
- Notifications and badges: GET/POST `/api/hris/notifications`, GET/POST `/api/hris/nav-badges`.
- Logbook: GET/POST/PATCH/DELETE `/api/hris/logbook`. GET takes `?resource=me|departments|templates|summary|entries`.
- Monthly report: GET `/api/hris/reports` (guard hris.insights).
- Master data: GET/POST + PUT/DELETE `{id}` on `/api/master/departments`, `/api/master/employment-statuses`, `/api/master/positions`. Import and template routes exist for departments and positions.

**Tables:**
- Baseline (`schemas/hris/`):
  - `hris.employees(id, user_id, full_name, nip, ktp, npwp, email, phone, birth_date, gender, marital_status, address…, join_date, end_date, employment_status, is_active, department_id, section_id, job_title_id, reporting_to (self → manager), bank_name, bank_account, bpjs_tk, bpjs_kesehatan, emergency_*, photo_url)`. Later adds `is_access_app`.
  - `hris.attendance(employee_id, date, clock_in, clock_out, clock_in/out_location jsonb, work_hours, break_minutes, status attendance_status, is_late, late_minutes, is_overtime, overtime_hours, validated_by/at, validation_notes)`. Later adds `clock_in/out_photo_url`, `shift_id`, `scheduled_start/end`.
  - `hris.leaves(employee_id, leave_type, start_date, end_date, total_days, reason, attachment_url, status leave_status, approved_by, approved_at, rejection_reason)`.
  - `hris.leave_balances(employee_id, year, annual_leave_total/used/remaining, sick/unpaid/maternity/paternity/emergency/pilgrimage/menstrual_leave_used)`.
  - `hris.departments(name, code, brand_id, parent_department_id, cost_center)`, `hris.positions(brand_id, title, department, level)`. Positions later get `salary_min` and `salary_max`.
  - `hris.employment_statuses`, `hris.employment_history(change_type, prev_*/new_* dept/section/job/status/salary)`, `hris.employee_documents`.
  - `hris.onboarding_checklists(task_name, category, priority, due_date, due_days_after_join, completed…)`.
  - `hris.offboarding_checklists(resignation_type, status offboarding_status, exit interview, final payroll, asset_return_status jsonb, clearance_hrd/it/finance/manager)`.
  - `hris.hris_logbook_templates(department_id, name, frequency daily|weekly|monthly|custom)`, `…_template_items(title, weight, is_required)`.
  - `hris.hris_logbook_entries(template_id, department_id, entry_date, title, status, completion_percentage, kpi_score, notes, submitted_by/at, reviewed_by/at, review_notes)`, `…_entry_items(is_checked, weight)`.
  - `hris.sections`, `hris.staff`, `hris.staff_schedules`, `hris.staff_sections`, `hris.employee_schedules` (legacy), `hris.benefits`, `hris.employee_benefits`.
- Deltas:
  - `hris.shifts(name, start_time, end_time, break_minutes, late_tolerance_minutes, is_overnight)` and `hris.employee_shifts(employee_id, day_of_week 1..7, shift_id NULL=off, effective_from/to)`, from `20260717080000_absensi_v2_shifts.sql`.
  - `hris.overtime_requests(employee_id, date, start_time, end_time, hours, source employee|company, status, reason, requested_by, decided_by/at, rejection_reason)`, from `20260717170000`.
  - `hris.employment_contracts(contract_number, contract_type pkwt|pkwtt, status draft|active|ended|terminated|converted, start/end/probation_end_date, parent_contract_id, sequence, position_title, department_name, work_location, base_salary, allowances jsonb, signed_at, signed_document_url, kemnaker_registered_at, compensation_amount, compensation_paid_at, terminated_reason)`, from `20260716120000`. CHECKs: pkwt_needs_end_date, pkwt_no_probation, end_after_start.
  - `hris.announcements(title, body_html, cover_image_url, video_provider/id, tags[], status, is_pinned, target_scope, publish_at, expires_at)` + `announcement_departments` + `announcement_reads`, from `20260718110000`.
  - `hris.public_holidays(holiday_date, name, type nasional|cuti_bersama|perusahaan, deducts_leave, status draft|aktif, source manual|impor, source_ref)`, from `20260729100000`.
  - `hris.ess_module_reads`; `configuration.wa_notif_log` (WhatsApp dedupe).
- Enums:
  - `attendance_status` = present, late, absent, half-day, remote
  - `leave_status` = pending, approved, rejected, cancelled
  - `leave_type` = annual, sick, maternity, paternity, unpaid, emergency, pilgrimage, menstrual
  - `offboarding_status` = submitted, notice_period, exit_interview, completed
  - `resignation_type` = voluntary, termination, layoff, end_of_contract
  - `onboarding_category` = admin, it, hr, manager, general
  - `shift_type` = morning, afternoon, night, flexible, custom

**Business rules:**
- **Leave** (`leaves.go`, `domain/workforce.go`):
  - Weekends and `nasional` holidays don't consume leave; `cuti_bersama` does. A range with zero working days is refused.
  - Created as `pending`.
  - HR or the direct manager (`reporting_to`) approves or rejects pending requests. Approving annual leave consumes quota in the same transaction.
  - Cancel: the owner while pending; HR also once approved, with the quota refunded.
  - Leave balance row auto-created on first read with `ProratedAnnualQuota` = 12, or `floor(12 × remainingMonths(incl. join month)/12)` for an employee who joined that year. HRD can set it; new rows start at 12.
  - Manager WhatsApp: outbox topic `TopicLeaveRequested` → `hris.leave-request-whatsapp`, deduped via `configuration.wa_notif_log`. The decision response also returns a wa.me link.
- **Overtime** (`domain.CanDecideOvertime`):
  - Source `employee`: decided by HR or the manager, never by self.
  - Source `company` (an HR assignment): only the assigned employee confirms it.
  - Cancel: the requester, or HR for company assignments.
  - Hours computed from HH:MM with overnight wrap, rounded to 2 decimals. One active (pending/approved) request per employee per date.
- **Attendance:** clock-in snapshots the scheduled shift and computes lateness using `late_tolerance_minutes`. A selfie is mandatory both ways. Non-HR users only see and clock themselves.
- **Contracts** (`domain/contracts.go`, UU 13/2003 + PP 35/2021):
  - A PKWT needs an end date, has no probation and is capped at 60 months across the chain.
  - Compensation = months/12 × monthly wage (0 under 1 month).
  - A PKWTT may have up to 3 months of probation.
  - Number format `0001/PKWT/VII/2026` (Roman month).
  - Actions: activate, end, terminate, convert, renew, edit. Renewal starts the day after the previous contract ends.
  - `expiring` (days 1..90, default 30) lists contracts ending soon including overdue, probations ending, and active employees without a contract. There is no cron for it; it is only a read endpoint, used by `frontend/src/features/users/api.ts`.
- **Logbook** (`logbook.go`):
  - A template's items are copied into a dated entry. Entry statuses: draft → submitted → reviewed | rejected; only super_admin and hrd review.
  - DB function `public.update_hris_logbook_entry_score`: `completion_percentage = checked/total×100`, `kpi_score = checked_weight/total_weight×100`.
  - The summary groups entries per department with avg completion and avg KPI.
- **Monthly HRIS report** (`domain/insights.go` `BuildHrisReport`):
  - Headcount: total_active, new_hires, turnover_count, turnover_rate, by_status, by_department, monthly_trend.
  - Attendance: present/absent/late counts and rates, avg_work_hours, daily_trend.
  - Leaves: approved/pending counts, total_days, by_type.
- **Files and exports:** XLSX attendance recap, CSV leave export (formula-injection safe), PDF contract document (`contract_document.go`, includes rupiah "terbilang"), employee and master-data XLSX/CSV import (`platform/importer`), and storage for selfies, leave attachments, announcement covers and signed contracts.

**Dependencies:** platform `pdfgen`, `xlsx`, `extract`, `importer`, `imageproc`, `ratelimit`, `outbox`, `storage`. Ports to payroll (salary, payslips, KPI), recruitment (source candidate), directory (`configuration.users`), WhatsApp and the gym coach-leave rules.

========================================================
## 3. PAYROLL — `backend/internal/modules/payroll/`
========================================================
This module also contains KPI, performance reviews, 360 feedback and department tasks.

**Size:** 59 files, 14,470 lines including tests. The domain is 3,567 lines.

**Routes** (`module.go` routes + `fileRoutes` + `depttasks_http.go` taskRoutes):
- Payroll runs:
  - GET/POST `/api/hris/payroll`, GET/PUT/DELETE `/api/hris/payroll/{id}`, POST `/api/hris/payroll/{id}/calculate`
  - Payslips: GET `/api/hris/payslips`, GET `/api/hris/payslips/{id}/pdf`, POST `/api/hris/payslips/notify` (email PDF)
  - Settings: GET/PUT `/api/hris/payroll-settings`
  - Salary: GET/POST `/api/hris/employee-salary`, GET/PUT/DELETE `/api/hris/employee-salary/{id}`
  - Loans: GET/POST `/api/hris/loans`, POST `/api/hris/loans/{id}/approve`
- KPI:
  - GET/POST/DELETE `/api/hris/kpi/targets`
  - GET/PATCH `/api/hris/kpi/scorecards` (PATCH action finalize|reopen)
  - POST `/api/hris/kpi/rubric`, GET `/api/hris/kpi/recommendation`, POST `/api/hris/kpi/snapshot`
  - GET/PUT `/api/hris/kpi-config`
- Department tasks:
  - GET/POST `/api/hris/dept-tasks`, PATCH `/api/hris/dept-tasks/{id}`, PATCH `/api/hris/dept-tasks/occurrences/{id}`
  - IAM-guarded views: GET `/meta`, `/items`, `/monitor`, `/kpi`, `/kpi/contributions`, `/occurrences/{id}`; POST/GET/DELETE `/occurrences/{id}/attachments[/{attId}]`
- Performance reviews:
  - GET/POST `/api/hris/performance/cycles`, GET `/api/hris/performance/realtime`
  - GET `/api/hris/performance/reviews`, GET/PATCH `/api/hris/performance/reviews/{id}`
- 360 feedback (no UI yet; KPI admin menus required):
  - `/api/hris/feedback-approvals` (GET, POST, POST `/approve`, POST `/reject`)
  - `/api/hris/feedback-assignments[/{id}]`, `/api/hris/feedback-categories`, `/api/hris/feedback-cycles[/{id}]`
  - `/api/hris/feedback-responses[/{id}]` (POST approves, PATCH rejects)
  - `/api/hris/feedback-summaries`

**Auth:**
- `requireActor` gives any session an actor `{UserID, Role, EmployeeID, IsHR}`. Menu guards: `hris.compensation` (runs, salary), `hris.performance`, `hris`; tasks use `iam.Can("read"/"update", tasks|tasks.team|tasks.kpi)`.
- Role lists (`domain/access.go`, `domain/kpi.go`):

| Role list | Roles |
|---|---|
| HRRoles | super_admin, admin, hrd, hiring_manager |
| LoanManageRoles | super_admin, admin, hrd, finance_staff |
| PayslipFullRoles | super_admin, hrd, finance_staff |
| PayrollSettingsWriteRoles | super_admin, hrd |
| KPIManageRoles | super_admin, admin, hrd |

- Payslip scope: non-HR users are forced to their own slips, and only for paid runs.
- No tenant scope on runs. A paid run is split per company through each employee's `configuration.users.company_id`, else the branch's company (`app/adapters_payroll.go:129`, `domain/company_split.go`).

**Tables:**
- Payroll (baseline + deltas):
  - `hris.payroll_runs(run_name, period_month, period_year, status, processed_by/at, approved_by/at, paid_at, total_employees, total_gross, total_deductions, total_net, total_bjtk_employee, total_bjtk_employer, total_pph21, notes)`.
  - `hris.payroll_details` columns:
    - Earnings: base_salary, fixed/variable/transport/meal/housing allowance, overtime_pay, thr, bonus, other_earning.
    - Employee deductions: bpjs_tk_jht/jp, bpjs_kes, tapera, pph21, loan, unpaid_leave, other.
    - Totals: gross_salary, total_deductions, net_salary.
    - Attendance: working_days, present_days, late_days, unpaid_leave_days, overtime_hours.
    - Employer contributions: bpjs_tk_jht/jp/jkk/jkm_employer, bpjs_kes_employer, tapera_employer.
    - Tax: taxable_income, ptkp_amount, pph21_annual, pph21_monthly.
    - Delivery: status, payslip_sent…
    - Added by deltas: late_deduction, prorate_factor, loan_details.
  - `hris.payroll_settings(company_name, npwp, currency, bpjs_* rates, bpjs_kes_max_upah, tapera_*, pph21_bracket_1..5, ptkp_tk_0..k_3, thr_eligible_months, thr_prorate, payroll_day, overtime_multiplier)`. Deltas add `overtime_multiplier_holiday` (2.0), `late_deduction_mode` (off), `late_deduction_amount`, `overtime_hourly_divisor` (173), `loan_max_installment_percent` (30) and `loan_max_active_per_employee` (1).
  - `hris.payroll_tax_config(tax_year, ptkp_*, bracket_1..4_limit/rate, bracket_5_rate, jabatan_expense_percentage, jabatan_expense_max, pension_deduction_percentage, is_active)`.
  - `hris.employee_salary(employee_id, base_salary, fixed/variable/transport/meal/housing_allowance, loan_deduction, other_deduction, ptkp_status, is_taxable, bpjs_tk_enrolled, bpjs_kes_enrolled, tapera_enrolled, effective_date, end_date, is_active)`.
  - `hris.loans(employee_id, loan_type, principal_amount, interest_rate, tenor_months, requested_date, approved_*, rejected_*, disbursed_at/amount, monthly_installment, first_installment_month/year, remaining_balance, paid_amount, status pending|approved|rejected, is_active, purpose)`.
  - `hris.payroll_additions(employee_id, period_month, period_year, amount, label, source_type, source_id)` (gym coach payouts → irregular income).
- KPI (`20260719090000_kpi_scorecard_fase_a.sql`, `20260831140000`):
  - `performance.kpi_indicators(code, name, unit, direction higher_better|lower_better|boolean, source_kind auto|manual, source_ref, default_target)`.
  - `performance.kpi_role_indicators(role_code, indicator_id, weight)`, `performance.kpi_department_indicators(department_id, indicator_id, weight)`.
  - `performance.kpi_targets(indicator_id, period_year, period_month, role_code, department_id, employee_id, target)`.
  - `performance.kpi_snapshots(employee_id, indicator_id, period, actual, target, attainment, sample_size, source_detail jsonb)`.
  - `performance.kpi_scorecards(employee_id, period, role_code, score, raw_score, used_weight, breakdown jsonb, status draft|final, reviewed_by/at)`.
- Performance reviews:
  - `performance.review_cycles(name, period_year, period_quarter, start/end_date, status)`.
  - `performance.performance_reviews(cycle_id, employee_id, dept, position, period_label, totals work/behavioral/project/grand, category, status draft|final, reviewer/manager, sign dates, self_assessment, reviewer_notes, manager_notes)`.
  - `performance.behavioral_review_items(value_name, competency, score 1-5 descriptions, weight, score, weighted_score)`, `performance.behavioral_standards`, `performance.performance_categories`, `performance.score_scales`.
  - Legacy and unused: `performance.kpi_templates`/`_items`/`_behavioral`, `hris.employee_kpis`, `hris.project_assignments`, `hris.development_plans`, `performance.behavioral_assessments`.
- 360 feedback: `performance.feedback_cycles(kpi_weight, feedback_weight, is_anonymous, allow_self_assessment, require_manager_review, status)`, `feedback_assignments(cycle_id, employee_id, reviewer_id, relationship_type, status pending|in_progress|submitted→approved|rejected, due_date, reminder_count, last_reminder_at, manager_comments)`, `feedback_categories`, `feedback_criteria`, `feedback_relationship_types`, `feedback_responses(rating, comments, sentiment_score/label, ai_keywords[])`, `feedback_summaries(leadership/communication/collaboration/accountability/problem_solving/overall_360/kpi/final_score, final_grade, strengths[], weaknesses[], burnout_risk, promotion_potential, is_locked)`.
- Department tasks (`20260830120000`, `20260831100000`, `20261027120000_dept_tasks_progress_kpi.sql`):
  - `hris.department_tasks(department_id, assignee_employee_id, title, description, recurrence once|daily|weekly|monthly, weekly_day 1..7, monthly_day 1..28, due_date, is_active, priority low|normal|high|urgent, kpi_indicator_id, kpi_contribution numeric(15,4), timezone 'Asia/Jakarta', review_required, created_by(_user_id/_name))`.
  - `hris.department_task_occurrences(task_id, occurrence_date UNIQUE per task, status pending|in_progress|done|approved|rejected, progress_percent 0..100, started_at, done_at/by, review_notes, reviewed_by/at)`.
  - `hris.department_task_subtasks(task_id, title, weight, sort_order)`, `hris.department_task_occurrence_items(occurrence_id, subtask_id, is_checked)`.
  - `hris.department_task_updates(occurrence_id, kind progress|comment|status|review|attachment|created, note, progress_percent, author_*, detail jsonb)`.
  - `hris.department_task_attachments(occurrence_id, name, mime, size_bytes, storage_path)`.
  - `hris.department_task_kpi_contributions(occurrence_id UNIQUE, indicator_id, department_id, employee_id, period_year, period_month, amount)`.

**Business rules:**
- **Run state machine:** draft → processing → completed → paid, forward one step only. Only draft runs can be calculated; a paid run cannot be deleted.
- **Marking paid:** settles loan installments in the same transaction (decrements `remaining_balance`, `is_active=false` at 0) and publishes `payroll.run.paid` with per-company totals. Accounting then posts the journals.
- **Calculation** (`domain/calculator.go` `Calculate`, config `domain/config.go`):
  - Proration: `factor` clamped to 0..1 from contract coverage / scheduled days (`domain/inputs.go` BuildInput). `base = round(base×factor)` and every allowance is also × factor.
  - Overtime: `hourly = fullBase / OvertimeHourlyDivisor (173)`; `overtimePay = hours×hourly×1.5 + holidayHours×hourly×2.0`. Approved overtime requests are capped by realized clock-out and split by public holidays (`SplitOvertime`).
  - THR (only when IncludeThr and the employee has a contract or is permanent): full base if joined before the period year, else `round(base × clamp(floor(days(Dec31−join)/30),1,12)/12)`.
  - `gross = base + fixed + variable + transport + meal + housing + overtime + thr + bonus` (bonus = payroll_additions).
  - BPJS is computed on `base+fixed`, with separate caps: TK cap 10,414,000 and Kes cap 12,000,000.

| Contribution | Employee | Employer |
|---|---|---|
| JHT | 2% | 3.7% |
| JP | 1% (only if monthly ≥ 5,000,000) | 2% |
| Kes | 1% | 4% |
| Tapera (on TK base) | 2.5% | 0.5% |
| JKK | — | 0.24% |
| JKM | — | 0.3% |

  - Deductions: unpaid leave = `base/workingDays×days`. Late deduction by mode `per_minute` (minutes×amount) or `flat` (days×amount); default `off`.
  - PPh21:
    1. Regular annual = `(gross − irregular)×12`.
    2. Taxable = `max(0, annual − employeeBPJS×12 − min(annual×5%, 6,000,000))`.
    3. PKP = `floor((taxable − PTKP)/1000)×1000`.
    4. Progressive brackets as widths: 60M@5%, 190M@15%, 250M@25%, 4.5B@30%, rest@35%.
    5. Monthly = round(annual/12).
    6. Irregular income is taxed as Δ(annual tax with − without), withheld in that month (see `docs/PAYROLL_PPH21.md`).
  - PTKP: TK/0 54M, TK/1 58.5M, TK/2 63M, TK/3 67.5M, K/0 58.5M, K/1 63M, K/2 67.5M, K/3 72M. Unknown statuses fall back to TK/0. Values are overridable by `payroll_tax_config` (cumulative limits converted to widths) or `payroll_settings`.
  - `totalDeductions = JHT + JP + Kes + Tapera + PPh21 + unpaid + late + loan`; `net = gross − totalDeductions`.
- **Loans** (`domain/loans.go`, `loans.go`):
  - Flat interest: `monthly = round(principal×(1+rate%×tenor)/tenor)`. Employee self-requests are interest-free.
  - Limits: active loans < `loan_max_active_per_employee` (1), and installment ≤ base × `loan_max_installment_percent` (30%). An active salary is required.
  - Statuses pending → approved | rejected. The first installment is the month after approval. Allocation goes oldest loan first.
- **Payslip:** PDF (`payslip_pdf.go`, pdfgen). The email goes through the mailer port (`PAYROLL_FROM_EMAIL`/`FROM_EMAIL`), and a copy is filed in the Data Room under `Payroll/Slip Gaji/<tahun>/<periode>` when dataroom is enabled.
- **Run XLSX:** built client-side in `frontend/src/lib/payroll/ui-run-workbook.ts` (exceljs) with sheets Ringkasan and Rincian. This is a draft, not a bank format.
- **KPI scoring** (`domain/kpi.go`):
  - Attainment:
    - higher_better: actual/target, clamped 0..1.2.
    - lower_better: target/actual (target 0 → 1 if actual ≤ 0, else 0); actual ≤ 0 → 1.2.
    - boolean: actual ≥ 1 → 1, else 0.
    - nil excludes the indicator.
  - `ComposeScore`: Σ weight×attainment scaled to 0–100. Excluded weights are redistributed proportionally; the score is capped at 100 and `raw_score` is kept.
  - Weights: a configured department weighting wins over the role mapping.
  - Monthly snapshot (`POST /kpi/snapshot`, idempotent): upserts snapshots and draft scorecards; final scorecards are untouched.
  - Auto collectors per employee: att_ontime, pos_cash_variance, logbook_compliance, po_fulfillment, team_ontime, pos_sales_shift, leave_request_discipline, att_late_ratio, task_completion, coach_complaint_rate.
  - Org-wide collectors: leave_sla, vendor_pay_ontime, vendor_pay_sla, payroll_paid_ontime, payroll_ready_h2.
  - Manual supervisor rubric code `supervisor_rubric`, max 5. Scorecard actions: finalize and reopen.
- **Quarterly review** (`domain/performance.go`):
  - Opening a cycle creates a draft review per employee plus behavioral items copied from `behavioral_standards`.
  - Score = work result 60% (KPI average) + behavior 30% (weighted 1–5 → 0–100) + project 10% (0–100). Empty components are redistributed.
  - Actions: `self_assessment` (employee only), `rate_item` and `reviewer_notes` (department head or HRD), `sign`, `finalize` (HRD only → status `final`).
  - 360 feedback: `FinalScore = kpi×kw% + overall360×fw%` (default 70/30). Grades: A ≥ 90, B ≥ 80, C ≥ 70, D ≥ 60, else E.
- **Department tasks** (`domain/depttasks.go`, `depttask_views.go`, `depttasks.go`):
  - Validation: title at least 3 characters; weekly needs a day 1..7; monthly a day 1..28; once needs a due_date. Up to 50 subtasks; weights split 100 evenly with the rounding remainder on the last.
  - Occurrence dates are generated per day/week/month, never before the task's created date.
  - Occurrence status: pending → in_progress → done → approved | rejected. Rejected goes back to work.
  - Actions: start, progress (0–99% with a note), check_subtask, comment, done, approve, reject, reopen.
  - If `review_required=false` (a personal to-do), done goes straight to approved.
  - Who acts:
    - Creating: HR anywhere, a department head (has active direct reports) in their own department, anyone else a personal task.
    - Reviewing: HR or the department head.
    - Reopening an approved occurrence: HR or the head.
  - Effective progress: done or approved 100, pending 0, otherwise the weighted subtask share or manual %, capped at 99.
  - On approve, insert into `department_task_kpi_contributions(amount = task.kpi_contribution)` in the month of the occurrence; reject deletes it.
  - The task_completion KPI = approved / due up to today.
  - Monitor: per department and per assignee, overdue (oldest first) and waiting review.
  - Views: today, this ISO week (Mon–Sun) and calendar month, in the branch timezone.

**Scheduled jobs** (`app/module_payroll.go`):
- `dept-tasks.recurrence`, every 1h: `MaterializeOccurrences` creates occurrences for yesterday to tomorrow across all zones, from Jakarta to Jayapura.
- `dept-tasks.reminders`, every 15m: `SendReminders`.
  - From 07:00 local: a morning digest titled "Tugas hari ini: N (k terlambat)", listing up to 5 lines with overdue first.
  - From 16:00 local: an overdue notice titled "N tugas terlambat".
  - Recipients have open occurrences (pending, in_progress, rejected).
  - Writes a `public.notifications` inbox row linking to `/dashboard/tasks`, then WhatsApp and email best effort through `TaskChannels` (the gym channels). Deduped per (kind, employee, day) via `platform.reminder_sends`.

**Dependencies:**
- Ports: Employees and Workforce (from hris, through `app/adapters_payroll.go`), KPISources (`app/adapters_payroll_kpi.go`), PayslipMailer, DocumentFiler (dataroom), TaskChannels, TaskStore.
- Outbox publish `payroll.run.paid`. Platform: `jobs`, `localtime`, `mail`, `pdfgen`, `ratelimit`.

========================================================
## 4. RECRUITMENT — `backend/internal/modules/recruitment/`
========================================================
**Size:** 43 files, 11,151 lines including tests.

**Auth:**
- Staff routes use `h.staff(iam.HrisRecruitment)`.
- Public candidate portals authenticate with a 48–128 hex token in the URL, and are rate limited.
- `CanSeePortalToken` = super_admin, admin, hrd. `notifyRoles` controls who may send candidate notifications.

**Routes:**
- Candidates:
  - GET/POST `/api/candidates`, GET/PUT/DELETE `/api/candidates/{id}`
  - GET/POST `/{id}/notes`, GET/POST `/{id}/activities`, POST `/{id}/stage`, GET/PUT `/{id}/screening`
  - GET `/{id}/psikotes`, POST `/{id}/psikotes/sessions`, PUT `/{id}/psikotes/summary`
  - GET `/{id}/interview`, POST `/{id}/interview/sessions`
  - GET/POST `/{id}/offers`
  - POST/DELETE `/{id}/cv-upload`, GET `/{id}/report` (PDF), POST `/api/candidates/cv-extract`, GET/POST `/{id}/ai-analysis`
- Psikotes admin:
  - GET `/api/psikotes/instruments`, PUT `/instruments/{id}`, GET/POST `/instruments/{id}/questions`, PUT/DELETE `/api/psikotes/questions/{id}`
  - GET `/api/psikotes/session-tests/{id}/answers`, PUT `/session-tests/{id}/review`, POST `/session-tests/{id}/ai-insight`
  - GET `/api/psikotes/sessions/{id}/proctor-events`
- Interview admin: GET `/api/interview/sessions/{id}/proctor-events`, GET `/api/interview/sessions/{id}/recordings`, GET `/api/{interview|psikotes}/files/{path...}`.
- Public psikotes portal (`/api/psikotes/session/{token}`): GET; POST `/start`, `/finish`; POST `/tests/{testId}/start`; PUT `/tests/{testId}/answers`; POST `/tests/{testId}/finish`; POST `/tests/{testId}/upload`; POST `/proctor-event`; GET/POST `/chat`; POST `/live-frame`; GET/POST `/webrtc`.
- Public interview portal (`/api/interview/session/{token}`): GET; POST `/start`, `/answer`, `/recording-chunk`, `/proctor-event`; GET/POST `/chat`; POST `/live-frame`; GET/POST `/webrtc`.
- Offers: public GET `/api/offer/session/{token}`, POST `/api/offer/session/{token}/respond`; staff PUT `/api/offers/{id}/response` (manual).
- Live monitoring: GET `/api/recruitment/live-monitoring`; `/{type}/{id}/chat` (GET/POST), `/{type}/{id}/frame` (GET), `/{type}/{id}/webrtc` (GET/POST).
- Jobs and promotion:
  - GET/POST `/api/hris/job-openings`, PATCH/DELETE `/api/hris/job-openings/{id}`, public GET `/api/job-openings/public`
  - POST `/api/hris/promote` (candidate → employee)
  - GET `/api/positions` (guard hris), POST `/api/positions` (guard hris.master)
  - Public `/api/portal/options`, POST `/api/portal/submit`
- POST `/api/notifications/send` (candidate WhatsApp or email).

**Tables:**
- Baseline:
  - `recruitment.candidates(full_name, email, phone, domicile, source, position_id, brand_id, cv_url, photo_url, status, notes, promoted_to_employee_id, promotion_date, job_opening_id, last_experience, last_education, availability, expected_salary)`
  - `recruitment.interviews(candidate_id, interviewer_id, interview_date, type, scorecard jsonb, recommendation, notes, mode, meeting_link)`
  - `recruitment.job_openings(position_id, brand_id, department_id, title, slug, department, location, employment_type, work_mode, headcount, description, requirements, benefits, status draft|published|closed, published_at, closing_date)`
- Deltas (2026-07-14 .. 2026-10-05):
  - `candidate_ai_analysis(cv_text, extracted, summary, match_score, match_reason, job_context, model)`, `candidate_notes`, `candidate_activities(activity_type, description)`.
  - `candidate_screenings(contacted, interested, availability_note, confirmed_salary, willing_shift, willing_placement, notes, recommendation lolos|hold|tidak_lolos)`.
  - `psikotes_instruments(code, name, kind mcq|forced_choice|drawing, config)`, `psikotes_questions(body, options, answer_key)`.
  - `psikotes_sessions(token, status draft|sent|in_progress|completed|expired, webcam_consent, invited/expires/started/completed_at)`.
  - `psikotes_session_tests(status pending|in_progress|selesai|perlu_review|reviewed, answers, score, score_detail, attachment_path, review_notes)`.
  - `psikotes_proctor_events(tab_blur|fullscreen_exit|paste|disconnect|webcam_snapshot)`, `candidate_psikotes_summary(recommendation)`.
  - `interview_ai_sessions(token, status sent|in_progress|completed|expired, config, ai_summary, summary_model)`, `interview_ai_turns(turn_no, topic, question, question/answer_audio_path, answer_transcript, answer_mode voice|text)`, `interview_ai_proctor_events`.
  - `candidate_offers(version, token, status sent|negotiating|accepted|declined|expired, position_title, base_salary, benefits, start_date, response_note, responded_at, response_ip, response_source portal|manual, expires_at)`.
  - `live_monitor_frames`, `live_chat_messages(sender candidate|hr)`, `live_signals(offer_sdp, answer_sdp)`.
  - `configuration.app_settings(key, value)` holds the AI keys.

**Business rules:**
- **Pipeline statuses:** applied, screening, psikotes, interview, offer, hired, talent_pool, rejected, archived. New candidates start in applied or screening. Stage moves are free-form and logged as activity "Tahap diubah: X → Y".
- **Promotion:** only hired or talent_pool candidates. Employment status contract → auto-draft PKWT 12 months; probation → PKWTT with 3-month probation; permanent → PKWTT.
- **Psikotes scoring:** MCQ 0–100, rounded half up. PAPI Kostick: 20 scales, count chosen scales, dominant = top ties. Answer keys never leave the server.
- **Offer response:** action accept/negotiate/decline maps to status accepted/negotiating/declined.
- **WhatsApp templates:** undangan_screening, lolos_psikotes, undangan_psikotes, undangan_interview, lolos_interview, lolos_offer, offer_terkirim, penolakan.
- **AI:** DeepSeek and OpenAI chat completions and Whisper over net/http, with keys in `configuration.app_settings`. Used for CV extraction/OCR, CV-job match scoring, interview question generation and summary, and the psikotes drawing insight.
- **Files:** candidate pipeline report PDF (`files_report.go`, from the Offer stage), CV upload and extract, interview recordings, proctor snapshots.

**Dependencies:** platform `whatsapp`, `ocr`, `extract`, `pdfgen`, `ratelimit`, `storage`. Ports to hris (positions, employee creation on promote) and the mailer.

========================================================
## 5. Reimbursement / claim / expense / petty cash anywhere
========================================================
Not present. I grepped `reimburs|klaim|petty.?cash|kas kecil|expense_claim|expense_report` across `backend/internal`, the migrations and `frontend/src`. Matches are only:
- COA test fixtures naming "Kas Kecil"/"Petty Cash" accounts (`accounting/domain/coaimport_test.go`).
- The word "klaim" in an unrelated comment (`frontend/src/lib/payroll/period.ts`).
- CRM reward "claims".

There is no reimbursement table, route, page or approval flow. The closest building blocks:
- `hris.payroll_additions`: one-off amounts added to payroll and taxed as irregular income.
- The loan approval flow.
- Cash-out journals (`POST /api/accounting/cash-bank/cash-out`).
- Journal mappings, which could take a new event code.

========================================================
## 6. Requested features, per item
========================================================
**(a) Reimbursement system automation: NOT PRESENT.** See section 5.

**(b) Financial system analysis (dashboards, ratios, insights): PARTIAL.**
- `backend/internal/modules/accounting/service_reports.go` `Dashboard()` (`GET /api/accounting/dashboard`), with the UI in `frontend/src/features/accounting/reports/`.
  - KPIs: assets, liabilities, equity, net income, cash opening/closing/change, trial-balance diff, posted/draft counts. Plus P&L/BS/CF breakdowns, monthly revenue/expense/net trend and top 8 expense accounts.
  - Reports: trial balance, balance sheet, income statement, cash flow, general ledger, subsidiary ledger, AR/AP aging.
  - No financial ratios (liquidity, margin, DSO) and no AI insight are computed. Budget-vs-actual and changes-in-equity are "coming soon" pages.
- Elsewhere:
  - `backend/internal/modules/insights/executive.go`: executive dashboard with 14-day sales trend, stock value, purchasing, payroll and contracts; cached 60s.
  - `backend/internal/modules/investor/performance.go`: monthly revenue by stream plus the posted income statement.
  - `investor/digest.go`: daily email digest at 07:00 WIB (job `investor.daily-digest`).
  - `insights/assistant.go`: the "Do" LLM assistant with tools.

**(c) Employee performance automation (KPI, appraisal, scoring): PRESENT, extensive.**
- Files: `backend/internal/modules/payroll/{kpi.go,kpi_snapshot.go,domain/kpi.go,domain/kpi_collect.go,performance.go,domain/performance.go,feedback*.go}`.
- How it works:
  - A monthly KPI snapshot pulls auto-collected actuals (attendance, POS, logbook, tasks, payroll timeliness and others) against targets, computes attainment capped at 120%, and composes a weighted 0–100 scorecard (draft → final).
  - A supervisor rubric covers manual indicators.
  - Quarterly reviews combine KPI 60 / behavior 30 / project 10, with a sign-off and HRD finalize.
  - The 360-feedback backend exists with no UI.
- The snapshot is triggered manually (`POST /api/hris/kpi/snapshot`); there is no cron.
- UI: `/dashboard/hris/kpi-scorecard`, `/kpi-config`, `/performance`, `/dashboard/me/kpi`, `/dashboard/tasks/kpi`.

**(d) Daily work recap automation: PARTIAL.**
- Department logbook (`backend/internal/modules/hris/logbook.go`, `domain/insights.go`, UI `/dashboard/hris/logbook`):
  - A daily, weekly, monthly or custom checklist template per department becomes a dated entry, moving draft → submitted → reviewed/rejected.
  - A DB function scores completion and weighted KPI. The summary aggregates per department.
  - Feeds the `logbook_compliance` KPI.
- Daily task occurrences (department tasks): the 07:00 digest lists today's open tasks.
- Attendance daily roster: `GET /api/hris/attendance/daily-roster`.
- There is no free-text "daily report" or auto-generated daily summary of work done.

**(e) Monthly work recap automation: PARTIAL.**
- `GET /api/hris/reports` (`BuildHrisReport`): monthly headcount, attendance and leave recap, UI `/dashboard/hris/reports` with CSV.
- `GET /api/hris/attendance/export`: XLSX "Rekap Absensi".
- Monthly views of tasks (`view=month`) and KPI per indicator (`/api/hris/dept-tasks/kpi?month=`), plus the monthly KPI scorecard.
- There is no scheduled monthly recap email or report.

**(f) Deadline reminder automation: PRESENT for department tasks only.**
- Job `dept-tasks.reminders`, every 15 min (`backend/internal/modules/payroll/depttasks_jobs.go`, `domain/depttask_views.go` DigestHour=7, OverdueHour=16).
  - Per assignee in their branch timezone: a morning digest of today's plus overdue open tasks, and an afternoon overdue notice.
  - Delivered to the inbox (`public.notifications`), then WhatsApp and email. Deduped via `platform.reminder_sends`.
- Not automated:
  - Contract and probation expiry is only a read endpoint (`GET /api/hris/contracts/expiring`).
  - 360 assignments have `reminder_count`/`last_reminder_at` columns with no code.
  - Nothing for AP/AR due dates beyond aging lists.

**(g) "Things to improve" automation (retrospective, improvement notes, feedback): MINIMAL.**
- Present:
  - Review `self_assessment` (placeholder "…rencana perbaikan"), `reviewer_notes` and `manager_notes` on `performance.performance_reviews`.
  - Task occurrence `review_notes` on reject, plus the `department_task_updates` comment log.
  - Logbook `review_notes`.
  - 360 `feedback_summaries.strengths[]/weaknesses[]`, sentiment and `ai_keywords` columns (backend only).
- The legacy `hris.development_plans` table exists unused.
- There are no automated improvement suggestions or retrospectives.

**(h) Task management with daily/weekly/monthly to-dos tied to department KPIs: PRESENT.**
- Backend: `backend/internal/modules/payroll/{depttasks.go,depttasks_items.go,depttasks_kpi.go,depttasks_detail.go,depttasks_http.go,depttasks_jobs.go,domain/depttasks.go,domain/depttask_views.go}`. Tables are the `hris.department_task*` set.
- Frontend: `frontend/src/features/tasks/` (22 files, 2,507 lines) and `frontend/src/features/hris/kpi/components/dept-task*.tsx`.
- How it works:
  - Tasks recur once, daily, weekly or monthly per department/assignee, with priority, weighted subtasks, attachments and a progress/comment history.
  - Head-of-division review, with personal tasks needing no review.
  - Each task may link `kpi_indicator_id` + `kpi_contribution`. Approval writes a monthly contribution; the per-department weights come from `performance.kpi_department_indicators`. Completion feeds the `task_completion` KPI.
  - Views: Hari ini / Minggu ini / Bulan ini, team board, KPI view, monitor.

========================================================
## 7. Frontend
========================================================
**Data fetching:**
- Client components use TanStack React Query v5. Each feature folder has `api.ts` (fetch via `frontend/src/lib/api-client.ts` `apiGet/apiPost/apiPut/apiPatch/apiDelete` + `buildListUrl`), plus `queries.ts`, `mutations.ts` and `query-keys.ts`. 82 files in these features use useQuery/useMutation.
- No server actions (0 `"use server"`).
- Calls go to the same-origin `/api/*`. `frontend/src/proxy.ts` rewrites ported routes to the Go backend (`lib/backend-routes` `goBackendTarget`, `BACKEND_URL`; 503 if unset). There are no Next API routes left for these features.
- Uploads use `fetch` with FormData. XLSX is built client-side with exceljs (payroll run workbook); CSV via `lib/utils/csv-export`.

**Shared UI** (`frontend/src/components`):
- `ui/`: button, input, card, badge, dialog, combobox, select, label, textarea, checkbox, table, scroll-region, confirm-dialog, tabs, numeric-input, skeleton-table, datepicker, dropdown-menu, page-header, progress, sheet, switch.
- `layout/`: form-field, form-page-layout, form-section-card.
- `iam/iam-access-provider`, `motion`, `dashboard/coming-soon-page`.
- `hris/`: ApprovalButtons, AttendanceCalendar, CameraCapture, OnboardingChecklist, PromoteCandidateButton, RichTextEditor, SafeHtml, VideoEmbed.
- Libraries: `lib/format`, `lib/accounting/*`, `lib/hris/*`, `lib/kpi/*`, `lib/recruitment/*`, `lib/payroll/*`. Charts use recharts; forms use react-hook-form + zod; toasts use sonner.

**Feature sizes** (files/lines):
- accounting 128/12,102: account-types, ap, ar, beginning-balance, cash-bank, chart-of-accounts, coming-soon, fiscal-years, journal-entries, journal-mappings, ledger, period, reports.
- finance 5/1,280: invoices.
- hris 275/26,181: analytics, announcements, attendance, candidates, contracts, dashboard, ess, holidays, job-portal, kpi (22/3,282), leaves, live-monitoring, loans, logbook, offboarding, onboarding, org-chart, overtime, payroll, pipeline (31/4,187), psikotes, reports, salary, shifts, talent-pool.
- interview-portal 12/1,020; offer-portal 6/282; psikotes-portal 13/1,102; tasks 22/2,507; audit 7/434; master-data 23/1,433; users 45/5,972.

**Pages.** All paths are under `/dashboard` unless noted (app dir `frontend/src/app/dashboard/(dashboard)/`).

Accounting (41 pages, `features/accounting/*`):
- `/accounting/reports`: reports overview / financial dashboard.
- `/accounting/reports/trial-balance`, `/balance-sheet`, `/income-statement`, `/cash-flow`, `/general-ledger`.
- `/accounting/reports/budget-vs-actual` and `/changes-in-equity`: coming soon.
- `/accounting/chart-of-accounts` (CRUD + XLSX import) and `/accounting/account-types`.
- `/accounting/journal-mappings`, `/insert`, `/edit/[id]`: event→COA templates.
- `/accounting/journal-entries`, `/insert`, `/edit/[id]`: manual journals.
- `/accounting/journal-history`: posted/auto journal history.
- `/accounting/fiscal-years`, `/insert`, `/edit/[id]`, `/beginning-balance/[id]`.
- `/accounting/beginning-balance`.
- `/accounting/period/accounting-period` (open periods) and `/accounting/period/closing` (close with blockers).
- `/accounting/cash-bank` (balances), `/cash-bank/[accountId]` (ledger), `/cash-in`, `/cash-out`, `/transfer`, `/reconciliation` (coming soon).
- `/accounting/ledger/subsidiary`.
- `/accounting/accounts-payable` (payable register), `/invoices`, `/payments` (record/void), `/aging`, `/po/[id]` (features/purchasing/po).
- `/accounting/receivable`, `/invoices`, `/receipts`, `/aging`, `/invoices-b2b` (features/finance/invoices).

Finance (1 page): `/finance/invoices`, B2B invoices with payments and Faktur Pajak.

HRIS (38 pages, including 4 redirects):
- `/dashboard/hris` redirects to `/hris/candidates`.
- Recruitment:
  - `/hris/candidates` (list), `/hris/candidates/[id]` (detail: screening, psikotes, interview, offers, notes).
  - `/hris/pipeline` (kanban), `/hris/talent-pool`, `/hris/job-portal` (job openings).
  - `/hris/psikotes` (instrument/question bank and review), `/hris/live-monitoring` and `/[type]/[id]` (proctoring).
  - `/hris/analytics` (recruitment analytics).
- Workforce:
  - `/hris/attendance` (roster, validation, export), `/hris/leaves` and `/leaves/insert`, `/hris/overtime`.
  - `/hris/shifts`; `/hris/schedules` redirects to shifts.
  - `/hris/holidays`, `/hris/contracts`, `/hris/pengumuman` (announcements).
  - `/hris/onboarding/[employee_id]`, `/hris/offboarding/[employee_id]`, `/hris/org-chart`.
  - `/hris/sections` redirects to `/employees`.
- Compensation:
  - `/hris/payroll` (runs), `/payroll/[id]` (run detail, calculate, workbook XLSX), `/payroll/[id]/payslip/[employeeId]`, `/payroll/settings` (BPJS/PPh21/OT/late/loan limits).
  - `/hris/salary`, `/salary/insert`, `/salary/[id]`; `/hris/loans`.
- Performance:
  - `/hris/kpi-scorecard`, `/hris/kpi-config` (department indicator weights); `/hris/kpi-templates` redirects to kpi-config.
  - `/hris/performance` (quarterly review cycles and realtime KPI), `/hris/dept-tasks` (department task admin, `features/hris/kpi`).
  - `/hris/logbook` (templates, entries, history, summary); `/hris/logbook-list` redirects to logbook.
- `/hris/reports`: monthly HRIS report.

People and master data (8 pages):
- `/employees`, `/employees/insert`, `/employees/edit/[id]`, `/employees/[id]` (profile tabs), all from `features/users`.
- `/master/departments`, `/master/positions`, `/master/employment-statuses` (`features/master-data`, with CSV import).

ESS (9 pages, `features/hris/ess` + kpi): `/me` (beranda), `/me/absensi` (clock in/out with selfie), `/me/cuti`, `/me/lembur`, `/me/pinjaman`, `/me/slip-gaji`, `/me/shift-tim`, `/me/pengumuman`, `/me/kpi`.

Tasks (3 pages, `features/tasks`): `/tasks` (my to-dos today/week/month with quick capture), `/tasks/team` (department board plus monitor panel), `/tasks/kpi` (KPI and tasks per indicator).

Settings (in scope): `/settings/audit` (`features/audit`, GET `/api/audit`, served by `configuration/master/audit.go`), `/settings/users` (`features/users`, `/api/users`), plus `/settings/roles` and `/settings/menus` for IAM.

`/rekrutmen` redirects to `/dashboard/me`.

Public, outside `/dashboard` (app `(public)` and `(site)`):
- `/portal?job_opening_id=&position_id=&brand_id=`: application form.
- `/career`: public job list (`(site)`).
- `/psikotes/[token]`: online psikotes runner (`features/psikotes-portal`).
- `/interview/[token]`: AI voice/text interview (`features/interview-portal`).
- `/offer/[token]`: offer accept/negotiate/decline (`features/offer-portal`).

`/os` is a desktop shell (`features/os-desktop`) that links into `/dashboard/accounting/reports`, `/dashboard/hris/*` and `/dashboard/me`; it is not a separate feature implementation.

**Page counts:** accounting 41, finance 1, hris 38, employees 4, master 3, me 9, tasks 3, settings in scope 2 (audit, users), rekrutmen 1, public portals 5. Total about 107 page.tsx files, including 7 redirect stubs and 3 coming-soon pages.

**Docs worth reading before porting:**
- `docs/hris/HRIS_FASE2_PAYROLL_COMPLETE.md`, `docs/hris/HRIS_FASE3_KPI_PERFORMANCE.md`, `docs/kpi/KPI_PROGRESS.md`, `docs/epics/EPIC-010-kpi-scorecard.md`, `docs/epics/EPIC-007-workforce-ess.md`, `docs/epics/EPIC-002-psikotes-online.md`.
- `docs/PAYROLL_PPH21.md`, referenced in the calculator.
- `docs/nuhabit-mom-2026-10-08.md`: payroll XLSX draft status and the Tedja task import plan.