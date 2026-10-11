package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
)

type purgeRepo struct {
	domain.Repository
	calls atomic.Int32
}

func (r *purgeRepo) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	r.calls.Add(1)
	return 3, nil
}

func TestRunSessionCleanupPurgesImmediatelyThenPeriodicallyUntilCancelled(t *testing.T) {
	repo := &purgeRepo{}
	a := &App{Identity: usecase.New(repo, "test-secret", time.Minute, time.Hour)}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		a.RunSessionCleanup(ctx, 10*time.Millisecond)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for repo.calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("expected an immediate purge plus periodic ones, got %d calls", repo.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunSessionCleanup must return once ctx is cancelled")
	}
}
