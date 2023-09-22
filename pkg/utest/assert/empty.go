package assert

import (
	"fmt"
	"reflect"
)

// Nil asserts that the specified object is nil.
//
//	assert.Nil(t, err)
func Nil(t TestingT, object any, msgAndArgs ...interface{}) bool {
	if isNil(object) {
		return true
	}
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Fail(t, fmt.Sprintf("Expected nil, but got: %#v", object), msgAndArgs...)
}

// isNil checks if a specified object is nil or not, without Failing.
func isNil(object any) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	isNilableKind := containsKind(
		[]reflect.Kind{
			reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.Slice},
		kind)

	if isNilableKind && value.IsNil() {
		return true
	}

	return false
}

// containsKind checks if a specified kind in the slice of kinds.
func containsKind(kinds []reflect.Kind, kind reflect.Kind) bool {
	for i := 0; i < len(kinds); i++ {
		if kind == kinds[i] {
			return true
		}
	}

	return false
}
