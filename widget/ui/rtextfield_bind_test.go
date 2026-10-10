package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/lib/reactive"
)

func TestRTextFieldBindModel(t *testing.T) {
	query := reactive.NewSignal("")
	model := reactive.NewWritableComputed(
		func() string {
			return query.Get()
		},
		func(v string) {
			query.Set(strings.ToLower(strings.TrimSpace(v)))
		},
	)
	defer model.Dispose()

	field := NewRTextField()
	field.Bind(model)

	// User types mixed case: the normalizing setter runs and the echo
	// settles back into the field exactly once.
	field.value.Set("Dune")
	require.Equal(t, "dune", model.Get(), "setter did not normalize")
	require.Equal(t, "dune", field.value.Get(), "echo was not applied back")

	// Model changes push into the field.
	model.Set("interstellar")
	require.Equal(t, "interstellar", field.value.Get(), "model change did not push")
}
