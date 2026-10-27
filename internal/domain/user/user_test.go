package user

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAndGetters(t *testing.T) {
	u := New(WithID("u1"), WithName("Ada Lovelace"), WithEmail("ada@example.com"))

	assert.Equal(t, "u1", u.ID())
	assert.Equal(t, "Ada Lovelace", u.Name())
	assert.Equal(t, "ada@example.com", u.Email())
	assert.NoError(t, u.Validate())
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		user      User
		wantField string
		wantValue string
	}{
		{name: "empty id", user: New(WithName("Ada"), WithEmail("ada@example.com")), wantField: "id", wantValue: ""},
		{name: "blank id", user: New(WithID("   "), WithName("Ada"), WithEmail("ada@example.com")), wantField: "id", wantValue: "   "},
		{name: "empty name", user: New(WithID("u1"), WithEmail("ada@example.com")), wantField: "name", wantValue: ""},
		{name: "blank name", user: New(WithID("u1"), WithName("   "), WithEmail("ada@example.com")), wantField: "name", wantValue: "   "},
		{name: "empty email", user: New(WithID("u1"), WithName("Ada")), wantField: "email", wantValue: ""},
		{name: "email without domain", user: New(WithID("u1"), WithName("Ada"), WithEmail("ada")), wantField: "email", wantValue: "ada"},
		{name: "email with display name", user: New(WithID("u1"), WithName("Ada"), WithEmail("Ada <ada@example.com>")), wantField: "email", wantValue: "Ada <ada@example.com>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()

			var invalid InvalidError
			assert.ErrorAs(t, err, &invalid)
			assert.Equal(t, tt.wantField, invalid.Field)
			assert.Equal(t, tt.wantValue, invalid.Value)
		})
	}
}

func TestValidateEmailKeepsParseCause(t *testing.T) {
	err := New(WithID("u1"), WithName("Ada"), WithEmail("ada")).Validate()

	assert.NotNil(t, errors.Unwrap(err))
}

func TestErrors(t *testing.T) {
	cause := errors.New("root cause")

	tests := []struct {
		name    string
		err     error
		message string
		code    string
	}{
		{name: "invalid", err: InvalidError{Field: "email", Value: "x", err: cause}, message: "invalid email: x", code: "INVALID_ERROR"},
		{name: "not found", err: NotFoundError{Field: "id", Value: "u1"}, message: "id not found: u1", code: "NOT_FOUND_ERROR"},
		{name: "already exists", err: AlreadyExistsError{Field: "email", Value: "a@b.co"}, message: "email already exists: a@b.co", code: "ALREADY_EXISTS_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.message, tt.err.Error())

			var coded interface{ Code() string }
			assert.ErrorAs(t, tt.err, &coded)
			assert.Equal(t, tt.code, coded.Code())
		})
	}

	assert.ErrorIs(t, InvalidError{err: cause}, cause)
	assert.ErrorIs(t, NotFoundError{err: cause}, cause)
	assert.ErrorIs(t, AlreadyExistsError{err: cause}, cause)
}
