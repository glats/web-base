package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"app.ddd/internal/app/mock"
	"app.ddd/internal/common/metric"
	"app.ddd/internal/domain/user"
)

var (
	alan = user.New(user.WithID("u3"), user.WithName("Alan Turing"), user.WithEmail("alan@example.com"))
	twin = user.New(user.WithID("u0"), user.WithName("Grace"), user.WithEmail("twin@example.com"))
)

func TestListUsers(t *testing.T) {
	tests := []struct {
		name    string
		repoOut []user.User
		repoErr error
		want    []user.User
	}{
		{name: "sorts by name and then by id", repoOut: []user.User{grace, alan, twin, ada}, want: []user.User{ada, alan, twin, grace}},
		{name: "keeps an already sorted list", repoOut: []user.User{ada, grace}, want: []user.User{ada, grace}},
		{name: "returns an empty list", repoOut: []user.User{}, want: []user.User{}},
		{name: "propagates a repository failure", repoErr: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mock.NewUserRepository(t)
			repo.EXPECT().List(backgroundCtx).Return(tt.repoOut, tt.repoErr).Once()

			got, err := NewListUsersHandler(repo, metric.NoOp{}).Handle(backgroundCtx, ListUsers{})

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.repoErr, err)
		})
	}
}

func TestListUsersDoesNotReorderTheSliceOwnedByTheRepository(t *testing.T) {
	fromRepository := []user.User{grace, ada}
	repo := mock.NewUserRepository(t)
	repo.EXPECT().List(backgroundCtx).Return(fromRepository, nil).Once()

	_, err := NewListUsersHandler(repo, metric.NoOp{}).Handle(backgroundCtx, ListUsers{})

	assert.NoError(t, err)
	assert.Equal(t, []user.User{grace, ada}, fromRepository)
}
