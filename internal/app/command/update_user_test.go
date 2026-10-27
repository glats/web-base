package command

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"app.ddd/internal/app/mock"
	"app.ddd/internal/domain/user"
)

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(repo *mock.UserRepository)
		wantErr error
	}{
		{
			name: "updates an existing user",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(user.User{}, emailNotFound).Once()
				repo.EXPECT().Update(backgroundCtx, ada).Return(nil).Once()
			},
		},
		{
			name: "keeps the same email of the same user",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(ada, nil).Once()
				repo.EXPECT().Update(backgroundCtx, ada).Return(nil).Once()
			},
		},
		{
			name: "rejects an unknown id",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(user.User{}, idNotFound).Once()
			},
			wantErr: idNotFound,
		},
		{
			name: "rejects an email used by another user",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
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
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(user.User{}, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name: "propagates a repository failure on update",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
				repo.EXPECT().FindByEmail(backgroundCtx, "ada@example.com").Return(user.User{}, emailNotFound).Once()
				repo.EXPECT().Update(backgroundCtx, ada).Return(errBoom).Once()
			},
			wantErr: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mock.NewUserRepository(t)
			tt.setup(repo)

			err := NewUpdateUserHandler(repo, noMetricClient).Handle(backgroundCtx, updateUserCmd)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestUpdateUserRejectsInvalidData(t *testing.T) {
	repo := mock.NewUserRepository(t)

	err := NewUpdateUserHandler(repo, noMetricClient).Handle(backgroundCtx, UpdateUser{ID: "u1", Name: "Ada", Email: "ada"})

	var invalid user.InvalidError
	assert.ErrorAs(t, err, &invalid)
	assert.Equal(t, "email", invalid.Field)
}
