// Package qtpaths resolves the app's platform locations with QStandardPaths.
// Only main imports it; call Resolve after QCoreApplication_SetApplicationName("rapid")
// so the locations match the application name.
package qtpaths

import (
	qt "github.com/mappu/miqt/qt6"

	"rapid/services/settings"
)

// Resolve returns the Qt-resolved platform locations for the app.
func Resolve() settings.Paths {
	return settings.Paths{
		AppData:   qt.QStandardPaths_WritableLocation(qt.QStandardPaths__AppDataLocation),
		Downloads: qt.QStandardPaths_WritableLocation(qt.QStandardPaths__DownloadLocation),
	}
}
