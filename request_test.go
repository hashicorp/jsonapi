package jsonapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestUnmarshall_attrStringSlice(t *testing.T) {
	out := &Book{}
	tags := []string{"fiction", "sale"}
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "books",
			"id":         "1",
			"attributes": map[string]interface{}{"tags": tags},
		},
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if err := UnmarshalPayload(bytes.NewReader(b), out); err != nil {
		t.Fatal(err)
	}

	if e, a := len(tags), len(out.Tags); e != a {
		t.Fatalf("Was expecting %d tags, got %d", e, a)
	}

	sort.Strings(tags)
	sort.Strings(out.Tags)

	for i, tag := range tags {
		if e, a := tag, out.Tags[i]; e != a {
			t.Fatalf("At index %d, was expecting %s got %s", i, e, a)
		}
	}
}

func TestUnmarshall_attrInterface(t *testing.T) {
	tests := []struct {
		genericData interface{}
		expected    reflect.Kind
	}{
		{
			genericData: "foo",
			expected:    reflect.String,
		},
		{
			genericData: true,
			expected:    reflect.Bool,
		},
		{
			genericData: float64(5),
			expected:    reflect.Float64,
		},
		{
			genericData: []string{"foo", "bar"},
			expected:    reflect.Slice,
		},
		{
			genericData: map[string]string{
				"foo": "bar",
			},
			expected: reflect.Map,
		},
	}

	for _, tc := range tests {
		out := &GenericInterface{}
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "generic",
				"id":   "1",
				"attributes": map[string]interface{}{
					"interface": tc.genericData,
				},
			},
		}
		b, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}

		if err := UnmarshalPayload(bytes.NewReader(b), out); err != nil {
			t.Fatal(err)
		}
		if reflect.TypeOf(out.Data).Kind() != tc.expected {
			t.Fatalf("Expected %v to match interface %v", out.Data, tc.expected)
		}
	}
}

func TestUnmarshalToStructWithPointerAttr(t *testing.T) {
	out := new(WithPointer)
	in := map[string]interface{}{
		"name":      "The name",
		"is-active": true,
		"int-val":   8,
		"float-val": 1.1,
	}
	if err := UnmarshalPayload(sampleWithPointerPayload(in), out); err != nil {
		t.Fatal(err)
	}
	if *out.Name != "The name" {
		t.Fatalf("Error unmarshalling to string ptr")
	}
	if !*out.IsActive {
		t.Fatalf("Error unmarshalling to bool ptr")
	}
	if *out.IntVal != 8 {
		t.Fatalf("Error unmarshalling to int ptr")
	}
	if *out.FloatVal != 1.1 {
		t.Fatalf("Error unmarshalling to float ptr")
	}
}

func TestUnmarshalPayload_missingTypeFieldShouldError(t *testing.T) {
	if err := UnmarshalPayload(
		strings.NewReader(`{"data":{"body":"hello world"}}`),
		&Post{},
	); err == nil {
		t.Fatalf("Expected an error but did not get one")
	}
}

func TestUnmarshalPayload_ptrsAllNil(t *testing.T) {
	out := new(WithPointer)
	if err := UnmarshalPayload(
		strings.NewReader(`{"data":{"type":"with-pointers"}}`), out); err != nil {
		t.Fatalf("Error unmarshalling to Foo: %v", err)
	}

	if out.ID != nil {
		t.Fatalf("Error unmarshalling; expected ID ptr to be nil")
	}
}

func TestUnmarshalPayloadWithPointerID(t *testing.T) {
	out := new(WithPointer)
	attrs := map[string]interface{}{}

	if err := UnmarshalPayload(sampleWithPointerPayload(attrs), out); err != nil {
		t.Fatalf("Error unmarshalling to Foo")
	}

	// these were present in the payload -- expect val to be not nil
	if out.ID == nil {
		t.Fatalf("Error unmarshalling; expected ID ptr to be not nil")
	}
	if e, a := uint64(2), *out.ID; e != a {
		t.Fatalf("Was expecting the ID to have a value of %d, got %d", e, a)
	}
}

func TestUnmarshalPayloadWithPointerAttr_AbsentVal(t *testing.T) {
	out := new(WithPointer)
	in := map[string]interface{}{
		"name":      "The name",
		"is-active": true,
	}

	if err := UnmarshalPayload(sampleWithPointerPayload(in), out); err != nil {
		t.Fatalf("Error unmarshalling to Foo")
	}

	// these were present in the payload -- expect val to be not nil
	if out.Name == nil || out.IsActive == nil {
		t.Fatalf("Error unmarshalling; expected ptr to be not nil")
	}

	// these were absent in the payload -- expect val to be nil
	if out.IntVal != nil || out.FloatVal != nil {
		t.Fatalf("Error unmarshalling; expected ptr to be nil")
	}
}

func TestUnmarshalToStructWithPointerAttr_BadType_bool(t *testing.T) {
	out := new(WithPointer)
	in := map[string]interface{}{
		"name": true, // This is the wrong type.
	}
	expectedErrorMessage := "jsonapi: Can't unmarshal true (bool) to struct field `Name`, which is a pointer to `string`"

	err := UnmarshalPayload(sampleWithPointerPayload(in), out)

	if err == nil {
		t.Fatalf("Expected error due to invalid type.")
	}
	if err.Error() != expectedErrorMessage {
		t.Fatalf("Unexpected error message: %s", err.Error())
	}
	if _, ok := err.(ErrUnsupportedPtrType); !ok {
		t.Fatalf("Unexpected error type: %s", reflect.TypeOf(err))
	}
}

func TestUnmarshalToStructWithPointerAttr_BadType_MapPtr(t *testing.T) {
	out := new(WithPointer)
	in := map[string]interface{}{
		"name": &map[string]interface{}{"a": 5}, // This is the wrong type.
	}
	expectedErrorMessage := "jsonapi: Can't unmarshal map[a:5] (map) to struct field `Name`, which is a pointer to `string`"

	err := UnmarshalPayload(sampleWithPointerPayload(in), out)

	if err == nil {
		t.Fatalf("Expected error due to invalid type.")
	}
	if err.Error() != expectedErrorMessage {
		t.Fatalf("Unexpected error message: %s", err.Error())
	}
	if _, ok := err.(ErrUnsupportedPtrType); !ok {
		t.Fatalf("Unexpected error type: %s", reflect.TypeOf(err))
	}
}

func TestUnmarshalToStructWithPointerAttr_BadType_Struct(t *testing.T) {
	out := new(WithPointer)
	type FooStruct struct{ A int }
	in := map[string]interface{}{
		"name": FooStruct{A: 5}, // This is the wrong type.
	}
	expectedErrorMessage := "jsonapi: Can't unmarshal map[A:5] (map) to struct field `Name`, which is a pointer to `string`"

	err := UnmarshalPayload(sampleWithPointerPayload(in), out)

	if err == nil {
		t.Fatalf("Expected error due to invalid type.")
	}
	if err.Error() != expectedErrorMessage {
		t.Fatalf("Unexpected error message: %s", err.Error())
	}
	if _, ok := err.(ErrUnsupportedPtrType); !ok {
		t.Fatalf("Unexpected error type: %s", reflect.TypeOf(err))
	}
}

func TestUnmarshalToStructWithPointerAttr_BadType_IntSlice(t *testing.T) {
	out := new(WithPointer)
	in := map[string]interface{}{
		"name": []int{4, 5}, // This is the wrong type.
	}
	expectedErrorMessage := "jsonapi: Can't unmarshal [4 5] (slice) to struct field `Name`, which is a pointer to `string`"

	err := UnmarshalPayload(sampleWithPointerPayload(in), out)

	if err == nil {
		t.Fatalf("Expected error due to invalid type.")
	}
	if err.Error() != expectedErrorMessage {
		t.Fatalf("Unexpected error message: %s", err.Error())
	}
	if _, ok := err.(ErrUnsupportedPtrType); !ok {
		t.Fatalf("Unexpected error type: %s", reflect.TypeOf(err))
	}
}

func TestStringPointerField(t *testing.T) {
	// Build Book payload
	description := "Hello World!"
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "books",
			"id":   "5",
			"attributes": map[string]interface{}{
				"author":      "aren55555",
				"description": description,
				"isbn":        "",
			},
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	// Parse JSON API payload
	book := new(Book)
	if err := UnmarshalPayload(bytes.NewReader(payload), book); err != nil {
		t.Fatal(err)
	}

	if book.Description == nil {
		t.Fatal("Was not expecting a nil pointer for book.Description")
	}
	if expected, actual := description, *book.Description; expected != actual {
		t.Fatalf("Was expecting descript to be `%s`, got `%s`", expected, actual)
	}
}

func TestUnmarshalNullableTime(t *testing.T) {
	aTime := time.Date(2016, 8, 17, 8, 27, 12, 23849, time.UTC)

	out := new(WithNullableAttrs)

	attrs := map[string]interface{}{
		"name":         "Name",
		"int_time":     aTime.Unix(),
		"rfc3339_time": aTime.Format(time.RFC3339),
		"iso8601_time": aTime.Format(iso8601TimeFormat),
	}

	if err := UnmarshalPayload(samplePayloadWithNullableAttrs(attrs), out); err != nil {
		t.Fatal(err)
	}

	if out.IntTime == nil {
		t.Fatal("Was not expecting a nil pointer for out.IntTime")
	}

	timeVal, err := out.IntTime.Get()
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := aTime, timeVal; expected.Equal(actual) {
		t.Fatalf("Was expecting int_time to be `%s`, got `%s`", expected, actual)
	}

	timeVal, err = out.IntTime.Get()
	if err != nil {
		t.Fatal(err)
	}

	if out.RFC3339Time == nil {
		t.Fatal("Was not expecting a nil pointer for out.RFC3339Time")
	}
	if expected, actual := aTime, timeVal; expected.Equal(actual) {
		t.Fatalf("Was expecting descript to be `%s`, got `%s`", expected, actual)
	}

	timeVal, err = out.IntTime.Get()
	if err != nil {
		t.Fatal(err)
	}

	if out.ISO8601Time == nil {
		t.Fatal("Was not expecting a nil pointer for out.ISO8601Time")
	}
	if expected, actual := aTime, timeVal; expected.Equal(actual) {
		t.Fatalf("Was expecting descript to be `%s`, got `%s`", expected, actual)
	}
}

func TestUnmarshalNullableBool(t *testing.T) {
	out := new(WithNullableAttrs)

	aBool := false

	attrs := map[string]interface{}{
		"name": "Name",
		"bool": aBool,
	}

	if err := UnmarshalPayload(samplePayloadWithNullableAttrs(attrs), out); err != nil {
		t.Fatal(err)
	}

	if out.Bool == nil {
		t.Fatal("Was not expecting a nil pointer for out.Bool")
	}

	boolVal, err := out.Bool.Get()
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := aBool, boolVal; expected != actual {
		t.Fatalf("Was expecting bool to be `%t`, got `%t`", expected, actual)
	}
}

func TestUnmarshalNullableRelationshipsNonNullValue(t *testing.T) {
	comment := &Comment{
		ID:   5,
		Body: "Hello World",
	}

	payload := &OnePayload{
		Data: &Node{
			ID:   "10",
			Type: "with-nullables",
			Relationships: map[string]interface{}{
				"nullable_comment": &RelationshipOneNode{
					Data: &Node{
						Type: "comments",
						ID:   strconv.Itoa(comment.ID),
					},
				},
			},
		},
	}

	outBuf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(outBuf).Encode(payload); err != nil {
		t.Fatal(err)
	}

	out := new(WithNullableAttrs)

	if err := UnmarshalPayload(outBuf, out); err != nil {
		t.Fatal(err)
	}

	nullableCommentOpt := out.NullableComment
	if !nullableCommentOpt.IsSpecified() {
		t.Fatal("Expected NullableComment to be specified")
	}

	nullableComment, err := nullableCommentOpt.Get()
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := comment.ID, nullableComment.ID; expected != actual {
		t.Fatalf("Was expecting NullableComment to be `%d`, got `%d`", expected, actual)
	}
}

func TestUnmarshalNullableRelationshipsExplicitNullValue(t *testing.T) {
	payload := &OnePayload{
		Data: &Node{
			ID:   "10",
			Type: "with-nullables",
			Relationships: map[string]interface{}{
				"nullable_comment": &RelationshipOneNode{
					Data: nil,
				},
			},
		},
	}

	outBuf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(outBuf).Encode(payload); err != nil {
		t.Fatal(err)
	}

	out := new(WithNullableAttrs)

	if err := UnmarshalPayload(outBuf, out); err != nil {
		t.Fatal(err)
	}

	nullableCommentOpt := out.NullableComment
	if !nullableCommentOpt.IsSpecified() || !nullableCommentOpt.IsNull() {
		t.Fatal("Expected NullableComment to be specified and explicit null")
	}

}

func TestUnmarshalNullableRelationshipsNonExistentValue(t *testing.T) {
	payload := &OnePayload{
		Data: &Node{
			ID:            "10",
			Type:          "with-nullables",
			Relationships: map[string]interface{}{},
		},
	}

	outBuf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(outBuf).Encode(payload); err != nil {
		t.Fatal(err)
	}

	out := new(WithNullableAttrs)

	if err := UnmarshalPayload(outBuf, out); err != nil {
		t.Fatal(err)
	}

	nullableCommentOpt := out.NullableComment
	if nullableCommentOpt.IsSpecified() || nullableCommentOpt.IsNull() {
		t.Fatal("Expected NullableComment to NOT be specified and NOT be explicit null")
	}
}

func TestUnmarshalNullableRelationshipsNoRelationships(t *testing.T) {
	payload := &OnePayload{
		Data: &Node{
			ID:   "10",
			Type: "with-nullables",
		},
	}

	outBuf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(outBuf).Encode(payload); err != nil {
		t.Fatal(err)
	}

	out := new(WithNullableAttrs)

	if err := UnmarshalPayload(outBuf, out); err != nil {
		t.Fatal(err)
	}

	nullableCommentOpt := out.NullableComment
	if nullableCommentOpt.IsSpecified() || nullableCommentOpt.IsNull() {
		t.Fatal("Expected NullableComment to NOT be specified and NOT be explicit null")
	}
}

func TestMalformedTag(t *testing.T) {
	out := new(BadModel)
	err := UnmarshalPayload(samplePayload(), out)
	if err == nil || err != ErrBadJSONAPIStructTag {
		t.Fatalf("Did not error out with wrong number of arguments in tag")
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	in := strings.NewReader("{}")
	out := new(Blog)

	err := UnmarshalPayload(in, out)

	if err == nil {
		t.Fatalf("Did not error out the invalid JSON.")
	}
}

func TestUnmarshalInvalidJSON_BadType(t *testing.T) {
	var badTypeTests = []struct {
		Field    string
		BadValue interface{}
		Error    error
	}{ // The `Field` values here correspond to the `ModelBadTypes` jsonapi fields.
		{Field: "string_field", BadValue: 0, Error: ErrUnknownFieldNumberType},  // Expected string.
		{Field: "float_field", BadValue: "A string.", Error: ErrInvalidType},    // Expected float64.
		{Field: "time_field", BadValue: "A string.", Error: ErrInvalidTime},     // Expected int64.
		{Field: "time_ptr_field", BadValue: "A string.", Error: ErrInvalidTime}, // Expected *time / int64.
	}
	for _, test := range badTypeTests {
		t.Run(fmt.Sprintf("Test_%s", test.Field), func(t *testing.T) {
			out := new(ModelBadTypes)
			in := map[string]interface{}{}
			in[test.Field] = test.BadValue
			expectedErrorMessage := test.Error.Error()

			err := UnmarshalPayload(samplePayloadWithBadTypes(in), out)

			if err == nil {
				t.Fatalf("Expected error due to invalid type.")
			}
			if err.Error() != expectedErrorMessage {
				t.Fatalf("Unexpected error message: %s", err.Error())
			}
		})
	}
}

func TestUnmarshalSetsID(t *testing.T) {
	in := samplePayloadWithID()
	out := new(Blog)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.ID != 2 {
		t.Fatalf("Did not set ID on dst interface")
	}
}

func TestUnmarshal_nonNumericID(t *testing.T) {
	data := samplePayloadWithoutIncluded()
	data["data"].(map[string]interface{})["id"] = "non-numeric-id"
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(payload)
	out := new(Post)

	if err := UnmarshalPayload(in, out); err != ErrBadJSONAPIID {
		t.Fatalf(
			"Was expecting a `%s` error, got `%s`",
			ErrBadJSONAPIID,
			err,
		)
	}
}

func TestUnmarshalSetsAttrs(t *testing.T) {
	out, err := unmarshalSamplePayload()
	if err != nil {
		t.Fatal(err)
	}

	if out.CreatedAt.IsZero() {
		t.Fatalf("Did not parse time")
	}

	if out.ViewCount != 1000 {
		t.Fatalf("View count not properly serialized")
	}
}

func TestUnmarshal_Times(t *testing.T) {
	aTime := time.Date(2016, 8, 17, 8, 27, 12, 0, time.UTC)

	for _, tc := range []struct {
		desc         string
		inputPayload *OnePayload
		wantErr      bool
		verification func(tm *TimestampModel) error
	}{
		// Default:
		{
			desc: "default_byValue",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"defaultv": aTime.Unix(),
					},
				},
			},
			verification: func(tm *TimestampModel) error {
				if !tm.DefaultV.Equal(aTime) {
					return errors.New("times not equal!")
				}
				return nil
			},
		},
		{
			desc: "default_byPointer",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"defaultp": aTime.Unix(),
					},
				},
			},
			verification: func(tm *TimestampModel) error {
				if !tm.DefaultP.Equal(aTime) {
					return errors.New("times not equal!")
				}
				return nil
			},
		},
		{
			desc: "default_invalid",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"defaultv": "not a timestamp!",
					},
				},
			},
			wantErr: true,
		},
		// ISO 8601:
		{
			desc: "iso8601_byValue",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"iso8601v": "2016-08-17T08:27:12Z",
					},
				},
			},
			verification: func(tm *TimestampModel) error {
				if !tm.ISO8601V.Equal(aTime) {
					return errors.New("times not equal!")
				}
				return nil
			},
		},
		{
			desc: "iso8601_byPointer",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"iso8601p": "2016-08-17T08:27:12Z",
					},
				},
			},
			verification: func(tm *TimestampModel) error {
				if !tm.ISO8601P.Equal(aTime) {
					return errors.New("times not equal!")
				}
				return nil
			},
		},
		{
			desc: "iso8601_invalid",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"iso8601v": "not a timestamp",
					},
				},
			},
			wantErr: true,
		},
		// RFC 3339
		{
			desc: "rfc3339_byValue",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"rfc3339v": "2016-08-17T08:27:12Z",
					},
				},
			},
			verification: func(tm *TimestampModel) error {
				if got, want := tm.RFC3339V, aTime; got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			desc: "rfc3339_byPointer",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"rfc3339p": "2016-08-17T08:27:12Z",
					},
				},
			},
			verification: func(tm *TimestampModel) error {
				if got, want := *tm.RFC3339P, aTime; got != want {
					return fmt.Errorf("got %v, want %v", got, want)
				}
				return nil
			},
		},
		{
			desc: "rfc3339_invalid",
			inputPayload: &OnePayload{
				Data: &Node{
					Type: "timestamps",
					Attributes: map[string]interface{}{
						"rfc3339v": "not a timestamp",
					},
				},
			},
			wantErr: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			// Serialize the OnePayload using the standard JSON library.
			in := bytes.NewBuffer(nil)
			if err := json.NewEncoder(in).Encode(tc.inputPayload); err != nil {
				t.Fatal(err)
			}

			out := &TimestampModel{}
			err := UnmarshalPayload(in, out)
			if got, want := (err != nil), tc.wantErr; got != want {
				t.Fatalf("UnmarshalPayload error: got %v, want %v", got, want)
			}
			if tc.verification != nil {
				if err := tc.verification(out); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestUnmarshalRelationshipsWithoutIncluded(t *testing.T) {
	data, err := json.Marshal(samplePayloadWithoutIncluded())
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)
	out := new(Post)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	// Verify each comment has at least an ID
	for _, comment := range out.Comments {
		if comment.ID == 0 {
			t.Fatalf("The comment did not have an ID")
		}
	}
}

func TestUnmarshalRelationships(t *testing.T) {
	out, err := unmarshalSamplePayload()
	if err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Title != "Bas" || out.CurrentPost.Body != "Fuubar" {
		t.Fatalf("Attributes were not set")
	}

	if len(out.Posts) != 2 {
		t.Fatalf("There should have been 2 posts")
	}
}

func Test_UnmarshalPayload_polymorphicRelations(t *testing.T) {
	in := bytes.NewReader([]byte(`{
		"data": {
			"type": "blogs",
			"id":   "3",
			"attributes": {
				"title": "Hello, World"
			},
			"relationships": {
				"hero-media": {
					"data": {
						"type": "videos",
						"id":   "1"
					}
				},
				"media": {
					"data": [
						{
							"type": "images",
							"id":   "1"
						},
						{
							"type": "videos",
							"id":   "2"
						}
					]
				}
			}
		},
		"included": [
			{
				"type": "videos",
				"id": "1",
				"attributes": {
					"captions": "It's Awesome!"
				}
			},
			{
				"type": "images",
				"id":   "1",
				"attributes": {
					"src": "/media/clear1x1.gif"
				}
			},
			{
				"type": "videos",
				"id":   "2",
				"attributes": {
					"captions": "Oh, I didn't see you there"
				}
			}
		]
	}`))
	out := new(BlogPostWithPoly)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Title != "Hello, World" {
		t.Errorf("expected Title %q but got %q", "Hello, World", out.Title)
	}

	if out.Hero.Image != nil {
		t.Errorf("expected Hero image to be nil but got %+v", out.Hero.Image)
	}

	if out.Hero.Video == nil || out.Hero.Video.Captions != "It's Awesome!" {
		t.Errorf("expected Hero to be the expected video relation but got %+v", out.Hero.Video)
	}

	// Unmarshals included records
	if out.Media[0].Image == nil || out.Media[0].Image.Src != "/media/clear1x1.gif" {
		t.Errorf("expected Media 0 to be the expected image relation but got %+v", out.Media[0].Image)
	}

	if out.Media[1].Video == nil || out.Media[1].Video.Captions != "Oh, I didn't see you there" {
		t.Errorf("expected Media 1 to be the expected video relation but got %+v", out.Media[1].Video)
	}
}

func Test_UnmarshalPayload_polymorphicRelations_no_choice(t *testing.T) {
	type pointerToOne struct {
		ID    string      `jsonapi:"primary,blogs"`
		Title string      `jsonapi:"attr,title"`
		Hero  *OneOfMedia `jsonapi:"polyrelation,hero-media,omitempty"`
	}

	in := bytes.NewReader([]byte(`{
		"data": {
			"type": "blogs",
			"id":   "3",
			"attributes": {
				"title": "Hello, World"
			},
			"relationships": {
				"hero-media": {
					"data": {
						"type": "absolutely-not",
						"id":   "1",
						"attributes": {
							"captions": "It's Awesome!"
						}
					}
				}
			}
		}
	}`))
	out := new(pointerToOne)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Title != "Hello, World" {
		t.Errorf("expected Title %q but got %q", "Hello, World", out.Title)
	}

	if out.Hero == nil {
		t.Fatal("expected Hero to not be nil")
	}

	if out.Hero.Image != nil || out.Hero.Video != nil {
		t.Fatal("expected both Hero fields to be nil")
	}
}

func Test_UnmarshalPayload_polymorphicRelations_omitted(t *testing.T) {
	type pointerToOne struct {
		ID    string      `jsonapi:"primary,blogs"`
		Title string      `jsonapi:"attr,title"`
		Hero  *OneOfMedia `jsonapi:"polyrelation,hero-media"`
	}

	in := bytes.NewReader([]byte(`{
		"data": {
			"type": "blogs",
			"id":   "3",
			"attributes": {
				"title": "Hello, World"
			}
		}
	}`))
	out := new(pointerToOne)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Title != "Hello, World" {
		t.Errorf("expected Title %q but got %q", "Hello, World", out.Title)
	}

	if out.Hero != nil {
		t.Fatalf("expected Hero to be nil, but got %+v", out.Hero)
	}
}

func Test_UnmarshalPayload_polymorphicRelations_deprecatedRelation(t *testing.T) {
	type withDeprecatedRelation struct {
		ID    string      `jsonapi:"primary,blogs"`
		Title string      `jsonapi:"attr,title"`
		Media *OneOfMedia `jsonapi:"polyrelation,media"`
		Image *Image      `jsonapi:"relation,media"` // Deprecated
	}

	in := bytes.NewReader([]byte(`{
		"data": [{
			"type": "blogs",
			"id":   "3",
			"attributes": {
				"title": "Hello, World"
			},
			"relationships": {
				"media": {
					"data": {
						"type": "videos",
						"id":   "123"
					}
				}
			}
		}]
	}`))

	model := reflect.TypeOf(new(withDeprecatedRelation))

	out, err := UnmarshalManyPayload(in, model)
	if err != nil {
		t.Fatal(err)
	}

	result := out[0].(*withDeprecatedRelation)

	if result.Title != "Hello, World" {
		t.Errorf("expected Title %q but got %q", "Hello, World", result.Title)
	}

	if result.Media.Video.ID != "123" {
		t.Fatalf("expected Video to be \"123\", but got %+v", result.Media.Video)
	}
}

func Test_choiceStructMapping(t *testing.T) {
	cases := []struct {
		val reflect.Type
	}{
		{val: reflect.TypeOf(&OneOfMedia{})},
		{val: reflect.TypeOf([]*OneOfMedia{{}})},
	}

	for _, c := range cases {
		result := choiceStructMapping(c.val)
		imageField, ok := result["images"]
		if !ok || imageField.FieldNum != 0 {
			t.Errorf("expected \"images\" to be the first field, but got %d", imageField.FieldNum)
		}
		videoField, ok := result["videos"]
		if !ok || videoField.FieldNum != 1 {
			t.Errorf("expected \"videos\" to be the third field, but got %d", videoField.FieldNum)
		}
	}
}

func TestUnmarshalNullRelationship(t *testing.T) {
	sample := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "posts",
			"id":   "1",
			"attributes": map[string]interface{}{
				"body":  "Hello",
				"title": "World",
			},
			"relationships": map[string]interface{}{
				"latest_comment": map[string]interface{}{
					"data": nil, // empty to-one relationship
				},
			},
		},
	}
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}

	in := bytes.NewReader(data)
	out := new(Post)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.LatestComment != nil {
		t.Fatalf("Latest Comment was not set to nil")
	}
}

func TestUnmarshalNullRelationshipInSlice(t *testing.T) {
	sample := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "posts",
			"id":   "1",
			"attributes": map[string]interface{}{
				"body":  "Hello",
				"title": "World",
			},
			"relationships": map[string]interface{}{
				"comments": map[string]interface{}{
					"data": []interface{}{}, // empty to-many relationships
				},
			},
		},
	}
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}

	in := bytes.NewReader(data)
	out := new(Post)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if len(out.Comments) != 0 {
		t.Fatalf("Wrong number of comments; Comments should be empty")
	}
}

func TestUnmarshalNestedRelationships(t *testing.T) {
	out, err := unmarshalSamplePayload()
	if err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Comments == nil {
		t.Fatalf("Did not materialize nested records, comments")
	}

	if len(out.CurrentPost.Comments) != 2 {
		t.Fatalf("Wrong number of comments")
	}
}

func TestUnmarshalRelationshipsSerializedEmbedded(t *testing.T) {
	out, err := sampleSerializedEmbeddedTestModel()
	if err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Title != "Foo" || out.CurrentPost.Body != "Bar" {
		t.Fatalf("Attributes were not set")
	}

	if len(out.Posts) != 2 {
		t.Fatalf("There should have been 2 posts")
	}

	if out.Posts[0].LatestComment.Body != "foo" {
		t.Fatalf("The comment body was not set")
	}
}

func TestUnmarshalNestedRelationshipsEmbedded(t *testing.T) {
	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayloadEmbedded(out, testModel()); err != nil {
		t.Fatal(err)
	}

	model := new(Blog)

	if err := UnmarshalPayload(out, model); err != nil {
		t.Fatal(err)
	}

	if model.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if model.CurrentPost.Comments == nil {
		t.Fatalf("Did not materialize nested records, comments")
	}

	if len(model.CurrentPost.Comments) != 2 {
		t.Fatalf("Wrong number of comments")
	}

	if model.CurrentPost.Comments[0].Body != "foo" {
		t.Fatalf("Comment body not set")
	}
}

func TestUnmarshalRelationshipsSideloaded(t *testing.T) {
	payload, err := samplePayloadWithSideloaded()
	if err != nil {
		t.Fatal(err)
	}
	out := new(Blog)

	if err := UnmarshalPayload(payload, out); err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Title != "Foo" || out.CurrentPost.Body != "Bar" {
		t.Fatalf("Attributes were not set")
	}

	if len(out.Posts) != 2 {
		t.Fatalf("There should have been 2 posts")
	}
}

func TestUnmarshalNestedRelationshipsSideloaded(t *testing.T) {
	payload, err := samplePayloadWithSideloaded()
	if err != nil {
		t.Fatal(err)
	}
	out := new(Blog)

	if err := UnmarshalPayload(payload, out); err != nil {
		t.Fatal(err)
	}

	if out.CurrentPost == nil {
		t.Fatalf("Current post was not materialized")
	}

	if out.CurrentPost.Comments == nil {
		t.Fatalf("Did not materialize nested records, comments")
	}

	if len(out.CurrentPost.Comments) != 2 {
		t.Fatalf("Wrong number of comments")
	}

	if out.CurrentPost.Comments[0].Body != "foo" {
		t.Fatalf("Comment body not set")
	}
}

func TestUnmarshalNestedRelationshipsEmbedded_withClientIDs(t *testing.T) {
	model := new(Blog)

	if err := UnmarshalPayload(samplePayload(), model); err != nil {
		t.Fatal(err)
	}

	if model.Posts[0].ClientID == "" {
		t.Fatalf("ClientID not set from request on related record")
	}
}

func TestUnmarshalLinks(t *testing.T) {
	model := new(Blog)

	if err := UnmarshalPayload(samplePayload(), model); err != nil {
		t.Fatal(err)
	}

	if model.Links == nil {
		t.Fatalf("Expected Links field on model to be set")
	}

	if e, a := "http://somesite.com/blogs/1", model.Links[KeySelfLink]; e != a {
		t.Fatalf("Was expecting links.%s to have a value of %s, got %s", KeySelfLink, e, a)
	}

	if e, a := "http://somesite.com/posts/1", model.Posts[0].Links[KeySelfLink]; e != a {
		t.Fatalf("Was expecting posts.0.links.%s to have a value of %s, got %s", KeySelfLink, e, a)
	}

	expectedLinkObject := Link{Href: "http://somesite.com/posts/2", Meta: Meta{"foo": "bar"}}
	if e, a := expectedLinkObject, model.CurrentPost.Links[KeySelfLink]; !reflect.DeepEqual(e, a) {
		t.Fatalf("Was expecting posts.0.links.%s to have a value of %s, got %s", KeySelfLink, e, a)
	}

	if e, a := "http://somesite.com/comments/1", model.CurrentPost.Comments[0].Links[KeySelfLink]; e != a {
		t.Fatalf("Was expecting posts.0.links.%s to have a value of %s, got %s", KeySelfLink, e, a)
	}
}

func unmarshalSamplePayload() (*Blog, error) {
	in := samplePayload()
	out := new(Blog)

	if err := UnmarshalPayload(in, out); err != nil {
		return nil, err
	}

	return out, nil
}

func TestUnmarshalManyPayload(t *testing.T) {
	sample := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"type": "posts",
				"id":   "1",
				"attributes": map[string]interface{}{
					"body":  "First",
					"title": "Post",
				},
			},
			map[string]interface{}{
				"type": "posts",
				"id":   "2",
				"attributes": map[string]interface{}{
					"body":  "Second",
					"title": "Post",
				},
			},
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)

	posts, err := UnmarshalManyPayload(in, reflect.TypeOf(new(Post)))
	if err != nil {
		t.Fatal(err)
	}

	if len(posts) != 2 {
		t.Fatal("Wrong number of posts")
	}

	for _, p := range posts {
		_, ok := p.(*Post)
		if !ok {
			t.Fatal("Was expecting a Post")
		}
	}
}

func TestOnePayload_withLinks(t *testing.T) {
	rawJSON := []byte("{\"data\": { \"type\": \"posts\", \"id\": \"1\", \"attributes\": { \"body\": \"First\", \"title\": \"Post\" } }, \"links\": { \"self\": \"http://somesite.com/posts/1\" } }")

	in := bytes.NewReader(rawJSON)

	payload := new(OnePayload)
	if err := json.NewDecoder(in).Decode(payload); err != nil {
		t.Fatal(err)
	}

	if payload.Links == nil {
		t.Fatal("Was expecting a non nil ptr Link field")
	}

	links := *payload.Links

	self, ok := links[KeySelfLink]
	if !ok {
		t.Fatal("Was expecting a non nil 'self' link field")
	}
	if e, a := "http://somesite.com/posts/1", self; e != a {
		t.Fatalf("Was expecting links.%s to have a value of %s, got %s", KeySelfLink, e, a)
	}

}

func TestManyPayload_withLinks(t *testing.T) {
	firstPageURL := "http://somesite.com/movies?page[limit]=50&page[offset]=50"
	prevPageURL := "http://somesite.com/movies?page[limit]=50&page[offset]=0"
	nextPageURL := "http://somesite.com/movies?page[limit]=50&page[offset]=100"
	lastPageURL := "http://somesite.com/movies?page[limit]=50&page[offset]=500"

	sample := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"type": "posts",
				"id":   "1",
				"attributes": map[string]interface{}{
					"body":  "First",
					"title": "Post",
				},
			},
			map[string]interface{}{
				"type": "posts",
				"id":   "2",
				"attributes": map[string]interface{}{
					"body":  "Second",
					"title": "Post",
				},
			},
		},
		"links": map[string]interface{}{
			KeyFirstPage:    firstPageURL,
			KeyPreviousPage: prevPageURL,
			KeyNextPage:     nextPageURL,
			KeyLastPage:     lastPageURL,
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)

	payload := new(ManyPayload)
	if err = json.NewDecoder(in).Decode(payload); err != nil {
		t.Fatal(err)
	}

	if payload.Links == nil {
		t.Fatal("Was expecting a non nil ptr Link field")
	}

	links := *payload.Links

	first, ok := links[KeyFirstPage]
	if !ok {
		t.Fatal("Was expecting a non nil ptr Link field")
	}
	if e, a := firstPageURL, first; e != a {
		t.Fatalf("Was expecting links.%s to have a value of %s, got %s", KeyFirstPage, e, a)
	}

	prev, ok := links[KeyPreviousPage]
	if !ok {
		t.Fatal("Was expecting a non nil ptr Link field")
	}
	if e, a := prevPageURL, prev; e != a {
		t.Fatalf("Was expecting links.%s to have a value of %s, got %s", KeyPreviousPage, e, a)
	}

	next, ok := links[KeyNextPage]
	if !ok {
		t.Fatal("Was expecting a non nil ptr Link field")
	}
	if e, a := nextPageURL, next; e != a {
		t.Fatalf("Was expecting links.%s to have a value of %s, got %s", KeyNextPage, e, a)
	}

	last, ok := links[KeyLastPage]
	if !ok {
		t.Fatal("Was expecting a non nil ptr Link field")
	}
	if e, a := lastPageURL, last; e != a {
		t.Fatalf("Was expecting links.%s to have a value of %s, got %s", KeyLastPage, e, a)
	}
}

func TestUnmarshalCustomTypeAttributes(t *testing.T) {
	customInt := CustomIntType(5)
	customFloat := CustomFloatType(1.5)
	customString := CustomStringType("Test")

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "customtypes",
			"id":   "1",
			"attributes": map[string]interface{}{
				"int":        5,
				"intptr":     5,
				"intptrnull": nil,

				"float":  1.5,
				"string": "Test",
			},
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	// Parse JSON API payload
	customAttributeTypes := new(CustomAttributeTypes)
	if err := UnmarshalPayload(bytes.NewReader(payload), customAttributeTypes); err != nil {
		t.Fatal(err)
	}

	if expected, actual := customInt, customAttributeTypes.Int; expected != actual {
		t.Fatalf("Was expecting custom int to be `%d`, got `%d`", expected, actual)
	}
	if expected, actual := customInt, *customAttributeTypes.IntPtr; expected != actual {
		t.Fatalf("Was expecting custom int pointer to be `%d`, got `%d`", expected, actual)
	}
	if customAttributeTypes.IntPtrNull != nil {
		t.Fatalf("Was expecting custom int pointer to be <nil>, got `%d`", customAttributeTypes.IntPtrNull)
	}

	if expected, actual := customFloat, customAttributeTypes.Float; expected != actual {
		t.Fatalf("Was expecting custom float to be `%f`, got `%f`", expected, actual)
	}
	if expected, actual := customString, customAttributeTypes.String; expected != actual {
		t.Fatalf("Was expecting custom string to be `%s`, got `%s`", expected, actual)
	}
}

func TestUnmarshalCustomTypeAttributes_ErrInvalidType(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "customtypes",
			"id":   "1",
			"attributes": map[string]interface{}{
				"int":        "bad",
				"intptr":     5,
				"intptrnull": nil,

				"float":  1.5,
				"string": "Test",
			},
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	// Parse JSON API payload
	customAttributeTypes := new(CustomAttributeTypes)
	err = UnmarshalPayload(bytes.NewReader(payload), customAttributeTypes)
	if err == nil {
		t.Fatal("Expected an error unmarshalling the payload due to type mismatch, got none")
	}

	if err != ErrInvalidType {
		t.Fatalf("Expected error to be %v, was %v", ErrInvalidType, err)
	}
}

func samplePayloadWithoutIncluded() map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"type": "posts",
			"id":   "1",
			"attributes": map[string]interface{}{
				"body":  "Hello",
				"title": "World",
			},
			"relationships": map[string]interface{}{
				"comments": map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{
							"type": "comments",
							"id":   "123",
						},
						map[string]interface{}{
							"type": "comments",
							"id":   "456",
						},
					},
				},
				"latest_comment": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "comments",
						"id":   "55555",
					},
				},
			},
		},
	}
}

func samplePayload() io.Reader {
	payload := &OnePayload{
		Data: &Node{
			Type: "blogs",
			Attributes: map[string]interface{}{
				"title":      "New blog",
				"created_at": 1436216820,
				"view_count": 1000,
			},
			Relationships: map[string]interface{}{
				"posts": &RelationshipManyNode{
					Data: []*Node{
						{
							Type: "posts",
							Attributes: map[string]interface{}{
								"title": "Foo",
								"body":  "Bar",
							},
							ClientID: "1",
							Links: &Links{
								"self": "http://somesite.com/posts/1",
							},
						},
						{
							Type: "posts",
							Attributes: map[string]interface{}{
								"title": "X",
								"body":  "Y",
							},
							ClientID: "2",
							Links: &Links{
								"self": "http://somesite.com/posts/2",
							},
						},
					},
				},
				"current_post": &RelationshipOneNode{
					Data: &Node{
						Type: "posts",
						Attributes: map[string]interface{}{
							"title": "Bas",
							"body":  "Fuubar",
						},
						ClientID: "3",
						Relationships: map[string]interface{}{
							"comments": &RelationshipManyNode{
								Data: []*Node{
									{
										Type: "comments",
										Attributes: map[string]interface{}{
											"body": "Great post!",
										},
										ClientID: "4",
										Links: &Links{
											"self": "http://somesite.com/comments/1",
										},
									},
									{
										Type: "comments",
										Attributes: map[string]interface{}{
											"body": "Needs some work!",
										},
										ClientID: "5",
										Links: &Links{
											"self": "http://somesite.com/comments/2",
										},
									},
								},
							},
						},
						Links: &Links{
							"self": &Link{
								Href: "http://somesite.com/posts/2",
								Meta: Meta{
									"foo": "bar",
								},
							},
						},
					},
				},
			},
			Links: &Links{
				"self": "http://somesite.com/blogs/1",
			},
		},
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(payload) //nolint:errcheck

	return out
}

func samplePayloadWithID() io.Reader {
	payload := &OnePayload{
		Data: &Node{
			ID:   "2",
			Type: "blogs",
			Attributes: map[string]interface{}{
				"title":      "New blog",
				"view_count": 1000,
			},
		},
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(payload) //nolint:errcheck

	return out
}

func samplePayloadWithBadTypes(m map[string]interface{}) io.Reader {
	payload := &OnePayload{
		Data: &Node{
			ID:         "2",
			Type:       "badtypes",
			Attributes: m,
		},
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(payload) //nolint:errcheck

	return out
}

func sampleWithPointerPayload(m map[string]interface{}) io.Reader {
	payload := &OnePayload{
		Data: &Node{
			ID:         "2",
			Type:       "with-pointers",
			Attributes: m,
		},
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(payload) //nolint:errcheck

	return out
}

func samplePayloadWithNullableAttrs(m map[string]interface{}) io.Reader {
	payload := &OnePayload{
		Data: &Node{
			ID:         "5",
			Type:       "with-nullables",
			Attributes: m,
		},
	}

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(payload) //nolint:errcheck

	return out
}

func testModel() *Blog {
	return &Blog{
		ID:        5,
		ClientID:  "1",
		Title:     "Title 1",
		CreatedAt: time.Now(),
		Posts: []*Post{
			{
				ID:    1,
				Title: "Foo",
				Body:  "Bar",
				Comments: []*Comment{
					{
						ID:   1,
						Body: "foo",
					},
					{
						ID:   2,
						Body: "bar",
					},
				},
				LatestComment: &Comment{
					ID:   1,
					Body: "foo",
				},
			},
			{
				ID:    2,
				Title: "Fuubar",
				Body:  "Bas",
				Comments: []*Comment{
					{
						ID:   1,
						Body: "foo",
					},
					{
						ID:   3,
						Body: "bas",
					},
				},
				LatestComment: &Comment{
					ID:   1,
					Body: "foo",
				},
			},
		},
		CurrentPost: &Post{
			ID:    1,
			Title: "Foo",
			Body:  "Bar",
			Comments: []*Comment{
				{
					ID:   1,
					Body: "foo",
				},
				{
					ID:   2,
					Body: "bar",
				},
			},
			LatestComment: &Comment{
				ID:   1,
				Body: "foo",
			},
		},
	}
}

func samplePayloadWithSideloaded() (io.Reader, error) {
	testModel := testModel()

	out := bytes.NewBuffer(nil)
	if err := MarshalPayload(out, testModel); err != nil {
		return nil, err
	}

	return out, nil
}

func sampleSerializedEmbeddedTestModel() (*Blog, error) {
	out := bytes.NewBuffer(nil)
	if err := MarshalOnePayloadEmbedded(out, testModel()); err != nil {
		return nil, err
	}

	blog := new(Blog)
	if err := UnmarshalPayload(out, blog); err != nil {
		return nil, err
	}

	return blog, nil
}

func TestUnmarshalNestedStructPtr(t *testing.T) {
	type Director struct {
		Firstname string `jsonapi:"attr,firstname"`
		Surname   string `jsonapi:"attr,surname"`
	}
	type Movie struct {
		ID       string    `jsonapi:"primary,movies"`
		Name     string    `jsonapi:"attr,name"`
		Director *Director `jsonapi:"attr,director"`
	}
	sample := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "movies",
			"id":   "123",
			"attributes": map[string]interface{}{
				"name": "The Shawshank Redemption",
				"director": map[string]interface{}{
					"firstname": "Frank",
					"surname":   "Darabont",
				},
			},
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)
	out := new(Movie)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Name != "The Shawshank Redemption" {
		t.Fatalf("expected out.Name to be `The Shawshank Redemption`, but got `%s`", out.Name)
	}
	if out.Director.Firstname != "Frank" {
		t.Fatalf("expected out.Director.Firstname to be `Frank`, but got `%s`", out.Director.Firstname)
	}
	if out.Director.Surname != "Darabont" {
		t.Fatalf("expected out.Director.Surname to be `Darabont`, but got `%s`", out.Director.Surname)
	}
}

func TestUnmarshalNestedStruct(t *testing.T) {
	boss := map[string]interface{}{
		"firstname": "Hubert",
		"surname":   "Farnsworth",
		"age":       176,
		"hired-at":  "2016-08-17T08:27:12Z",
	}

	sample := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "companies",
			"id":   "123",
			"attributes": map[string]interface{}{
				"name":       "Planet Express",
				"boss":       boss,
				"founded-at": "2016-08-17T08:27:12Z",
				"teams": []map[string]interface{}{
					map[string]interface{}{
						"name": "Dev",
						"members": []map[string]interface{}{
							map[string]interface{}{"firstname": "Sean"},
							map[string]interface{}{"firstname": "Iz"},
						},
						"leader": map[string]interface{}{"firstname": "Iz"},
					},
					map[string]interface{}{
						"name": "DxE",
						"members": []map[string]interface{}{
							map[string]interface{}{"firstname": "Akshay"},
							map[string]interface{}{"firstname": "Peri"},
						},
						"leader": map[string]interface{}{"firstname": "Peri"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)
	out := new(Company)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Boss.Firstname != "Hubert" {
		t.Fatalf("expected `Hubert` at out.Boss.Firstname, but got `%s`", out.Boss.Firstname)
	}

	if out.Boss.Age != 176 {
		t.Fatalf("expected `176` at out.Boss.Age, but got `%d`", out.Boss.Age)
	}

	if out.Boss.HiredAt.IsZero() {
		t.Fatalf("expected out.Boss.HiredAt to be zero, but got `%t`", out.Boss.HiredAt.IsZero())
	}

	if len(out.Teams) != 2 {
		t.Fatalf("expected len(out.Teams) to be 2, but got `%d`", len(out.Teams))
	}

	if out.Teams[0].Name != "Dev" {
		t.Fatalf("expected out.Teams[0].Name to be `Dev`, but got `%s`", out.Teams[0].Name)
	}

	if out.Teams[1].Name != "DxE" {
		t.Fatalf("expected out.Teams[1].Name to be `DxE`, but got `%s`", out.Teams[1].Name)
	}

	if len(out.Teams[0].Members) != 2 {
		t.Fatalf("expected len(out.Teams[0].Members) to be 2, but got `%d`", len(out.Teams[0].Members))
	}

	if len(out.Teams[1].Members) != 2 {
		t.Fatalf("expected len(out.Teams[1].Members) to be 2, but got `%d`", len(out.Teams[1].Members))
	}

	if out.Teams[0].Members[0].Firstname != "Sean" {
		t.Fatalf("expected out.Teams[0].Members[0].Firstname to be `Sean`, but got `%s`", out.Teams[0].Members[0].Firstname)
	}

	if out.Teams[0].Members[1].Firstname != "Iz" {
		t.Fatalf("expected out.Teams[0].Members[1].Firstname to be `Iz`, but got `%s`", out.Teams[0].Members[1].Firstname)
	}

	if out.Teams[1].Members[0].Firstname != "Akshay" {
		t.Fatalf("expected out.Teams[1].Members[0].Firstname to be `Akshay`, but got `%s`", out.Teams[1].Members[0].Firstname)
	}

	if out.Teams[1].Members[1].Firstname != "Peri" {
		t.Fatalf("expected out.Teams[1].Members[1].Firstname to be `Peri`, but got `%s`", out.Teams[1].Members[1].Firstname)
	}

	if out.Teams[0].Leader.Firstname != "Iz" {
		t.Fatalf("expected out.Teams[0].Leader.Firstname to be `Iz`, but got `%s`", out.Teams[0].Leader.Firstname)
	}

	if out.Teams[1].Leader.Firstname != "Peri" {
		t.Fatalf("expected out.Teams[1].Leader.Firstname to be `Peri`, but got `%s`", out.Teams[1].Leader.Firstname)
	}
}

func TestUnmarshalNestedStructSlice(t *testing.T) {
	fry := map[string]interface{}{
		"firstname": "Philip J.",
		"surname":   "Fry",
		"age":       25,
		"hired-at":  "2016-08-17T08:27:12Z",
	}

	bender := map[string]interface{}{
		"firstname": "Bender Bending",
		"surname":   "Rodriguez",
		"age":       19,
		"hired-at":  "2016-08-17T08:27:12Z",
	}

	deliveryCrew := map[string]interface{}{
		"name": "Delivery Crew",
		"members": []interface{}{
			fry,
			bender,
		},
	}

	sample := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "companies",
			"id":   "123",
			"attributes": map[string]interface{}{
				"name": "Planet Express",
				"teams": []interface{}{
					deliveryCrew,
				},
			},
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)
	out := new(Company)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if out.Teams[0].Name != "Delivery Crew" {
		t.Fatalf("Nested struct not unmarshalled: Expected `Delivery Crew` but got `%s`", out.Teams[0].Name)
	}

	if len(out.Teams[0].Members) != 2 {
		t.Fatalf("Nested struct not unmarshalled: Expected to have `2` Members but got `%d`",
			len(out.Teams[0].Members))
	}

	if out.Teams[0].Members[0].Firstname != "Philip J." {
		t.Fatalf("Nested struct not unmarshalled: Expected `Philip J.` but got `%s`",
			out.Teams[0].Members[0].Firstname)
	}
}

func TestUnmarshalNestedStructPointerSlice(t *testing.T) {
	personA := map[string]interface{}{
		"name": "persona",
		"age":  25,
	}

	personB := map[string]interface{}{
		"name": "personb",
		"age":  19,
	}

	sample := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "companies",
			"id":   "123",
			"attributes": map[string]interface{}{
				"name": "Planet Express",
				"people": []interface{}{
					personA,
					personB,
				},
			},
		},
	}

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatal(err)
	}
	in := bytes.NewReader(data)
	out := new(Company)

	if err := UnmarshalPayload(in, out); err != nil {
		t.Fatal(err)
	}

	if len(out.People) != 2 {
		t.Fatalf("Length of people should be 2, but is instead %d", len(out.People))
	}

	if out.People[0].Name != "persona" {
		t.Fatalf("Nested pointer struct not unmarshalled: Expected `persona` but got `%s`", out.People[0].Name)
	}

	if out.People[0].Age != 25 {
		t.Fatalf("Nested pointer struct not unmarshalled: Expected `25` but got `%d`", out.People[0].Age)
	}

	if out.People[1].Name != "personb" {
		t.Fatalf("Nested pointer struct not unmarshalled: Expected `personb` but got `%s`", out.People[1].Name)
	}

	if out.People[1].Age != 19 {
		t.Fatalf("Nested pointer struct not unmarshalled: Expected `19` but got `%d`", out.People[1].Age)
	}
}
