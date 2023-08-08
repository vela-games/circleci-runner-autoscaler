package workers

import (
	"context"
	"log"
	"time"

	"golang.org/x/sync/errgroup"
)

type WorkerDispatcher struct {
	RunEvery time.Duration
	Group    *errgroup.Group
}

func (w *WorkerDispatcher) Start(ctx context.Context, worker Worker) {
	w.Group.Go(func() error {
		for {
			worker.Handle(ctx)
			select {
			case <-time.After(w.RunEvery):
				continue
			case <-ctx.Done():
				log.Printf("exiting %T", worker)
				return nil
			}
		}
	})
}
