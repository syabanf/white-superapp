package domain

import (
	"sort"
	"time"
)

type DataState string

const (
	StateHealthy      DataState = "healthy"
	StateStale        DataState = "stale"
	StateFailed       DataState = "failed"
	StateDemo         DataState = "demo"
	StateEmpty        DataState = "empty"
	StateNotConnected DataState = "not_connected"
)

// StaleAfter is how long after the latest sync start a module counts as stale.
const StaleAfter = 48 * time.Hour

type DataQuality struct {
	State        DataState
	LastSyncedAt *time.Time
	DataFrom     *time.Time
	DataTo       *time.Time
}

// Quality mirrors features/overview/queries.ts quality(), checked in order:
// no source, failed sync, demo, empty, stale, healthy. lastSyncedAt is the
// latest job's startedAt; without a job there is no stale check.
func Quality(hasSource, isDemo bool, points []time.Time, latest *SyncJob, now time.Time) DataQuality {
	q := DataQuality{}
	if latest != nil {
		t := latest.StartedAt
		q.LastSyncedAt = &t
	}
	if len(points) > 0 {
		sorted := append([]time.Time(nil), points...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Before(sorted[j]) })
		first, last := sorted[0], sorted[len(sorted)-1]
		q.DataFrom, q.DataTo = &first, &last
	}
	switch {
	case !hasSource:
		q.State = StateNotConnected
	case latest != nil && latest.Status == JobFailed:
		q.State = StateFailed
	case isDemo:
		q.State = StateDemo
	case len(points) == 0:
		q.State = StateEmpty
	case q.LastSyncedAt != nil && now.Sub(*q.LastSyncedAt) > StaleAfter:
		q.State = StateStale
	default:
		q.State = StateHealthy
	}
	return q
}

// IsDemo is true when every source row lacks a connection (seed data, CSV
// import, Apify-only data). No rows at all also count as demo.
func IsDemo(connectionIDs []*string) bool {
	for _, c := range connectionIDs {
		if c != nil {
			return false
		}
	}
	return true
}
