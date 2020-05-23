package reltest

import (
	"strings"

	"github.com/Fs02/rel"
	"github.com/stretchr/testify/mock"
)

// Delete asserts and simulate delete function for test.
type Delete struct {
	*Expect
}

// For match expect calls for given record.
func (d *Delete) For(record interface{}) *Delete {
	d.Arguments[0] = record
	return d
}

// ForType match expect calls for given type.
// Type must include package name, example: `model.User`.
func (d *Delete) ForType(typ string) *Delete {
	return d.For(mock.AnythingOfType("*" + strings.TrimPrefix(typ, "*")))
}

// ExpectDelete to be called.
func ExpectDelete(r *Repository, options []rel.Cascade) *Delete {
	return &Delete{
		Expect: newExpect(r, "Delete", []interface{}{mock.Anything, options}, []interface{}{nil}),
	}
}
