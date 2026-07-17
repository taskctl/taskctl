package collections

// OrEmpty returns s unchanged, or a non-nil empty slice when s is nil, so it
// marshals to a JSON [] rather than null.
func OrEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
