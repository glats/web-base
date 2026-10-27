package command

import (
	"context"

	"app.ddd/internal/adapter"
	"app.ddd/internal/common/decorator"
)

type DeleteUser struct {
	ID string
}

type DeleteUserHandler decorator.CommandHandler[DeleteUser]

type deleteUserCommand struct {
	repository adapter.UserRepository
}

func NewDeleteUserHandler(repository adapter.UserRepository, metric decorator.MetricsClient) DeleteUserHandler {
	return decorator.ApplyCommandDecorators[DeleteUser](deleteUserCommand{repository: repository}, metric)
}

func (c deleteUserCommand) Handle(ctx context.Context, cmd DeleteUser) error {
	if _, err := c.repository.FindByID(ctx, cmd.ID); err != nil {
		return err
	}

	return c.repository.Delete(ctx, cmd.ID)
}
