package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
)

const setupKey = "setup"

type SetupState struct {
	CompletedAt *time.Time `json:"completedAt"`
	SkippedAt   *time.Time `json:"skippedAt"`
	Clients     int32      `json:"-"`
}

type Workspace struct {
	store domain.Store
	clock clock.Clock
}

func NewWorkspace(store domain.Store, clk clock.Clock) *Workspace {
	return &Workspace{store: store, clock: clk}
}

func (w *Workspace) SetupState(ctx context.Context) (*SetupState, error) {
	var st SetupState
	if _, err := w.store.GetSetting(ctx, setupKey, &st); err != nil {
		return nil, err
	}
	n, err := w.store.CountClients(ctx)
	if err != nil {
		return nil, err
	}
	st.Clients = n
	return &st, nil
}

// MarkSetup stamps completedAt or skippedAt and keeps the other value.
func (w *Workspace) MarkSetup(ctx context.Context, completed bool) (*SetupState, error) {
	st, err := w.SetupState(ctx)
	if err != nil {
		return nil, err
	}
	now := w.clock.Now()
	if completed {
		st.CompletedAt = &now
	} else {
		st.SkippedAt = &now
	}
	if err := w.store.PutSetting(ctx, setupKey, st); err != nil {
		return nil, err
	}
	return st, nil
}
