package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPragmasApplied(t *testing.T) {
	require.NoError(t, OpenMemory(t.Context()))
	t.Cleanup(func() { Close() })

	var foreignKeys int
	require.NoError(t, DB().QueryRowContext(t.Context(), `PRAGMA foreign_keys`).Scan(&foreignKeys))
	require.Equal(t, 1, foreignKeys, "foreign_keys pragma must be on")

	var busyTimeout int
	require.NoError(t, DB().QueryRowContext(t.Context(), `PRAGMA busy_timeout`).Scan(&busyTimeout))
	require.Equal(t, 5000, busyTimeout, "busy_timeout pragma must be applied")
}
