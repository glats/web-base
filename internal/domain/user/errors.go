package user

import "fmt"

type InvalidError struct {
	Field string
	Value any
	err   error
}

func (e InvalidError) Error() string {
	return fmt.Sprintf("invalid %s: %v", e.Field, e.Value)
}

func (e InvalidError) Code() string {
	return "INVALID_ERROR"
}

func (e InvalidError) Unwrap() error { return e.err }

type NotFoundError struct {
	Field string
	Value any
	err   error
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %v", e.Field, e.Value)
}

func (e NotFoundError) Code() string {
	return "NOT_FOUND_ERROR"
}

func (e NotFoundError) Unwrap() error { return e.err }

type AlreadyExistsError struct {
	Field string
	Value any
	err   error
}

func (e AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists: %v", e.Field, e.Value)
}

func (e AlreadyExistsError) Code() string {
	return "ALREADY_EXISTS_ERROR"
}

func (e AlreadyExistsError) Unwrap() error { return e.err }
