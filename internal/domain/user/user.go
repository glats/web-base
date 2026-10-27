package user

import (
	"net/mail"
	"strings"
)

type User struct {
	id    string
	name  string
	email string
}

type userAttributes func(*User)

func New(attrs ...userAttributes) User {
	u := User{}
	for _, attr := range attrs {
		attr(&u)
	}
	return u
}

func WithID(id string) userAttributes {
	return func(u *User) { u.id = id }
}

func WithName(name string) userAttributes {
	return func(u *User) { u.name = name }
}

func WithEmail(email string) userAttributes {
	return func(u *User) { u.email = email }
}

func (u User) ID() string {
	return u.id
}

func (u User) Name() string {
	return u.name
}

func (u User) Email() string {
	return u.email
}

func (u User) Validate() error {
	if strings.TrimSpace(u.id) == "" {
		return InvalidError{Field: "id", Value: u.id}
	}
	if strings.TrimSpace(u.name) == "" {
		return InvalidError{Field: "name", Value: u.name}
	}
	address, err := mail.ParseAddress(u.email)
	if err != nil {
		return InvalidError{Field: "email", Value: u.email, err: err}
	}
	if address.Address != u.email {
		return InvalidError{Field: "email", Value: u.email}
	}
	return nil
}
