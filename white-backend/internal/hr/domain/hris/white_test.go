package hris

import (
	"testing"
	"time"
)

func TestDecideLeave(t *testing.T) {
	mgr, hr, both, owner := LeaveActor{IsManager: true}, LeaveActor{IsHR: true}, LeaveActor{IsHR: true, IsManager: true}, LeaveActor{IsOwner: true, IsHR: true}
	cases := []struct {
		cur     string
		a       LeaveActor
		approve bool
		want    string
	}{
		{LeavePending, mgr, true, LeaveManagerApproved},
		{LeavePending, hr, true, LeaveApproved},
		{LeavePending, both, true, LeaveApproved},
		{LeaveManagerApproved, hr, true, LeaveApproved},
		{LeavePending, mgr, false, LeaveRejected},
		{LeaveManagerApproved, hr, false, LeaveRejected},
		{LeaveManagerApproved, mgr, true, ""},
		{LeavePending, LeaveActor{}, true, ""},
		{LeavePending, owner, true, ""},
		{LeaveApproved, hr, true, ""},
	}
	for i, c := range cases {
		got, reason := DecideLeave(c.cur, c.a, c.approve)
		if got != c.want || (got == "") == (reason == "") {
			t.Errorf("case %d: %q %q, want %q", i, got, reason, c.want)
		}
	}
	if ok, _ := CanCancelLeave(LeavePending, LeaveActor{IsOwner: true}); !ok {
		t.Fatal("owner cancels pending")
	}
	if ok, _ := CanCancelLeave(LeaveApproved, LeaveActor{IsOwner: true}); ok {
		t.Fatal("owner cannot cancel approved")
	}
	if ok, _ := CanCancelLeave(LeaveApproved, LeaveActor{IsHR: true}); !ok {
		t.Fatal("hr cancels approved")
	}
}

func TestWorkHoursAndReminders(t *testing.T) {
	in := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)
	if h := WorkHours(in, in.Add(9*time.Hour), nil); h != 8 {
		t.Fatal(h)
	}
	zero := 0
	if h := WorkHours(in, in.Add(30*time.Minute), &zero); h != 0.5 {
		t.Fatal(h)
	}
	if h := WorkHours(in, in.Add(30*time.Minute), nil); h != 0 {
		t.Fatal(h)
	}
	if n, ok := ReminderDue("2026-10-01", "2026-10-31", ContractReminderDays); !ok || n != 30 {
		t.Fatal(n, ok)
	}
	if _, ok := ReminderDue("2026-10-01", "2026-10-20", ContractReminderDays); ok {
		t.Fatal("19 days is not a reminder day")
	}
	if n, ok := ReminderDue("2026-10-01", "2026-10-01", ProbationReminderDays); !ok || n != 0 {
		t.Fatal(n, ok)
	}
	if !IsBirthday("1990-10-26", "2026-10-26") || IsBirthday("1990-10-25", "2026-10-26") {
		t.Fatal("birthday")
	}
	if !IsBirthday("2000-02-29", "2027-02-28") || IsBirthday("2000-02-29", "2028-02-28") {
		t.Fatal("leap birthday")
	}
}
