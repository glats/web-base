package query

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"app.ddd/internal/adapter"
	"app.ddd/internal/common/decorator"
	"app.ddd/internal/domain/user"
)

type ListUsers struct{}

type ListUsersHandler decorator.QueryHandler[ListUsers, []user.User]

type listUsersQuery struct {
	repository adapter.UserRepository
}

func NewListUsersHandler(repository adapter.UserRepository, metric decorator.MetricsClient) ListUsersHandler {
	return decorator.ApplyQueryDecorators[ListUsers, []user.User](listUsersQuery{repository: repository}, metric)
}

func (q listUsersQuery) Handle(ctx context.Context, _ ListUsers) ([]user.User, error) {
	users, err := q.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	sorted := slices.Clone(users)
	slices.SortStableFunc(sorted, func(a, b user.User) int {
		return cmp.Or(strings.Compare(a.Name(), b.Name()), strings.Compare(a.ID(), b.ID()))
	})

	return sorted, nil
}
