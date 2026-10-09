package helpers

import "fmt"

func FormatSize(n int64) string {
	if n <= 0 {
		return "—"
	}
	units := [...]string{"B", "KB", "MB", "GB", "TB"}
	value := float64(n)
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	digits := 1
	if value >= 100 || i == 0 {
		digits = 0
	}
	return fmt.Sprintf("%.*f %s", digits, value, units[i])
}
