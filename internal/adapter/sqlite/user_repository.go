package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob"
	bobsqlite "github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/dm"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/dialect/sqlite/um"

	"app.ddd/internal/adapter"
	"app.ddd/internal/adapter/sqlite/dberrors"
	"app.ddd/internal/adapter/sqlite/models"
	"app.ddd/internal/domain/user"
)

type UserRepository struct {
	db bob.Executor
}

var _ adapter.UserRepository = (*UserRepository)(nil)

func NewUserRepository(db bob.Executor) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u user.User) error {
	setter := &models.UserSetter{
		ID:    omit.From(u.ID()),
		Name:  omit.From(u.Name()),
		Email: omit.From(u.Email()),
	}

	if _, err := models.Users.Insert(setter).One(ctx, r.db); err != nil {
		return fmt.Errorf("create user: %w", translateUniqueViolation(err, u))
	}

	return nil
}

func (r *UserRepository) Update(ctx context.Context, u user.User) error {
	setter := models.UserSetter{
		Name:  omit.From(u.Name()),
		Email: omit.From(u.Email()),
	}

	_, err := models.Users.Update(setter.UpdateMod(), um.Where(models.Users.Columns.ID.EQ(bobsqlite.Arg(u.ID())))).One(ctx, r.db)
	if errors.Is(err, sql.ErrNoRows) {
		return user.NotFoundError{Field: "id", Value: u.ID()}
	}
	if err != nil {
		return fmt.Errorf("update user: %w", translateUniqueViolation(err, u))
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	deleted, err := models.Users.Delete(dm.Where(models.Users.Columns.ID.EQ(bobsqlite.Arg(id)))).Exec(ctx, r.db)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if deleted == 0 {
		return user.NotFoundError{Field: "id", Value: id}
	}

	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (user.User, error) {
	model, err := models.Users.Query(sm.Where(models.Users.Columns.ID.EQ(bobsqlite.Arg(id)))).One(ctx, r.db)
	if errors.Is(err, sql.ErrNoRows) {
		return user.User{}, user.NotFoundError{Field: "id", Value: id}
	}
	if err != nil {
		return user.User{}, fmt.Errorf("find user by id: %w", err)
	}

	return toDomain(model), nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (user.User, error) {
	model, err := models.Users.Query(sm.Where(models.Users.Columns.Email.EQ(bobsqlite.Arg(email)))).One(ctx, r.db)
	if errors.Is(err, sql.ErrNoRows) {
		return user.User{}, user.NotFoundError{Field: "email", Value: email}
	}
	if err != nil {
		return user.User{}, fmt.Errorf("find user by email: %w", err)
	}

	return toDomain(model), nil
}

func (r *UserRepository) List(ctx context.Context) ([]user.User, error) {
	rows, err := models.Users.Query(sm.OrderBy(models.Users.Columns.Name), sm.OrderBy(models.Users.Columns.ID)).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, toDomain(row))
	}

	return users, nil
}

func toDomain(model *models.User) user.User {
	return user.New(user.WithID(model.ID), user.WithName(model.Name), user.WithEmail(model.Email))
}

func translateUniqueViolation(err error, u user.User) error {
	switch {
	case dberrors.UserErrors.ErrUniquePkMainUsers.Is(err):
		return user.AlreadyExistsError{Field: "id", Value: u.ID()}
	case dberrors.UserErrors.ErrUniqueSqliteAutoindexUsers2.Is(err):
		return user.AlreadyExistsError{Field: "email", Value: u.Email()}
	default:
		return err
	}
}
