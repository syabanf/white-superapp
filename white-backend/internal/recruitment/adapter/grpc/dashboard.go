package grpcapi

import (
	"context"
	"math"
	"time"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

func (s *candidateServer) GetRecruitmentDashboard(ctx context.Context, req *pb.RecruitmentDashboardRequest) (*pb.RecruitmentDashboard, error) {
	d, err := s.svc.Candidates.Dashboard(ctx, app.DashboardQuery{From: req.GetFrom(), To: req.GetTo(), Compare: req.GetCompare(),
		Filter: domain.DashboardFilter{JobOpeningID: req.GetJobOpeningId(), Department: req.GetDepartment(), Source: req.GetSource(),
			Stage: req.GetStage()}})
	if err != nil {
		return nil, err
	}
	now := d.Now
	det := d.Detail
	out := &pb.RecruitmentDashboard{
		Period:            periodPB(d.Current),
		ComparePeriod:     periodPB(d.Previous),
		Compare:           d.Compare,
		Kpis:              kpisPB(d.Kpis),
		PreviousKpis:      kpisPB(d.PrevKpis),
		Trend:             trendPB(d.Trend),
		PreviousTrend:     trendPB(d.PrevTrend),
		StuckAfterDays:    domain.StuckAfterDays,
		PsikotesStatus:    countsPB(det.PsikotesStatus),
		AiInterviewStatus: countsPB(det.AiStatus),
	}
	for _, f := range d.Funnel {
		out.Funnel = append(out.Funnel, &pb.FunnelStage{Stage: f.Stage, Label: statusLabel(f.Stage), Reached: int32(f.Reached),
			Current: int32(f.Current), Exited: int32(f.Exited)})
	}
	for _, x := range det.StageDurations {
		out.StageDurations = append(out.StageDurations, &pb.StageDuration{Stage: x.Stage, Label: statusLabel(x.Stage), Count: int32(x.N),
			AvgDays: round1(x.Avg), MinDays: round1(x.Min), P25Days: round1(x.P25), MedianDays: round1(x.Median), P75Days: round1(x.P75),
			MaxDays: round1(x.Max)})
	}
	for _, x := range det.OpeningStages {
		out.OpeningStages = append(out.OpeningStages, &pb.OpeningStageCount{JobOpeningId: x.JobOpeningID, Title: x.Title, Status: x.Status,
			Count: int32(x.N)})
	}
	for _, x := range det.Stuck {
		out.Stuck = append(out.Stuck, &pb.StuckCandidate{Id: x.ID, FullName: x.FullName, JobOpeningId: x.JobOpeningID,
			JobOpeningTitle: x.Title, Status: x.Status, EnteredAt: ts(x.EnteredAt), DaysInStage: int32(now.Sub(x.EnteredAt).Hours() / 24)})
	}
	for _, x := range d.Sources {
		out.Sources = append(out.Sources, &pb.SourceStat{Source: x.Source, Label: domain.SourceLabel(x.Source), Paid: x.Paid,
			Applications: int32(x.Applications), Screened: int32(x.Screened), Interviewed: int32(x.Interviewed), Offered: int32(x.Offered),
			Hires: int32(x.Hires), AvgDaysToHire: x.AvgDaysToHire})
	}
	for _, x := range det.SourceTrend {
		out.SourceTrend = append(out.SourceTrend, &pb.SourceWeek{Start: x.Start, Source: x.Source, Count: int32(x.N)})
	}
	for _, x := range det.Openings {
		out.Openings = append(out.Openings, openingPB(x, now))
	}
	for _, x := range det.PsikotesScores {
		out.PsikotesScores = append(out.PsikotesScores, &pb.ScoreBucket{Instrument: x.Instrument, Bucket: int32(x.Bucket), Count: int32(x.N),
			AvgScore: round1(x.AvgScore)})
	}
	for _, x := range det.Scorecards {
		out.Scorecards = append(out.Scorecards, &pb.CriterionScore{Type: x.Type, Criterion: x.Criterion, Count: int32(x.N),
			AvgScore: round2(x.Avg)})
	}
	for _, x := range det.Interviewers {
		l := &pb.InterviewerLoad{Name: x.Name, EmployeeId: x.EmployeeID, Scheduled: int32(x.Scheduled), Completed: int32(x.Completed),
			Cancelled: int32(x.Cancelled), Proceed: int32(x.Proceed), Pool: int32(x.Pool), Reject: int32(x.Reject)}
		if x.AvgScore != nil {
			v := round2(*x.AvgScore)
			l.AvgScore = &v
		}
		out.Interviewers = append(out.Interviewers, l)
	}
	for _, x := range det.Offers {
		o := &pb.DashboardOffer{CandidateId: x.CandidateID, CandidateName: x.FullName, JobOpeningId: x.JobOpeningID, PositionTitle: x.Title,
			Status: x.Status, BaseSalary: rupiah(x.BaseSalary), SalaryMin: rupiahPtr(x.SalaryMin), SalaryMax: rupiahPtr(x.SalaryMax),
			ExpectedSalary: x.ExpectedSalary, SentAt: ts(x.SentAt), RespondedAt: tsPtr(x.RespondedAt), Versions: int32(x.Versions)}
		if x.RespondedAt != nil {
			v := round1(x.RespondedAt.Sub(x.SentAt).Hours() / 24)
			o.ResponseDays = &v
		}
		out.Offers = append(out.Offers, o)
	}
	return out, nil
}

func periodPB(p domain.Period) *pb.DashboardPeriod {
	return &pb.DashboardPeriod{From: p.From.Format(domain.IsoDate), To: p.To.Format(domain.IsoDate)}
}

func kpisPB(k domain.Kpis) *pb.RecruitmentKpis {
	return &pb.RecruitmentKpis{OpenPositions: int32(k.OpenPositions), OpenHeadcount: int32(k.OpenHeadcount), Applicants: int32(k.Applicants),
		InProcess: int32(k.InProcess), OffersSent: int32(k.OffersSent), OffersAccepted: int32(k.OffersAccepted),
		OffersDeclined: int32(k.OffersDeclined), OffersExpired: int32(k.OffersExpired), Hires: int32(k.Hires),
		AvgDaysToHire: k.AvgDaysToHire, OfferAcceptanceRate: k.OfferAcceptanceRate}
}

func trendPB(list []domain.TrendBucket) []*pb.RecruitmentTrendPoint {
	out := make([]*pb.RecruitmentTrendPoint, len(list))
	for i, b := range list {
		p := &pb.RecruitmentTrendPoint{Start: b.Start, Applications: int32(b.Applications), Hires: int32(b.Hires),
			OffersSent: int32(b.OffersSent), OffersAccepted: int32(b.OffersAccepted), OffersDeclined: int32(b.OffersDeclined),
			OffersExpired: int32(b.OffersExpired), OffersNegotiating: int32(b.OffersNegotiating), InProcess: int32(b.InProcess),
			OpenPositions: int32(b.OpenPositions), OpenHeadcount: int32(b.OpenHeadcount)}
		if b.Hires > 0 {
			v := round1(b.HireDays / float64(b.Hires))
			p.AvgDaysToHire = &v
		}
		out[i] = p
	}
	return out
}

func openingPB(o domain.OpeningStat, now time.Time) *pb.OpeningStat {
	end := now
	out := &pb.OpeningStat{Id: o.ID, Title: o.Title, Department: o.Department, Status: o.Status, Headcount: int32(o.Headcount),
		OpenedOn: domain.DayKey(o.OpenedAt), ClosingDate: o.ClosingDate, ApplicantsTotal: int32(o.ApplicantsTotal),
		ApplicantsPeriod: int32(o.ApplicantsPeriod), Hires: int32(o.Hired), TimeToFillDays: roundPtr(o.FillDays)}
	if o.ClosedAt != nil {
		// the window ends at midnight after the last day
		out.WindowEnd = domain.DayKey(o.ClosedAt.Add(-time.Second))
		if o.ClosedAt.Before(end) {
			end = *o.ClosedAt
		}
	}
	out.DaysOpen = int32(max(0, end.Sub(o.OpenedAt).Hours()/24))
	if o.ClosingDate != nil && o.Status == "published" {
		if day, ok := domain.ParseDay(*o.ClosingDate); ok {
			today, _ := domain.ParseDay(domain.DayKey(now))
			left := int32(day.Sub(today).Hours() / 24)
			out.DaysToClose = &left
		}
	}
	for _, c := range o.Stages {
		out.Stages = append(out.Stages, &pb.CountByKey{Key: c.Key, Label: statusLabel(c.Key), Count: int32(c.Count)})
	}
	return out
}

func statusLabel(status string) string {
	label, _ := domain.StatusLabel(status)
	return label
}

func rupiah(m domain.Money) int64 { return int64(m) / 100 }

func rupiahPtr(m *domain.Money) *int64 {
	if m == nil {
		return nil
	}
	v := rupiah(*m)
	return &v
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func roundPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	r := round1(*v)
	return &r
}
