package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app.ddd/internal/adapter"
	"app.ddd/internal/domain/user"
)

var (
	ada   = user.New(user.WithID("u1"), user.WithName("Ada Lovelace"), user.WithEmail("ada@example.com"))
	grace = user.New(user.WithID("u2"), user.WithName("Grace Hopper"), user.WithEmail("grace@example.com"))
	alan  = user.New(user.WithID("u3"), user.WithName("Alan Turing"), user.WithEmail("alan@example.com"))
)

func newRepository(t *testing.T) *UserRepository {
	t.Helper()

	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))

	db, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return NewUserRepository(db)
}

func seed(t *testing.T, repo *UserRepository, users ...user.User) {
	t.Helper()

	for _, u := range users {
		require.NoError(t, repo.Create(context.Background(), u))
	}
}

var _ adapter.UserRepository = (*UserRepository)(nil)

func TestCreateAndFind(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	seed(t, repo, ada)

	byID, err := repo.FindByID(ctx, "u1")
	require.NoError(t, err)
	byEmail, err := repo.FindByEmail(ctx, "ada@example.com")
	require.NoError(t, err)

	assert.Equal(t, ada, byID)
	assert.Equal(t, ada, byEmail)
}

func TestFindReturnsNotFound(t *testing.T) {
	repo := newRepository(t)

	tests := []struct {
		name      string
		find      func() error
		wantField string
		wantValue string
	}{
		{name: "by id", find: func() error { _, err := repo.FindByID(context.Background(), "nope"); return err }, wantField: "id", wantValue: "nope"},
		{name: "by email", find: func() error { _, err := repo.FindByEmail(context.Background(), "no@example.com"); return err }, wantField: "email", wantValue: "no@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var notFound user.NotFoundError
			require.ErrorAs(t, tt.find(), &notFound)

			assert.Equal(t, tt.wantField, notFound.Field)
			assert.Equal(t, tt.wantValue, notFound.Value)
		})
	}
}

func TestCreateRejectsDuplicates(t *testing.T) {
	repo := newRepository(t)
	seed(t, repo, ada)

	tests := []struct {
		name      string
		user      user.User
		wantField string
		wantValue string
	}{
		{name: "same id", user: user.New(user.WithID("u1"), user.WithName("Other"), user.WithEmail("other@example.com")), wantField: "id", wantValue: "u1"},
		{name: "same email", user: user.New(user.WithID("u9"), user.WithName("Other"), user.WithEmail("ada@example.com")), wantField: "email", wantValue: "ada@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exists user.AlreadyExistsError
			require.ErrorAs(t, repo.Create(context.Background(), tt.user), &exists)

			assert.Equal(t, tt.wantField, exists.Field)
			assert.Equal(t, tt.wantValue, exists.Value)
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	seed(t, repo, ada, grace)

	t.Run("changes name and email", func(t *testing.T) {
		changed := user.New(user.WithID("u1"), user.WithName("Ada King"), user.WithEmail("king@example.com"))

		require.NoError(t, repo.Update(ctx, changed))

		got, err := repo.FindByID(ctx, "u1")
		require.NoError(t, err)
		assert.Equal(t, changed, got)
	})

	t.Run("rejects an unknown id", func(t *testing.T) {
		var notFound user.NotFoundError

		require.ErrorAs(t, repo.Update(ctx, user.New(user.WithID("nope"), user.WithName("X"), user.WithEmail("x@example.com"))), &notFound)

		assert.Equal(t, "id", notFound.Field)
		assert.Equal(t, "nope", notFound.Value)
	})

	t.Run("rejects the email of another user", func(t *testing.T) {
		var exists user.AlreadyExistsError

		require.ErrorAs(t, repo.Update(ctx, user.New(user.WithID("u1"), user.WithName("Ada"), user.WithEmail("grace@example.com"))), &exists)

		assert.Equal(t, "email", exists.Field)
		assert.Equal(t, "grace@example.com", exists.Value)
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	seed(t, repo, ada, grace)

	require.NoError(t, repo.Delete(ctx, "u1"))

	_, err := repo.FindByID(ctx, "u1")
	var notFound user.NotFoundError
	assert.ErrorAs(t, err, &notFound)
	_, err = repo.FindByID(ctx, "u2")
	assert.NoError(t, err)

	require.ErrorAs(t, repo.Delete(ctx, "u1"), &notFound)
	assert.Equal(t, "u1", notFound.Value)
}

func TestListIsOrderedByNameThenID(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)

	empty, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)

	twin := user.New(user.WithID("u0"), user.WithName("Grace Hopper"), user.WithEmail("twin@example.com"))
	seed(t, repo, grace, alan, twin, ada)

	got, err := repo.List(ctx)

	require.NoError(t, err)
	assert.Equal(t, []user.User{ada, alan, twin, grace}, got)
}

func TestEveryMethodHonorsTheContext(t *testing.T) {
	repo := newRepository(t)
	seed(t, repo, ada)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		call func() error
	}{
		{name: "create", call: func() error { return repo.Create(ctx, grace) }},
		{name: "update", call: func() error { return repo.Update(ctx, ada) }},
		{name: "delete", call: func() error { return repo.Delete(ctx, "u1") }},
		{name: "find by id", call: func() error { _, err := repo.FindByID(ctx, "u1"); return err }},
		{name: "find by email", call: func() error { _, err := repo.FindByEmail(ctx, "ada@example.com"); return err }},
		{name: "list", call: func() error { _, err := repo.List(ctx); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()

			assert.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestOtherErrorsAreWrappedAndNotMappedToDomainErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))
	db, err := Open(path)
	require.NoError(t, err)
	repo := NewUserRepository(db)
	require.NoError(t, db.Close())
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{name: "create", call: func() error { return repo.Create(ctx, ada) }},
		{name: "update", call: func() error { return repo.Update(ctx, ada) }},
		{name: "delete", call: func() error { return repo.Delete(ctx, "u1") }},
		{name: "find by id", call: func() error { _, err := repo.FindByID(ctx, "u1"); return err }},
		{name: "find by email", call: func() error { _, err := repo.FindByEmail(ctx, "ada@example.com"); return err }},
		{name: "list", call: func() error { _, err := repo.List(ctx); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()

			require.Error(t, err)
			assert.ErrorContains(t, err, "database is closed")
			var notFound user.NotFoundError
			var exists user.AlreadyExistsError
			assert.False(t, errors.As(err, &notFound))
			assert.False(t, errors.As(err, &exists))
		})
	}
}

func TestDataSurvivesClosingAndReopeningTheFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	require.NoError(t, Migrate(path))

	first, err := Open(path)
	require.NoError(t, err)
	seed(t, NewUserRepository(first), ada)
	require.NoError(t, first.Close())

	second, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Close() })

	got, err := NewUserRepository(second).FindByID(ctx, "u1")

	require.NoError(t, err)
	assert.Equal(t, ada, got)
}

func TestConcurrentWritesAreAllStored(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	const writers = 20

	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- repo.Create(ctx, user.New(
				user.WithID(fmt.Sprintf("id-%02d", i)),
				user.WithName(fmt.Sprintf("User %02d", i)),
				user.WithEmail(fmt.Sprintf("user%02d@example.com", i)),
			))
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
	got, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, got, writers)
}
