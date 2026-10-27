package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoOpInc(t *testing.T) {
	assert.NotPanics(t, func() { NoOp{}.Inc("any.key", 1) })
}
