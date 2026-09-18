package components

import (
	"os"
	"testing"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

func TestMain(m *testing.M) {
	if os.Getenv("QT_QPA_PLATFORM") == "" {
		_ = os.Setenv("QT_QPA_PLATFORM", "offscreen")
	}
	qt.NewQApplication([]string{"rapid-ui-tests"})
	theme.Init()
	os.Exit(m.Run())
}
