package domain

import (
	"strings"
	"testing"
	"time"
)

func TestFollowupStageDue(t *testing.T) {
	due := time.Date(2026, 11, 10, 0, 0, 0, 0, Jakarta)
	at := func(day int) time.Time { return time.Date(2026, 11, day, 9, 0, 0, 0, Jakarta) }
	for day, want := range map[int]int{7: -3, 8: -3, 9: -3, 10: 0, 12: 0, 13: 3, 15: 3, 17: 7, 19: 7} {
		if got, ok := FollowupStageDue(due, at(day)); !ok || got != want {
			t.Errorf("day %d = %d %v, want %d", day, got, ok, want)
		}
	}
	for _, day := range []int{5, 6, 16, 20, 30} {
		if got, ok := FollowupStageDue(due, at(day)); ok {
			t.Errorf("day %d queued stage %d", day, got)
		}
	}
}

func TestFollowupSendable(t *testing.T) {
	for _, c := range []struct {
		at   time.Time
		want bool
	}{
		{time.Date(2026, 11, 4, 8, 0, 0, 0, Jakarta), true},
		{time.Date(2026, 11, 4, 17, 59, 0, 0, Jakarta), true},
		{time.Date(2026, 11, 4, 18, 0, 0, 0, Jakarta), false},
		{time.Date(2026, 11, 4, 7, 59, 0, 0, Jakarta), false},
		{time.Date(2026, 11, 7, 10, 0, 0, 0, Jakarta), false},
		{time.Date(2026, 11, 4, 3, 0, 0, 0, time.UTC), true},
	} {
		if got := FollowupSendable(c.at); got != c.want {
			t.Errorf("%s = %v", c.at, got)
		}
	}
}

func TestFollowupMessage(t *testing.T) {
	due := time.Date(2026, 11, 10, 0, 0, 0, 0, Jakarta)
	for stage, want := range map[int]string{-3: "akan jatuh tempo pada 10-11-2026", 0: "jatuh tempo hari ini", 7: "sudah lewat 7 hari"} {
		if msg := FollowupMessage("Bu Ratna", "INV-1", 150000000, due, stage); !strings.Contains(msg, want) || !strings.Contains(msg, "Halo Bu Ratna") || !strings.Contains(msg, "INV-1") {
			t.Errorf("stage %d: %s", stage, msg)
		}
	}
}
