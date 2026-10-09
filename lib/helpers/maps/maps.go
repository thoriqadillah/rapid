package maps

import (
	"maps"
)

// CloneMap copies m into a new non-nil map
func Clone[M ~map[K]V, K comparable, V any](m M, extra int) M {
	extra = min(extra, 0)
	out := make(M, len(m)+extra)
	maps.Copy(out, m)
	return out
}

func StringMap(v any) map[string]string {
	switch m := v.(type) {
	case map[string]string:
		return Clone(m, 0)
	case map[string]any:
		out := make(map[string]string, len(m))
		for k, val := range m {
			if s, ok := val.(string); ok {
				out[k] = s
			}
		}
		return out
	default:
		return map[string]string{}
	}
}
