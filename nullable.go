package jsonapi

import (
	"errors"
	"reflect"
	"time"
)

var supportedNullableTypes = map[string]reflect.Value{
	"bool":      reflect.ValueOf(false),
	"time.Time": reflect.ValueOf(time.Time{}),
}

// Nullable is a generic type, which implements a field that can be one of three states:
//
// - field is not set in the request
// - field is explicitly set to `null` in the request
// - field is explicitly set to a valid value in the request
//
// Nullable is intended to be used with JSON marshalling and unmarshalling.
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
// If the field is expected to be optional, add the `omitempty` JSON tags. Do NOT use `*Nullable`!
//
// Adapted from https://www.jvt.me/posts/2024/01/09/go-json-nullable/

type Nullable[T any] map[bool]T

// NewNullableWithValue is a convenience helper to allow constructing a
// Nullable with a given value, for instance to construct a field inside a
// struct without introducing an intermediate variable.
func NewNullableWithValue[T any](t T) Nullable[T] {
	var n Nullable[T]
	n.Set(t)
	return n
}

// NewNullNullable is a convenience helper to allow constructing a Nullable with
// an explicit `null`, for instance to construct a field inside a struct
// without introducing an intermediate variable
func NewNullNullable[T any]() Nullable[T] {
	var n Nullable[T]
	n.SetNull()
	return n
}

// Get retrieves the underlying value, if present, and returns an error if the value was not present
func (t Nullable[T]) Get() (T, error) {
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
func (t *Nullable[T]) Set(value T) {
	*t = map[bool]T{true: value}
}

// Set sets the underlying value to a given value
func (t *Nullable[T]) SetInterface(value interface{}) {
	t.Set(value.(T))
}

// IsNull indicate whether the field was sent, and had a value of `null`
func (t Nullable[T]) IsNull() bool {
	_, foundNull := t[false]
	return foundNull
}

// SetNull indicate that the field was sent, and had a value of `null`
func (t *Nullable[T]) SetNull() {
	var empty T
	*t = map[bool]T{false: empty}
}

// IsSpecified indicates whether the field was sent
func (t Nullable[T]) IsSpecified() bool {
	return len(t) != 0
}

// SetUnspecified indicate whether the field was sent
func (t *Nullable[T]) SetUnspecified() {
	*t = map[bool]T{}
}

// func (t Nullable[T]) MarshalJSON() ([]byte, error) {
// 	// if field was specified, and `null`, marshal it
// 	if t.IsNull() {
// 		return []byte("null"), nil
// 	}
//
// 	// if field was unspecified, and `omitempty` is set on the field's tags,
// 	// `json.Marshal` will omit this field
//
// 	// if the value is of type time.Time, format it as an RFC3339 string.
// 	v := reflect.ValueOf(t[true])
// 	if v.Type() == reflect.TypeOf(new(time.Time)) {
// 		return json.Marshal(v.Elem().Interface().(time.Time).Format(time.RFC3339))
// 	}
//
// 	// we have a value, so marshal it
// 	return json.Marshal(t[true])
// }

// func (t *Nullable[T]) UnmarshalJSON(data []byte) error {
// 	// If field is unspecified, UnmarshalJSON won't be called.
//
// 	// If field is specified, and `null`
// 	if bytes.Equal(data, []byte("null")) {
// 		t.SetNull()
// 		return nil
// 	}
//
// 	// Otherwise, we have an actual value, so parse it
// 	var v T
// 	if err := json.Unmarshal(data, &v); err != nil {
// 		return err
// 	}
//
// 	t.Set(v)
//
// 	return nil
// }

func NullableBool(v bool) Nullable[bool] {
	return NewNullableWithValue[bool](v)
}

func NullBool() Nullable[bool] {
	return NewNullNullable[bool]()
}

func NullableTime(v time.Time) Nullable[time.Time] {
	return NewNullableWithValue[time.Time](v)
}

func NullTime() Nullable[time.Time] {
	return NewNullNullable[time.Time]()
}
