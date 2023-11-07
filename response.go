package jsonapi

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

var (
	// ErrBadJSONAPIStructTag is returned when the Struct field's JSON API
	// annotation is invalid.
	ErrBadJSONAPIStructTag = errors.New("Bad jsonapi struct tag format")
	// ErrBadJSONAPIID is returned when the Struct JSON API annotated "id" field
	// was not a valid numeric type.
	ErrBadJSONAPIID = errors.New(
		"id should be either string, int(8,16,32,64) or uint(8,16,32,64)")
	// ErrExpectedSlice is returned when a variable or argument was expected to
	// be a slice of *Structs; MarshalMany will return this error when its
	// interface{} argument is invalid.
	ErrExpectedSlice = errors.New("models should be a slice of struct pointers")
	// ErrUnexpectedType is returned when marshalling an interface; the interface
	// had to be a pointer or a slice; otherwise this error is returned.
	ErrUnexpectedType = errors.New("models should be a struct pointer or slice of struct pointers")
	// ErrUnexpectedNil is returned when a slice of relation structs contains nil values
	ErrUnexpectedNil = errors.New("slice of struct pointers cannot contain nil")
)

// MarshalPayload writes a jsonapi response for one or many records. The
// related records are sideloaded into the "included" array. If this method is
// given a struct pointer as an argument it will serialize in the form
// "data": {...}. If this method is given a slice of pointers, this method will
// serialize in the form "data": [...]
//
// One Example: you could pass it, w, your http.ResponseWriter, and, models, a
// ptr to a Blog to be written to the response body:
//
//	 func ShowBlog(w http.ResponseWriter, r *http.Request) {
//		 blog := &Blog{}
//
//		 w.Header().Set("Content-Type", jsonapi.MediaType)
//		 w.WriteHeader(http.StatusOK)
//
//		 if err := jsonapi.MarshalPayload(w, blog); err != nil {
//			 http.Error(w, err.Error(), http.StatusInternalServerError)
//		 }
//	 }
//
// Many Example: you could pass it, w, your http.ResponseWriter, and, models, a
// slice of Blog struct instance pointers to be written to the response body:
//
//		 func ListBlogs(w http.ResponseWriter, r *http.Request) {
//	    blogs := []*Blog{}
//
//			 w.Header().Set("Content-Type", jsonapi.MediaType)
//			 w.WriteHeader(http.StatusOK)
//
//			 if err := jsonapi.MarshalPayload(w, blogs); err != nil {
//				 http.Error(w, err.Error(), http.StatusInternalServerError)
//			 }
//		 }
func MarshalPayload(w io.Writer, models interface{}) error {
	payload, err := Marshal(models)
	if err != nil {
		return err
	}

	return json.NewEncoder(w).Encode(payload)
}

// Marshal does the same as MarshalPayload except it just returns the payload
// and doesn't write out results. Useful if you use your own JSON rendering
// library.
func Marshal(models interface{}) (Payloader, error) {
	switch vals := reflect.ValueOf(models); vals.Kind() {
	case reflect.Slice:
		m, err := convertToSliceInterface(&models)
		if err != nil {
			return nil, err
		}

		payload, err := marshalMany(m)
		if err != nil {
			return nil, err
		}

		if linkableModels, isLinkable := models.(Linkable); isLinkable {
			jl := linkableModels.JSONAPILinks()
			if er := jl.validate(); er != nil {
				return nil, er
			}
			payload.Links = linkableModels.JSONAPILinks()
		}

		if metableModels, ok := models.(Metable); ok {
			payload.Meta = metableModels.JSONAPIMeta()
		}

		return payload, nil
	case reflect.Ptr:
		// Check that the pointer was to a struct
		if reflect.Indirect(vals).Kind() != reflect.Struct {
			return nil, ErrUnexpectedType
		}
		return marshalOne(models)
	default:
		return nil, ErrUnexpectedType
	}
}

// MarshalPayloadWithoutIncluded writes a jsonapi response with one or many
// records, without the related records sideloaded into "included" array.
// If you want to serialize the relations into the "included" array see
// MarshalPayload.
//
// models interface{} should be either a struct pointer or a slice of struct
// pointers.
func MarshalPayloadWithoutIncluded(w io.Writer, model interface{}) error {
	payload, err := Marshal(model)
	if err != nil {
		return err
	}
	payload.clearIncluded()

	return json.NewEncoder(w).Encode(payload)
}

// marshalOne does the same as MarshalOnePayload except it just returns the
// payload and doesn't write out results. Useful is you use your JSON rendering
// library.
func marshalOne(model interface{}) (*OnePayload, error) {
	var included = make(map[string]*Node)
	rootNode, err := visitModelNode(model, included, true)
	if err != nil {
		return nil, err
	}
	payload := &OnePayload{Data: rootNode}

	payload.Included = nodeMapValues(included)

	return payload, nil
}

// marshalMany does the same as MarshalManyPayload except it just returns the
// payload and doesn't write out results. Useful is you use your JSON rendering
// library.
func marshalMany(models []interface{}) (*ManyPayload, error) {
	payload := &ManyPayload{
		Data: []*Node{},
	}

	included := make(map[string]*Node)

	for _, model := range models {
		var node *Node
		var err error
		node, err = visitModelNode(model, included, true)
		if err != nil {
			return nil, err
		}
		payload.Data = append(payload.Data, node)
	}
	payload.Included = nodeMapValues(included)

	return payload, nil
}

// MarshalOnePayloadEmbedded - This method not meant to for use in
// implementation code, although feel free.  The purpose of this
// method is for use in tests.  In most cases, your request
// payloads for create will be embedded rather than sideloaded for
// related records. This method will serialize a single struct
// pointer into an embedded json response. In other words, there
// will be no, "included", array in the json all relationships will
// be serialized inline in the data.
//
// However, in tests, you may want to construct payloads to post
// to create methods that are embedded to most closely resemble
// the payloads that will be produced by the client. This is what
// this method is intended for.
//
// model interface{} should be a pointer to a struct.
func MarshalOnePayloadEmbedded(w io.Writer, model interface{}) error {
	rootNode, err := visitModelNode(model, nil, false)
	if err != nil {
		return err
	}

	payload := &OnePayload{Data: rootNode}

	return json.NewEncoder(w).Encode(payload)
}

func visitModelNode(model interface{}, included map[string]*Node,
	sideload bool) (*Node, error) {

	visitor := &modelVisitor{
		Included: included,
		Sideload: sideload,
	}

	return visitor.Visit(model)
}

func nodeMapValues(m map[string]*Node) []*Node {
	nodes := make([]*Node, len(m))

	i := 0
	for _, n := range m {
		nodes[i] = n
		i++
	}

	return nodes
}

func convertToSliceInterface(i *interface{}) ([]interface{}, error) {
	vals := reflect.ValueOf(*i)
	if vals.Kind() != reflect.Slice {
		return nil, ErrExpectedSlice
	}
	var response []interface{}
	for x := 0; x < vals.Len(); x++ {
		response = append(response, vals.Index(x).Interface())
	}
	return response, nil
}
