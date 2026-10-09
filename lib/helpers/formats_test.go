package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "—"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{102400, "100 KB"},
		{1048576, "1.0 MB"},
		{1047552, "1023 KB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{5 * 1099511627776, "5.0 TB"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, FormatSize(c.in), "FormatSize(%d)", c.in)
	}
}
