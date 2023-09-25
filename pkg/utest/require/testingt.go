package require

import "knative.dev/hack/pkg/utest/assert"

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	assert.TestingT
	FailNow()
}

type tHelper interface {
	Helper()
}
