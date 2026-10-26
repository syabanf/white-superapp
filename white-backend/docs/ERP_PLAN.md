# ERP workspace: plan and shared decisions

The platform has two workspaces: **Digital Marketing Platform** (the existing
analytics, publishing, SEO suite and reports) and **ERP** (finance, HR,
recruitment, sales). The ERP domains are ported from NüHabit
(`~/Documents/works/new-nuhabit`, inventories in `docs/inventory/nuhabit/`)
and extended with the agency automations listed below.

## Services

| Service | gRPC port | Schema | Source in NüHabit | New for WHITE |
|---|---|---|---|---|
| `finance` | 9007 | `finance` | `accounting` module (COA, account types, fiscal years/periods, journals, journal mappings, cash-bank, AP, AR, reports, dashboard) and `/api/finance` B2B invoices | reimbursement claims, financial ratios and AI analysis |
| `hr` | 9008 | `hr` | `hris` (employees, master data, attendance, leave, overtime, shifts, holidays, contracts, announcements, employee onboarding/offboarding, org chart, HR reports, ESS) and the compensation half of `payroll` (payroll runs, BPJS/PPh21, salary, loans, payslips) | |
| `performance` | 9012 | `performance` | the performance half of `payroll` (KPI config and scorecards, quarterly reviews, 360 feedback, department tasks) and `hris` logbook | daily and monthly work recap, deadline reminders, "things to improve" |
| `recruitment` | 9009 | `recruitment` | `recruitment` (job openings, candidates, pipeline, talent pool, psikotes, AI interview, offers, public portals) | |
| `sales` | 9010 | `sales` | `salesfunnel` (leads, accounts, contacts, pipelines, deals, activities, quotations with versioning, invoices, payments, targets, forecast) and `crm/advance` (discount approvals, workflow rules, scoring, custom fields) and the WA inbox | contract generation, pitch deck generation, client onboarding, AI chatbot replies to leads |
| `content` (marketing) | 9011 | `content` | none | editorial plan drafts, content reports, content edit/approval, design, asset library, story posting (with publishing) |

## Rules every ERP service follows

- Same layout and conventions as the marketing services (`docs/CONVENTIONS.md`, reference `internal/identity`).
- **Single company.** WHITE is one agency, so NüHabit's holding/company/branch scope is dropped. Keep `company_id` out of the schema.
- **Authorization** uses permission keys from `pkg/authn/perms.go`: `authn.RequirePerm(ctx, authn.PermFinanceWrite)`. ADMIN holds every key. `self.service` covers what any employee does for themselves (own attendance, leave, reimbursement, tasks, payslip). NüHabit IAM menu prefixes map as: `accounting` -> `finance.*`, `hris*` -> `hr.*`, `hris.compensation` -> `hr.payroll`, `hris.recruitment` -> `recruitment.*`, `sales-funnel` / `crm*` -> `sales.*`, `tasks*` -> `self.service` plus `hr.read` for team views.
- **People.** `identity.users` are logins. `hr.employees` are people on the payroll, with an optional `user_id`. Other services resolve "the employee of this user" through `hr.EmployeeService.GetEmployeeByUser`.
- **Cross-service effects go through events** (`pkg/events`), never cross-schema SQL:
  - `white.hr.payroll.paid` -> finance posts the payroll journals (PAYROLL_* event codes).
  - `white.sales.invoice.issued` / `white.sales.payment.recorded` -> finance creates AR invoices and receipts.
  - `white.finance.reimbursement.paid` -> hr (payslip addition when paid through payroll).
  - `white.sales.deal.won` -> sales starts client onboarding; identity creates the marketing client when the onboarding asks for it.
  - `white.client.deleted`, `white.user.deleted` as today.
- **Notifications** go through identity `NotificationService.Notify`. WhatsApp and email delivery are out of scope for the first cut; keep a `Notifier` port so they can be added.
- **AI** uses OpenRouter through integrations credentials (`SettingsService.GetProviderCredentials(OPENROUTER)`), with a deterministic mock when no key is set, the same way `internal/reports` does.
- **Scheduled work** runs from scheduler ticks (`white.erp.tick.15m`, `white.erp.tick.hourly`, `white.erp.tick.daily`) so services stay stateless.
- **Indonesian rules** (PPh21 TER, BPJS, overtime per PP 35/2021, leave balances, journal numbering) are ported verbatim from NüHabit with their tests.
- Money is `numeric(18,2)` in SQL and integer rupiah or decimal strings in proto; never float for stored amounts.

## Status

All six phases are built: permission keys and the session bridge, the
workspace switcher, the six services above with demo seeds, and the pages
(`/erp/**` for ERP, `/clients/[slug]/content/**` for content, public pages
`/karier`, `/psikotes`, `/wawancara`, `/offer`, `/f`, `/kontrak`,
`/onboarding`, `/chat`).

Known gaps:
- The marketing pages in Next.js still read Prisma, while content schedules
  posts into the Go publishing service, so those posts do not show in the
  Publikasi calendar until Publikasi moves to the gateway.
- `recruitment.StartAiInterview` still requires `webcam_consent` although
  there is no camera capture; the portal sends it with the candidate's
  consent checkbox.
- Pickers load up to 500 rows (leads, deals, employees); a search endpoint
  per service would scale better.
