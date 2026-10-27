package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"app.ddd/internal/app/command"
	"app.ddd/internal/app/mock"
	"app.ddd/internal/app/query"
	"app.ddd/internal/domain/user"
)

func TestNewAppWiresEveryHandler(t *testing.T) {
	application := NewApp(mock.NewUserRepository(t))

	assert.NotNil(t, application.Command.CreateUser)
	assert.NotNil(t, application.Command.UpdateUser)
	assert.NotNil(t, application.Command.DeleteUser)
	assert.NotNil(t, application.Query.ObtainUser)
	assert.NotNil(t, application.Query.ListUsers)
}

func TestNewAppHandlersUseTheGivenRepository(t *testing.T) {
	ctx := context.Background()
	ada := user.New(user.WithID("u1"), user.WithName("Ada"), user.WithEmail("ada@example.com"))
	repo := mock.NewUserRepository(t)
	repo.EXPECT().FindByID(ctx, "u1").Return(ada, nil).Times(2)
	repo.EXPECT().Delete(ctx, "u1").Return(nil).Once()
	repo.EXPECT().List(ctx).Return([]user.User{ada}, nil).Once()

	application := NewApp(repo)

	got, err := application.Query.ObtainUser.Handle(ctx, query.ObtainUser{ID: "u1"})
	assert.NoError(t, err)
	assert.Equal(t, ada, got)

	users, err := application.Query.ListUsers.Handle(ctx, query.ListUsers{})
	assert.NoError(t, err)
	assert.Equal(t, []user.User{ada}, users)

	assert.NoError(t, application.Command.DeleteUser.Handle(ctx, command.DeleteUser{ID: "u1"}))
}
