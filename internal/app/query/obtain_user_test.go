package query

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"app.ddd/internal/app/mock"
	"app.ddd/internal/common/metric"
	"app.ddd/internal/domain/user"
)

var (
	errBoom       = errors.New("boom")
	ada           = user.New(user.WithID("u1"), user.WithName("Ada"), user.WithEmail("ada@example.com"))
	grace         = user.New(user.WithID("u2"), user.WithName("Grace"), user.WithEmail("grace@example.com"))
	idNotFound    = user.NotFoundError{Field: "id", Value: "u1"}
	backgroundCtx = context.Background()
)

func TestObtainUser(t *testing.T) {
	tests := []struct {
		name     string
		repoUser user.User
		repoErr  error
		want     user.User
	}{
		{name: "returns the user", repoUser: ada, want: ada},
		{name: "propagates not found", repoErr: idNotFound},
		{name: "propagates a repository failure", repoErr: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mock.NewUserRepository(t)
			repo.EXPECT().FindByID(backgroundCtx, "u1").Return(tt.repoUser, tt.repoErr).Once()

			got, err := NewObtainUserHandler(repo, metric.NoOp{}).Handle(backgroundCtx, ObtainUser{ID: "u1"})

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.repoErr, err)
		})
	}
}
