package command

import (
	"context"

	"app.ddd/internal/adapter"
	"app.ddd/internal/common/decorator"
	"app.ddd/internal/domain/user"
)

type UpdateUser struct {
	ID    string
	Name  string
	Email string
}

type UpdateUserHandler decorator.CommandHandler[UpdateUser]

type updateUserCommand struct {
	repository adapter.UserRepository
}

func NewUpdateUserHandler(repository adapter.UserRepository, metric decorator.MetricsClient) UpdateUserHandler {
	return decorator.ApplyCommandDecorators[UpdateUser](updateUserCommand{repository: repository}, metric)
}

func (c updateUserCommand) Handle(ctx context.Context, cmd UpdateUser) error {
	u := user.New(user.WithID(cmd.ID), user.WithName(cmd.Name), user.WithEmail(cmd.Email))
	if err := u.Validate(); err != nil {
		return err
	}

	if _, err := c.repository.FindByID(ctx, u.ID()); err != nil {
		return err
	}

	owner, err := c.repository.FindByEmail(ctx, u.Email())
	if err == nil && owner.ID() != u.ID() {
		return user.AlreadyExistsError{Field: "email", Value: u.Email()}
	}
	if err != nil && !isNotFound(err) {
		return err
	}

	return c.repository.Update(ctx, u)
}
