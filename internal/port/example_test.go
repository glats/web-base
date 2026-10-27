package port_test

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"app.ddd/internal/adapter"
	"app.ddd/internal/app"
	"app.ddd/internal/app/command"
	"app.ddd/internal/app/query"
	"app.ddd/internal/domain/user"
	"app.ddd/internal/port"
	"app.ddd/internal/service"
)

type memoryUsers struct {
	mu    sync.Mutex
	users map[string]user.User
}

var _ adapter.UserRepository = (*memoryUsers)(nil)

func newMemoryUsers() *memoryUsers {
	return &memoryUsers{users: map[string]user.User{}}
}

func (r *memoryUsers) Create(_ context.Context, u user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID()] = u
	return nil
}

func (r *memoryUsers) Update(ctx context.Context, u user.User) error {
	return r.Create(ctx, u)
}

func (r *memoryUsers) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.users, id)
	return nil
}

func (r *memoryUsers) FindByID(_ context.Context, id string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return user.User{}, user.NotFoundError{Field: "id", Value: id}
}

func (r *memoryUsers) FindByEmail(_ context.Context, email string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email() == email {
			return u, nil
		}
	}
	return user.User{}, user.NotFoundError{Field: "email", Value: email}
}

func (r *memoryUsers) List(_ context.Context) ([]user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	users := make([]user.User, 0, len(r.users))
	for _, u := range r.users {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Name() < users[j].Name() })
	return users, nil
}

type helloJob struct {
	app *app.Application
}

var _ port.Job = helloJob{}

func (j helloJob) Start(ctx context.Context) error {
	err := j.app.Command.CreateUser.Handle(ctx, command.CreateUser{ID: "u1", Name: "Ada Lovelace", Email: "ada@example.com"})
	if err != nil {
		return err
	}

	u, err := j.app.Query.ObtainUser.Handle(ctx, query.ObtainUser{ID: "u1"})
	if err != nil {
		return err
	}

	fmt.Printf("hello, %s!\n", u.Name())
	return nil
}

func Example_helloJob() {
	application := service.NewApp(newMemoryUsers())

	var job port.Job = helloJob{app: application}
	if err := job.Start(context.Background()); err != nil {
		fmt.Println(err)
	}

	// Output: hello, Ada Lovelace!
}
