import { z } from "zod";
import { th } from "./strings";

const day = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, th.invalidDate);
const optDay = z.union([z.literal(""), day]).default("");
const hhmm = z.string().regex(/^([01]\d|2[0-3]):[0-5]\d$/, th.shifts.timeInvalid);
const money = z.string().regex(/^\d+(\.\d{1,2})?$/, th.required);
const text = (max = 200) => z.string().trim().max(max).default("");

export const employeeSchema = z.object({
  employeeNo: text(40),
  fullName: z.string().trim().min(1, th.employees.nameRequired).max(120),
  email: z.union([z.literal(""), z.string().trim().toLowerCase().email(th.employees.emailInvalid)]).default(""),
  phone: text(30),
  departmentId: text(64),
  positionId: text(64),
  managerId: text(64),
  employmentStatus: text(40),
  joinDate: z.string().min(1, th.employees.joinRequired).pipe(day),
  endDate: optDay,
  probationEndDate: optDay,
  nik: z.union([z.literal(""), z.string().regex(/^\d{16}$/, th.employees.nikInvalid)]).default(""),
  npwp: text(30),
  bpjsTkNo: text(30),
  bpjsKesNo: text(30),
  birthPlace: text(80),
  birthDate: optDay,
  gender: text(10),
  maritalStatus: text(20),
  religion: text(30),
  address: text(500),
  bankName: text(60),
  bankAccountNo: text(40),
  bankAccountName: text(120),
  emergencyContactName: text(120),
  emergencyContactPhone: text(30),
  emergencyContactRelation: text(60),
  userId: text(64),
});

export const leaveSchema = z
  .object({
    employeeId: text(64),
    leaveType: z.string().min(1, th.required),
    startDate: day,
    endDate: day,
    reason: z.string().trim().min(1, th.leave.reasonRequired).max(1000),
  })
  .refine((v) => v.endDate >= v.startDate, { path: ["endDate"], message: th.endBeforeStart });

export const overtimeSchema = z
  .object({
    employeeId: text(64),
    date: day,
    startTime: hhmm,
    endTime: hhmm,
    reason: z.string().trim().min(1, th.required).max(1000),
  })
  .refine((v) => v.endTime !== v.startTime, { path: ["endTime"], message: th.overtime.timeOrder });

export const contractSchema = z
  .object({
    employeeId: z.string().min(1, th.required),
    contractType: z.enum(["pkwt", "pkwtt"]),
    startDate: day,
    endDate: optDay,
    probationEndDate: optDay,
    positionTitle: text(120),
    departmentName: text(120),
    workLocation: text(200),
    baseSalary: z.union([z.literal(""), money]).default(""),
    notes: text(1000),
  })
  .refine((v) => v.contractType !== "pkwt" || v.endDate !== "", { path: ["endDate"], message: th.contracts.pkwtEndRequired })
  .refine((v) => !v.endDate || v.endDate >= v.startDate, { path: ["endDate"], message: th.endBeforeStart });

export const announcementSchema = z
  .object({
    title: z.string().trim().min(1, th.announcements.titleRequired).max(200),
    body: z.string().trim().min(1, th.announcements.bodyRequired).max(20000),
    tags: z.array(z.string().trim().min(1).max(40)).max(10).default([]),
    status: z.enum(["draft", "published", "archived"]),
    isPinned: z.boolean().default(false),
    targetScope: z.enum(["global", "departments"]),
    departmentIds: z.array(z.string()).default([]),
    publishAt: z.string().nullable().default(null),
    expiresAt: z.string().nullable().default(null),
  })
  .refine((v) => v.targetScope === "global" || v.departmentIds.length > 0, {
    path: ["departmentIds"],
    message: th.announcements.departmentsRequired,
  });

export const salarySchema = z.object({
  employeeId: z.string().min(1, th.required),
  baseSalary: money.refine((v) => Number(v) > 0, th.salary.baseRequired),
  fixedAllowance: money.default("0.00"),
  variableAllowance: money.default("0.00"),
  transportAllowance: money.default("0.00"),
  mealAllowance: money.default("0.00"),
  housingAllowance: money.default("0.00"),
  ptkpStatus: z.string().min(1, th.required),
  isTaxable: z.boolean(),
  bpjsTkEnrolled: z.boolean(),
  bpjsKesEnrolled: z.boolean(),
  taperaEnrolled: z.boolean(),
  effectiveDate: day,
  notes: text(500),
});

export const loanSchema = z.object({
  employeeId: text(64),
  loanType: z.enum(["kasbon", "loan", "emergency"]),
  principalAmount: money.refine((v) => Number(v) > 0, th.loans.principalRequired),
  interestRate: z.union([z.literal(""), z.string().regex(/^\d+(\.\d{1,2})?$/)]).default(""),
  tenorMonths: z.coerce.number().int().min(1, th.loans.tenorRange).max(24, th.loans.tenorRange),
  purpose: z.string().trim().min(1, th.loans.purposeRequired).max(500),
});

export const additionSchema = z.object({
  employeeId: z.string().min(1, th.required),
  amount: money.refine((v) => Number(v) > 0, th.additions.amountRequired),
  label: z.string().trim().min(1, th.additions.labelRequired).max(120),
  periodMonth: z.coerce.number().int().min(0).max(12).default(0),
  periodYear: z.coerce.number().int().min(0).default(0),
});

export const shiftSchema = z.object({
  name: z.string().trim().min(1, th.required).max(60),
  startTime: hhmm,
  endTime: hhmm,
  breakMinutes: z.coerce.number().int().min(0).max(480),
  lateToleranceMinutes: z.coerce.number().int().min(0).max(240),
  isOvernight: z.boolean(),
  isActive: z.boolean(),
  sortOrder: z.coerce.number().int().min(0).default(0),
});

export const holidaySchema = z.object({
  holidayDate: day,
  name: z.string().trim().min(1, th.required).max(120),
  type: z.enum(["nasional", "cuti_bersama", "perusahaan"]),
  deductsLeave: z.boolean(),
  status: z.enum(["draft", "aktif"]),
  note: text(300),
});

export const departmentSchema = z.object({
  code: z.string().trim().min(1, th.masters.codeRequired).max(20),
  name: z.string().trim().min(1, th.masters.nameRequired).max(80),
  parentId: text(64),
  headEmployeeId: text(64),
  isActive: z.boolean(),
});

export const positionSchema = z
  .object({
    title: z.string().trim().min(1, th.masters.nameRequired).max(80),
    departmentId: text(64),
    level: z.string().min(1, th.required),
    salaryMin: z.union([z.literal(""), money]).default(""),
    salaryMax: z.union([z.literal(""), money]).default(""),
    isActive: z.boolean(),
  })
  .refine((v) => !v.salaryMin || !v.salaryMax || Number(v.salaryMax) >= Number(v.salaryMin), {
    path: ["salaryMax"],
    message: th.masters.rangeInvalid,
  });

export const statusOptionSchema = z.object({
  code: z
    .string()
    .trim()
    .min(1, th.masters.codeRequired)
    .max(30)
    .regex(/^[a-z0-9_]+$/, th.masters.codeRequired),
  name: z.string().trim().min(1, th.masters.nameRequired).max(60),
  isActive: z.boolean(),
  sortOrder: z.coerce.number().int().min(0),
});
