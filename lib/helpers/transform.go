package helpers

import (
	"strconv"
	"strings"
)

func StringToInt[T int | int8 | int16 | int32 | int64](v string) T {
	i, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return T(i)
}

func StringToIntPtr[T int | int8 | int16 | int32 | int64](v string) *T {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	n := StringToInt[T](v)
	return &n
}

func StringToBool(v string) bool {
	b, _ := strconv.ParseBool(strings.TrimSpace(v))
	return b
}

func StringToFloat(v string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f
}
