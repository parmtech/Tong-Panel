package jobs

import (
	"context"
	"errors"
	"time"
)

const (
	developmentNoopWorkerID = "local-dev-noop"
	developmentNoopLease    = 30 * time.Second
	developmentNoopPoll     = 100 * time.Millisecond
)

// ErrDevelopmentNoopWorker is deliberately generic. A lease capability or
// underlying database detail must never be included in worker diagnostics.
var ErrDevelopmentNoopWorker = errors.New("development no-op worker failed")

// RunDevelopmentNoopWorker runs only the metadata lifecycle of validation.noop.
// There is no executor callback, command, path, host adapter or external effect.
// Its store copy uses the worker context without changing concurrent API calls.
func (s *SQLiteStore) RunDevelopmentNoopWorker(ctx context.Context) error {
	if s == nil || s.db == nil || ctx == nil {
		return ErrDevelopmentNoopWorker
	}
	worker := *s
	worker.ctx = ctx
	for {
		if ctx.Err() != nil {
			return nil
		}
		err := worker.processDevelopmentNoop()
		if ctx.Err() != nil {
			return nil
		}
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrNoJobs) && !errors.Is(err, ErrLeaseLost) {
			return ErrDevelopmentNoopWorker
		}
		timer := time.NewTimer(developmentNoopPoll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (s *SQLiteStore) processDevelopmentNoop() error {
	claim, err := s.ClaimNext(developmentNoopWorkerID, developmentNoopLease)
	if err != nil {
		return err
	}
	// Check before Start as well as the authoritative row returned by Start.
	// A future schema extension must not silently turn this into an executor.
	if claim.Job.Kind != KindNoop {
		return ErrDevelopmentNoopWorker
	}
	started, err := s.Start(claim)
	if err != nil {
		return err
	}
	if started.Kind != KindNoop {
		return ErrDevelopmentNoopWorker
	}
	_, err = s.Complete(claim)
	return err
}
