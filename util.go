package main

import "slices"

func mapAppendUnique[K, V comparable, M ~map[K][]V](m M, k K, v V) {
	values := m[k]
	if !slices.Contains(values, v) {
		m[k] = append(values, v)
	}
}

func appendCapped[T any, S ~[]T](dst S, src S) S {
	if len(dst)+len(src) > cap(dst) {
		src = src[:cap(dst)-len(dst)]
	}
	return append(dst, src...)
}
