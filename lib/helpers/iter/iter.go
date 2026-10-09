package iter

func Map[I, O any](s []I, f func(I) O) []O {
	result := make([]O, 0, len(s))
	for _, v := range s {
		result = append(result, f(v))
	}
	return result
}

func ForEach[I any](s []I, f func(I)) {
	for _, v := range s {
		f(v)
	}
}

func Filter[I any](s []I, f func(I) bool) []I {
	result := make([]I, 0, len(s))
	for _, v := range s {
		if f(v) {
			result = append(result, v)
		}
	}
	return result
}

func Reduce[I, O any](s []I, f func(I, O) O, initial O) O {
	result := initial
	for _, v := range s {
		result = f(v, result)
	}
	return result
}
