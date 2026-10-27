package command

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"app.ddd/internal/app/mock"
	"app.ddd/internal/domain/user"
)

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(repo *mock.UserRepository)
		wantErr error
	}{
		{
			name: "deletes an existing user",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
				repo.EXPECT().Delete(backgroundCtx, "u1").Return(nil).Once()
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
			name: "propagates a repository failure on delete",
			setup: func(repo *mock.UserRepository) {
				repo.EXPECT().FindByID(backgroundCtx, "u1").Return(ada, nil).Once()
				repo.EXPECT().Delete(backgroundCtx, "u1").Return(errBoom).Once()
			},
			wantErr: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mock.NewUserRepository(t)
			tt.setup(repo)

			err := NewDeleteUserHandler(repo, noMetricClient).Handle(backgroundCtx, deleteUserCmd)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
