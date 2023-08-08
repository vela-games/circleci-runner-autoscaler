package workers

import (
	"context"
)

// Interface for all workers to implement
type Worker interface {
	Handle(context.Context)
}

type Dispatcher interface {
	Start(context.Context, Worker)
}
