package jsonapi

import (
	"errors"
)

// NullableAttr is a generic type, which implements a field that can be one of three states:
//
// - field is not set in the request
// - field is explicitly set to `null` in the request
// - field is explicitly set to a valid value in the request
//
// NullableAttr is intended to be used with JSON marshalling and unmarshalling.
// This is generally useful for PATCH requests, where attributes with zero
// values are intentionally not marshaled into the request payload so that
// existing attribute values are not overwritten.
//
// Internal implementation details:
//
// - map[true]T means a value was provided
// - map[false]T means an explicit null was provided
// - nil or zero map means the field was not provided
//
// If the field is expected to be optional, add the `omitempty` JSON tags. Do NOT use `*NullableAttr`!
//
// Adapted from https://www.jvt.me/posts/2024/01/09/go-json-nullable/
type NullableAttr[T any] map[bool]T

// NullableRelationship is a generic type, which implements a field that can be one of three states:
//
// - relationship is not set in the request
// - relationship is explicitly set to `null` in the request
// - relationship is explicitly set to a valid relationship value in the request
//
// NullableRelationship is intended to be used with JSON marshalling and unmarshalling.
// This is generally useful for PATCH requests, where relationships with zero
// values are intentionally not marshaled into the request payload so that
// existing attribute values are not overwritten.
//
// Internal implementation details:
//
// - map[true]T means a value was provided
// - map[false]T means an explicit null was provided
// - nil or zero map means the field was not provided
//
// If the relationship is expected to be optional, add the `omitempty` JSON tags. Do NOT use `*NullableRelationship`!
//
// Slice types are not currently supported for NullableRelationships as the nullable nature can be expressed via empty array
// `polyrelation` JSON tags are NOT currently supported.
//
// NullableRelationships must have an inner type of pointer:
//
// - NullableRelationship[*Comment] - valid
// - NullableRelationship[[]*Comment] - invalid
// - NullableRelationship[Comment] - invalid
type NullableRelationship[T any] map[bool]T

// NewNullableAttrWithValue is a convenience helper to allow constructing a
// NullableAttr with a given value, for instance to construct a field inside a
// struct without introducing an intermediate variable.
func NewNullableAttrWithValue[T any](t T) NullableAttr[T] {
	var n NullableAttr[T]
	n.Set(t)
	return n
}

// NewNullNullableAttr is a convenience helper to allow constructing a NullableAttr with
// an explicit `null`, for instance to construct a field inside a struct
// without introducing an intermediate variable
func NewNullNullableAttr[T any]() NullableAttr[T] {
	var n NullableAttr[T]
	n.SetNull()
	return n
}

// Get retrieves the underlying value, if present, and returns an error if the value was not present
func (t NullableAttr[T]) Get() (T, error) {
	var empty T
	if t.IsNull() {
		return empty, errors.New("value is null")
	}
	if !t.IsSpecified() {
		return empty, errors.New("value is not specified")
	}
	return t[true], nil
}

// Set sets the underlying value to a given value
func (t *NullableAttr[T]) Set(value T) {
	*t = map[bool]T{true: value}
}

// Set sets the underlying value to a given value
func (t *NullableAttr[T]) SetInterface(value interface{}) {
	t.Set(value.(T))
}

// IsNull indicate whether the field was sent, and had a value of `null`
func (t NullableAttr[T]) IsNull() bool {
	_, foundNull := t[false]
	return foundNull
}

// SetNull sets the value to an explicit `null`
func (t *NullableAttr[T]) SetNull() {
	var empty T
	*t = map[bool]T{false: empty}
}

// IsSpecified indicates whether the field was sent
func (t NullableAttr[T]) IsSpecified() bool {
	return len(t) != 0
}

// SetUnspecified sets the value to be absent from the serialized payload
func (t *NullableAttr[T]) SetUnspecified() {
	*t = map[bool]T{}
}

// NewNullableRelationshipWithValue is a convenience helper to allow constructing a
// NullableRelationship with a given value, for instance to construct a field inside a
// struct without introducing an intermediate variable.
func NewNullableRelationshipWithValue[T any](t T) NullableRelationship[T] {
	var n NullableRelationship[T]
	n.Set(t)
	return n
}

// NewNullNullableRelationship is a convenience helper to allow constructing a NullableRelationship with
// an explicit `null`, for instance to construct a field inside a struct
// without introducing an intermediate variable
func NewNullNullableRelationship[T any]() NullableRelationship[T] {
	var n NullableRelationship[T]
	n.SetNull()
	return n
}

// Get retrieves the underlying value, if present, and returns an error if the value was not present
func (t NullableRelationship[T]) Get() (T, error) {
	var empty T
	if t.IsNull() {
		return empty, errors.New("value is null")
	}
	if !t.IsSpecified() {
		return empty, errors.New("value is not specified")
	}
	return t[true], nil
}

// Set sets the underlying value to a given value
func (t *NullableRelationship[T]) Set(value T) {
	*t = map[bool]T{true: value}
}

// SetInterface sets the underlying value from an empty interface,
// performing a type assertion to T.
func (t *NullableRelationship[T]) SetInterface(value interface{}) {
	t.Set(value.(T))
}

// IsNull indicates whether the field was sent, and had a value of `null`
func (t NullableRelationship[T]) IsNull() bool {
	_, foundNull := t[false]
	return foundNull
}

// SetNull sets the value to an explicit `null`
func (t *NullableRelationship[T]) SetNull() {
	var empty T
	*t = map[bool]T{false: empty}
}

// IsSpecified indicates whether the field was sent
func (t NullableRelationship[T]) IsSpecified() bool {
	return len(t) != 0
}

// SetUnspecified sets the value to be absent from the serialized payload
func (t *NullableRelationship[T]) SetUnspecified() {
	*t = map[bool]T{}
}
