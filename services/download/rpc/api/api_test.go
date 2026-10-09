package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToDownloadVerifiedLengthAbsentStaysNil(t *testing.T) {
	dl := Aria2Status{GID: "g1", Status: "active"}.ToDownload()
	require.Nil(t, dl.VerifiedLength, "absent verifiedLength must be nil, not a pointer to 0")

	dl = Aria2Status{GID: "g1", Status: "active", VerifiedLength: "1234"}.ToDownload()
	require.NotNil(t, dl.VerifiedLength)
	require.Equal(t, int64(1234), *dl.VerifiedLength)
}
