package adapter

import (
	"context"

	"app.ddd/internal/domain/user"
)

type UserRepository interface {
	Create(ctx context.Context, u user.User) error
	Update(ctx context.Context, u user.User) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (user.User, error)
	FindByEmail(ctx context.Context, email string) (user.User, error)
	List(ctx context.Context) ([]user.User, error)
}
