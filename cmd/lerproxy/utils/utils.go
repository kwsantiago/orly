package utils

import "strings"

// GetKeys returns a slice containing all the keys from the provided map.
//
// # Parameters
//
// - m (map[string]string): The input map from which to extract keys.
//
// # Return Values
//
// - []string: A slice of strings representing the keys in the map.
//
// # Expected behaviour
//
// - Iterates over each key in the map and appends it to a new slice.
//
// - Returns the slice containing all the keys.
func GetKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// SingleJoiningSlash joins two strings with a single slash between them,
// ensuring that the resulting path doesn't contain multiple consecutive
// slashes.
//
// # Parameters
//
// - a (string): The first string to join.
//
// - b (string): The second string to join.
//
// # Return Values
//
// - result (string): The joined string with a single slash between them if
// needed.
//
// # Expected behaviour
//
// - If both a and b start and end with a slash, the resulting string will have
// only one slash between them.
//
// - If neither a nor b starts or ends with a slash, the strings will be joined
// with a single slash in between.
//
// - Otherwise, the two strings are simply concatenated.
func SingleJoiningSlash(a, b string) string {
	suffixSlash := strings.HasSuffix(a, "/")
	prefixSlash := strings.HasPrefix(b, "/")
	switch {
	case suffixSlash && prefixSlash:
		return a + b[1:]
	case !suffixSlash && !prefixSlash:
		return a + "/" + b
	}
	return a + b
}
