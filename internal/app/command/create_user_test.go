package command

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
	errBoom        = errors.New("boom")
	ada            = user.New(user.WithID("u1"), user.WithName("Ada"), user.WithEmail("ada@example.com"))
	otherUser      = user.New(user.WithID("u2"), user.WithName("Grace"), user.WithEmail("ada@example.com"))
	idNotFound     = user.NotFoundError{Field: "id", Value: "u1"}
	emailNotFound  = user.NotFoundError{Field: "email", Value: "ada@example.com"}
	idExists       = user.AlreadyExistsError{Field: "id", Value: "u1"}
	emailExists    = user.AlreadyExistsError{Field: "email", Value: "ada@example.com"}
	createUserCmd  = CreateUser{ID: "u1", Name: "Ada", Email: "ada@example.com"}
	updateUserCmd  = UpdateUser{ID: "u1", Name: "Ada", Email: "ada@example.com"}
	deleteUserCmd  = DeleteUser{ID: "u1"}
	backgroundCtx  = context.Background()
	noMetricClient = metric.NoOp{}
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(repo *mock.UserRepository)
		wantErr error
	}{
		{
			name: "creates a new user",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(user.User{}, idNotFound).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(user.User{}, emailNotFound).Once()
				repo.EXPECT().Create(backgroundCtx, ada).Return(nil).Once()
			},
		},
		{
			name: "rejects an existing id",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
			},
			wantErr: idExists,
		},
		{
			name: "rejects an email used by another user",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(user.User{}, idNotFound).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(otherUser, nil).Once()
			},
			wantErr: emailExists,
		},
		{
			name: "propagates a repository failure on find by id",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(user.User{}, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name: "propagates a repository failure on find by email",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(user.User{}, idNotFound).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(user.User{}, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name: "propagates a repository failure on create",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(user.User{}, idNotFound).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(user.User{}, emailNotFound).Once()
				repo.EXPECT().Create(backgroundCtx, ada).Return(errBoom).Once()
			},
			wantErr: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mock.NewUserRepository(t)
			tt.setup(repo)

			err := NewCreateUserHandler(repo, noMetricClient).Handle(backgroundCtx, createUserCmd)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCreateUserRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name      string
		cmd       CreateUser
		wantField string
	}{
		{name: "empty id", cmd: CreateUser{Name: "Ada", Email: "ada@example.com"}, wantField: "id"},
		{name: "empty name", cmd: CreateUser{ID: "u1", Email: "ada@example.com"}, wantField: "name"},
		{name: "malformed email", cmd: CreateUser{ID: "u1", Name: "Ada", Email: "ada"}, wantField: "email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mock.NewUserRepository(t)

			err := NewCreateUserHandler(repo, noMetricClient).Handle(backgroundCtx, tt.cmd)

			var invalid user.InvalidError
			assert.ErrorAs(t, err, &invalid)
			assert.Equal(t, tt.wantField, invalid.Field)
		})
	}
}
