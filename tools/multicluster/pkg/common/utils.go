package common

// Contains checks if a string is present in the provided slice.
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// AnyAreEmpty returns true if any of the given strings have the zero value.
func AnyAreEmpty(values ...string) bool {
	for _, v := range values {
		if v == "" {
			return true
		}
	}
	return false
}
