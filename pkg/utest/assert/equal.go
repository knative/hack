package assert

import (
	"fmt"

	"knative.dev/hack/pkg/constraints"
)

// Greater asserts that the first element is greater than the second
//
//	assert.Greater(t, 2, 1)
//	assert.Greater(t, float64(2), float64(1))
//	assert.Greater(t, "b", "a")
func Greater[O constraints.Ordered](t TestingT, e1, e2 O, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if e1 > e2 {
		return true
	}
	return Fail(t, fmt.Sprintf("\"%v\" is not greater than \"%v\"", e1, e2), msgAndArgs...)
}

// Equal asserts that two objects are equal.
//
//	assert.Equal(t, 123, 123)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equal[C comparable](t TestingT, expected, actual C, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if expected != actual {
		return Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %#v\n"+
			"actual  : %#v", expected, actual), msgAndArgs...)
	}

	return true

}
