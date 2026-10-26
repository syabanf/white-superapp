import { z } from "zod";
import { rs } from "./strings";

const text = (max = 5000) => z.string().trim().max(max).default("");
const money = z
  .string()
  .trim()
  .regex(/^(\d+(\.\d{1,2})?)?$/)
  .default("");
const day = z
  .string()
  .regex(/^(\d{4}-\d{2}-\d{2})?$/)
  .default("");

export const jobSchema = z
  .object({
    title: z.string().trim().min(1, rs.jobs.titleRequired).max(200),
    slug: text(200),
    departmentId: text(64),
    department: text(120),
    location: text(200),
    employmentType: text(60),
    workMode: text(60),
    headcount: z.coerce.number().int().min(1).max(999).default(1),
    description: text(),
    requirements: text(),
    benefits: text(),
    salaryMin: money,
    salaryMax: money,
    hiringManagerUserId: text(64),
    status: z.enum(["draft", "published", "closed"]).default("draft"),
    closingDate: day,
  })
  .refine((v) => !v.salaryMin || !v.salaryMax || Number(v.salaryMax) >= Number(v.salaryMin), {
    message: rs.jobs.salaryOrder,
    path: ["salaryMax"],
  });

const rupiahInt = z
  .string()
  .trim()
  .regex(/^\d*$/)
  .default("");

export const candidateSchema = z.object({
  fullName: z.string().trim().min(1, rs.candidates.nameRequired).max(200),
  email: z.union([z.literal(""), z.string().trim().email(rs.candidates.emailInvalid)]).default(""),
  phone: text(40),
  domicile: text(200),
  source: text(40),
  jobOpeningId: text(64),
  status: z.enum(["applied", "screening"]).default("applied"),
  notes: text(),
  lastExperience: text(500),
  lastEducation: text(500),
  availability: text(20),
  expectedSalary: rupiahInt,
});

export const screeningSchema = z.object({
  contacted: z.boolean(),
  interested: z.boolean().optional(),
  availabilityNote: text(200),
  confirmedSalary: rupiahInt,
  willingShift: z.boolean().optional(),
  willingPlacement: z.boolean().optional(),
  notes: text(),
  recommendation: z.enum(["", "lolos", "hold", "tidak_lolos"]).default(""),
});

export const interviewSchema = z
  .object({
    candidateId: z.string().min(1, rs.interviews.pickCandidate),
    interviewerEmployeeId: text(64),
    interviewerName: text(120),
    interviewDate: z.string().min(1, rs.interviews.dateRequired),
    durationMinutes: z.coerce.number().int().min(15).max(480).default(60),
    type: z.enum(["hrd", "hiring_manager"]).default("hrd"),
    mode: z.enum(["offline", "online"]).default("online"),
    meetingLink: text(500),
    location: text(300),
    notes: text(),
  })
  .refine((v) => v.interviewerEmployeeId || v.interviewerName, {
    message: rs.interviews.interviewerRequired,
    path: ["interviewerName"],
  });

export const scorecardSchema = z.object({
  scorecard: z
    .array(z.object({ criterion: z.string().trim().min(1).max(120), score: z.number().int().min(1).max(5), note: text(500) }))
    .min(1),
  recommendation: z.enum(["proceed", "pool", "reject"]),
  notes: text(),
});

export const offerSchema = z.object({
  baseSalary: z.string().trim().regex(/^\d+(\.\d{1,2})?$/, rs.offers.salaryRequired),
  benefits: z.array(z.string().trim().min(1).max(300)).max(30).default([]),
  startDate: day,
  notes: text(),
  expiresDays: z.coerce.number().int().min(1).max(30).default(7),
  positionTitle: text(200),
});

export const instrumentSchema = z.object({
  name: z.string().trim().min(1).max(120),
  isActive: z.boolean(),
  durationSeconds: z.number().int().min(60).max(4 * 3600),
  questionCount: z.number().int().min(1).max(500).nullable(),
  shuffle: z.boolean(),
  instructions: text(),
});

export const questionSchema = z.object({
  body: text(),
  options: z.array(z.object({ key: z.string().trim().min(1).max(4), text: z.string().trim().min(1).max(1000) })).default([]),
  statementA: z.object({ text: text(1000), scale: text(4) }).optional(),
  statementB: z.object({ text: text(1000), scale: text(4) }).optional(),
  correctKey: text(4),
  sortOrder: z.coerce.number().int().min(0).default(0),
  isActive: z.boolean().default(true),
});
