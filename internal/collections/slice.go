// Package collections provides generic collection utilities.
package collections

// Concat concatenates multiple slices of the same type into a single slice.
// It preserves the order of elements from the input slices.
func Concat[T any](slices ...[]T) []T {
	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}

	result := make([]T, 0, totalLen)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}
