package port

import "context"

type Job interface {
	Start(ctx context.Context) error
}

type Listener[M any] interface {
	Receive(ctx context.Context, msg M) error
}
