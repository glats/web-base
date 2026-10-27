package decorator

import "context"

type QueryHandler[Q, R any] interface {
	Handle(context.Context, Q) (R, error)
}

type queryCall[Q, R any] func(context.Context, Q) (R, error)

func (call queryCall[Q, R]) Handle(ctx context.Context, value Q) (R, error) {
	return call(ctx, value)
}

func ApplyQueryDecorators[Q, R any](handler QueryHandler[Q, R], metrics MetricsClient) QueryHandler[Q, R] {
	return queryCall[Q, R](func(ctx context.Context, value Q) (result R, err error) {
		action := generateActionName(value)
		emitAction(ctx, "query", action, value, false)
		defer func() {
			if err == nil {
				emitAction(ctx, "query", action, value, true)
			}
		}()
		err = measure(metrics, "querys", action, func() (callErr error) {
			result, callErr = handler.Handle(ctx, value)
			return callErr
		})
		return result, err
	})
}
