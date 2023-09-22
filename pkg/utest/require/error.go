package require

import "knative.dev/hack/pkg/utest/assert"

// NoError asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.NoError(t, err) {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoError(t TestingT, err error, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NoError(t, err, msgAndArgs...) {
		return
	}
	t.FailNow()
}
