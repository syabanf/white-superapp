package postgres

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

// scope is the period bounds and filters every dashboard query takes.
type scope struct {
	from, to                       time.Time
	job, department, source, stage *string
}

func newScope(f domain.DashboardFilter, p domain.Period) scope {
	return scope{from: p.FromTS, to: p.ToTS, job: nilIfEmpty(f.JobOpeningID), department: nilIfEmpty(f.Department),
		source: nilIfEmpty(f.Source), stage: nilIfEmpty(f.Stage)}
}

func (s *Store) DashboardTrend(ctx context.Context, f domain.DashboardFilter, p domain.Period) ([]domain.TrendBucket, error) {
	sc := newScope(f, p)
	rows, err := s.q.DashboardTrend(ctx, sqlc.DashboardTrendParams{Active: domain.ActiveStatuses, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage, FromTs: sc.from, ToTs: sc.to, FromDay: p.From, ToDay: p.To})
	return each(rows, err, func(r sqlc.DashboardTrendRow) domain.TrendBucket {
		return domain.TrendBucket{Start: r.Bucket.Format(domain.IsoDate), Applications: int(r.Applications), Hires: int(r.Hires),
			HireDays: r.HireDays, OffersSent: int(r.OffersSent), OffersAccepted: int(r.OffersAccepted), OffersDeclined: int(r.OffersDeclined),
			OffersExpired: int(r.OffersExpired), OffersNegotiating: int(r.OffersNegotiating), InProcess: int(r.InProcess),
			OpenPositions: int(r.OpenPositions), OpenHeadcount: int(r.OpenHeadcount)}
	})
}

// DashboardDetail runs the breakdown queries of a period one after another;
// each is small and they share the pool with the rest of the service.
func (s *Store) DashboardDetail(ctx context.Context, f domain.DashboardFilter, p domain.Period, now time.Time) (*domain.DashboardDetail, error) {
	sc := newScope(f, p)
	d := &domain.DashboardDetail{}

	cohort, err := s.q.DashboardCohort(ctx, sqlc.DashboardCohortParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.Cohort, err = each(cohort, err, func(r sqlc.DashboardCohortRow) domain.CohortRow {
		return domain.CohortRow{Source: r.Source, Status: r.Status, MaxRank: int(r.MaxRank), N: int(r.N), HireDays: r.HireDays}
	}); err != nil {
		return nil, err
	}

	durations, err := s.q.DashboardStageDurations(ctx, sqlc.DashboardStageDurationsParams{FromTs: sc.from, ToTs: sc.to,
		JobOpeningID: sc.job, Department: sc.department, Source: sc.source, Stage: sc.stage, Now: now, Active: domain.ActiveStatuses})
	if d.StageDurations, err = each(durations, err, func(r sqlc.DashboardStageDurationsRow) domain.StageDuration {
		return domain.StageDuration{Stage: r.Status, N: int(r.N), Avg: r.AvgDays, Min: r.MinDays, P25: r.P25Days, Median: r.MedianDays,
			P75: r.P75Days, Max: r.MaxDays}
	}); err != nil {
		return nil, err
	}

	stages, err := s.q.DashboardOpeningStages(ctx, sqlc.DashboardOpeningStagesParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.OpeningStages, err = each(stages, err, func(r sqlc.DashboardOpeningStagesRow) domain.OpeningStageCount {
		return domain.OpeningStageCount{JobOpeningID: r.JobOpeningID, Title: r.Title, Status: r.Status, N: int(r.N)}
	}); err != nil {
		return nil, err
	}

	stuck, err := s.q.DashboardStuck(ctx, sqlc.DashboardStuckParams{Active: domain.ActiveStatuses,
		Before: now.AddDate(0, 0, -domain.StuckAfterDays), JobOpeningID: sc.job, Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.Stuck, err = each(stuck, err, func(r sqlc.DashboardStuckRow) domain.StuckCandidate {
		return domain.StuckCandidate{ID: r.ID, FullName: r.FullName, JobOpeningID: r.JobOpeningID, Title: r.Title, Status: r.Status,
			EnteredAt: r.EnteredAt}
	}); err != nil {
		return nil, err
	}

	weeks, err := s.q.DashboardSourceTrend(ctx, sqlc.DashboardSourceTrendParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.SourceTrend, err = each(weeks, err, func(r sqlc.DashboardSourceTrendRow) domain.SourceWeek {
		return domain.SourceWeek{Start: r.Bucket.Format(domain.IsoDate), Source: r.Source, N: int(r.N)}
	}); err != nil {
		return nil, err
	}

	openings, err := s.q.DashboardOpenings(ctx, sqlc.DashboardOpeningsParams{FromTs: sc.from, ToTs: sc.to, Source: sc.source,
		Stage: sc.stage, JobOpeningID: sc.job, Department: sc.department})
	if d.Openings, err = each(openings, err, openingStat); err != nil {
		return nil, err
	}

	psi, err := s.q.DashboardPsikotesStatus(ctx, sqlc.DashboardPsikotesStatusParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.PsikotesStatus, err = each(psi, err, func(r sqlc.DashboardPsikotesStatusRow) domain.Count {
		return domain.Count{Key: r.Status, Count: int(r.N)}
	}); err != nil {
		return nil, err
	}

	scores, err := s.q.DashboardPsikotesScores(ctx, sqlc.DashboardPsikotesScoresParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.PsikotesScores, err = each(scores, err, func(r sqlc.DashboardPsikotesScoresRow) domain.ScoreBucket {
		return domain.ScoreBucket{Instrument: r.Instrument, Bucket: int(r.Bucket), N: int(r.N), AvgScore: r.AvgScore}
	}); err != nil {
		return nil, err
	}

	ai, err := s.q.DashboardAiInterviewStatus(ctx, sqlc.DashboardAiInterviewStatusParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.AiStatus, err = each(ai, err, func(r sqlc.DashboardAiInterviewStatusRow) domain.Count {
		return domain.Count{Key: r.Status, Count: int(r.N)}
	}); err != nil {
		return nil, err
	}

	cards, err := s.q.DashboardScorecards(ctx, sqlc.DashboardScorecardsParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.Scorecards, err = each(cards, err, func(r sqlc.DashboardScorecardsRow) domain.CriterionScore {
		return domain.CriterionScore{Type: r.Type, Criterion: r.Criterion, N: int(r.N), Avg: r.AvgScore}
	}); err != nil {
		return nil, err
	}

	people, err := s.q.DashboardInterviewers(ctx, sqlc.DashboardInterviewersParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.Interviewers, err = each(people, err, func(r sqlc.DashboardInterviewersRow) domain.InterviewerLoad {
		l := domain.InterviewerLoad{Name: r.Interviewer, EmployeeID: r.EmployeeID, Scheduled: int(r.Scheduled), Completed: int(r.Completed),
			Cancelled: int(r.Cancelled), Proceed: int(r.Proceed), Pool: int(r.Pool), Reject: int(r.Reject)}
		if r.AvgScore >= 0 {
			l.AvgScore = &r.AvgScore
		}
		return l
	}); err != nil {
		return nil, err
	}

	offers, err := s.q.DashboardOffers(ctx, sqlc.DashboardOffersParams{FromTs: sc.from, ToTs: sc.to, JobOpeningID: sc.job,
		Department: sc.department, Source: sc.source, Stage: sc.stage})
	if d.Offers, err = each(offers, err, func(r sqlc.DashboardOffersRow) domain.OfferRow {
		return domain.OfferRow{CandidateID: r.CandidateID, FullName: r.FullName, JobOpeningID: r.JobOpeningID, Title: r.Title,
			Status: r.Status, BaseSalary: r.BaseSalary, SalaryMin: r.SalaryMin, SalaryMax: r.SalaryMax, ExpectedSalary: r.ExpectedSalary,
			SentAt: r.SentAt, RespondedAt: r.RespondedAt, Versions: int(r.Versions)}
	}); err != nil {
		return nil, err
	}
	return d, nil
}

func openingStat(r sqlc.DashboardOpeningsRow) domain.OpeningStat {
	o := domain.OpeningStat{ID: r.ID, Title: r.Title, Department: r.Department, Status: r.Status, Headcount: int(r.Headcount),
		OpenedAt: r.OpenedAt, ApplicantsTotal: int(r.ApplicantsTotal), ApplicantsPeriod: int(r.ApplicantsPeriod), Hired: int(r.Hired),
		Stages: []domain.Count{{Key: "applied", Count: int(r.Applied)}, {Key: "screening", Count: int(r.Screening)},
			{Key: "psikotes", Count: int(r.Psikotes)}, {Key: "interview", Count: int(r.Interview)}, {Key: "offer", Count: int(r.Offer)},
			{Key: "hired", Count: int(r.Hired)}, {Key: "rejected", Count: int(r.Rejected)}, {Key: "talent_pool", Count: int(r.Pooled)}}}
	if r.HasClosed {
		o.ClosedAt = &r.ClosedAt
	}
	if r.ClosingDate != nil {
		day := r.ClosingDate.Format(domain.IsoDate)
		o.ClosingDate = &day
	}
	if r.FillDays >= 0 {
		o.FillDays = &r.FillDays
	}
	return o
}
