package query

import (
	"context"

	"app.ddd/internal/adapter"
	"app.ddd/internal/common/decorator"
	"app.ddd/internal/domain/user"
)

type ObtainUser struct {
	ID string
}

type ObtainUserHandler decorator.QueryHandler[ObtainUser, user.User]

type obtainUserQuery struct {
	repository adapter.UserRepository
}

func NewObtainUserHandler(repository adapter.UserRepository, metric decorator.MetricsClient) ObtainUserHandler {
	return decorator.ApplyQueryDecorators[ObtainUser, user.User](obtainUserQuery{repository: repository}, metric)
}

func (q obtainUserQuery) Handle(ctx context.Context, query ObtainUser) (user.User, error) {
	return q.repository.FindByID(ctx, query.ID)
}
