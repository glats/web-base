package app

import (
	"app.ddd/internal/app/command"
	"app.ddd/internal/app/query"
	"app.ddd/internal/common/decorator"
	"app.ddd/internal/domain/user"
)

type Application struct {
	Command *Command
	Query   *Query
}

type Command struct {
	CreateUser decorator.CommandHandler[command.CreateUser]
	UpdateUser decorator.CommandHandler[command.UpdateUser]
	DeleteUser decorator.CommandHandler[command.DeleteUser]
}

type Query struct {
	ObtainUser decorator.QueryHandler[query.ObtainUser, user.User]
	ListUsers  decorator.QueryHandler[query.ListUsers, []user.User]
}
