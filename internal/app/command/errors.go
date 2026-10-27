package command

import (
	"errors"

	"app.ddd/internal/domain/user"
)

func isNotFound(err error) bool {
	var notFound user.NotFoundError
	return errors.As(err, &notFound)
}
