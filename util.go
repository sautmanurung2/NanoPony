package nanopony

// GetOrDefault returns val if it is greater than 0, otherwise returns def.
// This generic function works for various numeric types including time.Duration
// (which has int64 as its underlying type).
//
// Example:
//
//	val := GetOrDefault(config.MaxOpenConns, 20)
//	timeout := GetOrDefault(config.Timeout, 5 * time.Second)
func GetOrDefault[T ~int | ~int64 | ~float64](val, def T) T {
	if val > 0 {
		return val
	}
	return def
}
