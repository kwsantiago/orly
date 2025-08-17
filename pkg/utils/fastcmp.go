package utils

func FastEqual(a, b []byte) (same bool) {
	if len(a) != len(b) {
		return
	}
	for i, v := range a {
		if v != b[i] {
			return
		}
	}
	return true
}
