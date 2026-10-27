package command

import (
	"context"

	"app.ddd/internal/adapter"
	"app.ddd/internal/common/decorator"
	"app.ddd/internal/domain/user"
)

type CreateUser struct {
	ID    string
	Name  string
	Email string
}

type CreateUserHandler decorator.CommandHandler[CreateUser]

type createUserCommand struct {
	repository adapter.UserRepository
}

func NewCreateUserHandler(repository adapter.UserRepository, metric decorator.MetricsClient) CreateUserHandler {
	return decorator.ApplyCommandDecorators[CreateUser](createUserCommand{repository: repository}, metric)
}

func (c createUserCommand) Handle(ctx context.Context, cmd CreateUser) error {
	u := user.New(user.WithID(cmd.ID), user.WithName(cmd.Name), user.WithEmail(cmd.Email))
	if err := u.Validate(); err != nil {
		return err
	}

	if _, err := c.repository.FindByID(ctx, u.ID()); err == nil {
		return user.AlreadyExistsError{Field: "id", Value: u.ID()}
	} else if !isNotFound(err) {
		return err
	}

	if _, err := c.repository.FindByEmail(ctx, u.Email()); err == nil {
		return user.AlreadyExistsError{Field: "email", Value: u.Email()}
	} else if !isNotFound(err) {
		return err
	}

	return c.repository.Create(ctx, u)
}
