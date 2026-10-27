package decorator

import "context"

type CommandHandler[C any] interface {
	Handle(context.Context, C) error
}

type commandCall[C any] func(context.Context, C) error

func (call commandCall[C]) Handle(ctx context.Context, value C) error {
	return call(ctx, value)
}

func ApplyCommandDecorators[C any](handler CommandHandler[C], metrics MetricsClient) CommandHandler[C] {
	return commandCall[C](func(ctx context.Context, value C) (err error) {
		action := generateActionName(value)
		emitAction(ctx, "command", action, value, false)
		defer func() {
			if err == nil {
				emitAction(ctx, "command", action, value, true)
			}
		}()
		return measure(metrics, "commands", action, func() error {
			return handler.Handle(ctx, value)
		})
	})
}
