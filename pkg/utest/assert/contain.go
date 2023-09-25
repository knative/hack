package assert

import (
	"fmt"
	"strings"

	"knative.dev/hack/pkg/constraints"
)

// Contains asserts that the specified list(array, slice...) contains the
// specified substring or element.
//
//	assert.Contains(t, ["Hello", "World"], "World")
func Contains[O constraints.Ordered](t TestingT, haystack []O, needle O, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	for _, el := range haystack {
		if el == needle {
			return true
		}
	}
	return Fail(t, fmt.Sprintf("%#v does not contain %#v", haystack, needle), msgAndArgs...)
}

// ContainsSubstring asserts that the specified string contains the specified
// substring.
//
//	assert.ContainsSubstring(t, "Hello World", "World")
func ContainsSubstring(t TestingT, haystack, needle string, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	if !strings.Contains(haystack, needle) {
		return Fail(t, fmt.Sprintf("%#v does not contain substring %#v", haystack, needle), msgAndArgs...)
	}

	return true
}
