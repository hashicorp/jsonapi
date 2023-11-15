package jsonapi

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// modelVisitor is a type that recursively descends into a given model containing jsonapi
// annotations. Using those annotations, modelVisitor can produce a Node value that can
// in turn be encoded as json.
type modelVisitor struct {
	Included  map[string]*Node
	Sideload  bool
	rootModel interface{}
}

// modelFieldCursor is used to store information about each annotated struct field
// visited by a modelVisitor.
type modelFieldCursor struct {
	structValue reflect.Value
	nextField   int
	currentTag  []string
	modelType   reflect.Type
	modelValue  reflect.Value
	fieldType   reflect.StructField
	fieldValue  reflect.Value
}

// Next advances the cursor to the next struct field in the model. If there are no
// more fields available, false is returned and the cursor is not advanced.
func (c *modelFieldCursor) Next() bool {
	if c.nextField >= c.structValue.NumField() {
		return false
	}

	c.currentTag = []string{}
	c.fieldType = c.modelType.Field(c.nextField)
	c.fieldValue = c.modelValue.Field(c.nextField)

	structField := c.modelValue.Type().Field(c.nextField)
	tag := structField.Tag.Get(annotationJSONAPI)
	if tag != "" {
		c.currentTag = strings.Split(tag, annotationSeparator)
	}

	c.nextField += 1
	return true
}

// TagType returns the first part of the jsonapi tag, which indicates the type
// of field. Examples include "attr", "primary", "relation", "polyrelation"
func (c modelFieldCursor) TagType() string {
	if !c.HasTag() {
		return ""
	}
	return c.currentTag[0]
}

// HasTag helps determine whether the current field has any jsonapi tag.
func (c modelFieldCursor) HasTag() bool {
	return len(c.currentTag) != 0
}

// ValidTag returns a ErrBadJSONAPIStructTag if the tag is improperly annotated
func (c modelFieldCursor) ValidTag() error {
	if !c.HasTag() {
		return ErrBadJSONAPIStructTag
	}

	if (c.TagType() == annotationClientID && len(c.currentTag) != 1) ||
		(c.TagType() != annotationClientID && len(c.currentTag) < 2) {
		return ErrBadJSONAPIStructTag
	}

	return nil
}

// FieldTypePrimary returns true if the field is a "primary" type
func (c modelFieldCursor) FieldTypePrimary() bool {
	return c.TagType() == annotationPrimary
}

// FieldTypeClientID returns true if the field is a "client-id" type
func (c modelFieldCursor) FieldTypeClientID() bool {
	return c.TagType() == annotationClientID
}

// FieldTypeAttribute returns true if the field is a "attr" type
func (c modelFieldCursor) FieldTypeAttribute() bool {
	return c.TagType() == annotationAttribute
}

// FieldTypeRelation returns true if the field is a "relation" type
func (c modelFieldCursor) FieldTypeRelation() bool {
	return c.TagType() == annotationRelation
}

// FieldTypePolyRelation returns true if the field is a "polyrelation" type
func (c modelFieldCursor) FieldTypePolyRelation() bool {
	return c.TagType() == annotationPolyRelation
}

// FieldTypeLinks returns true if the field is a "links" type
func (c modelFieldCursor) FieldTypeLinks() bool {
	return c.TagType() == annotationLinks
}

func (m *modelVisitor) visitFieldTypePrimary(node *Node, cursor *modelFieldCursor) error {
	// Deal with PTRS
	var kind reflect.Kind
	if cursor.fieldValue.Kind() == reflect.Ptr {
		kind = cursor.fieldType.Type.Elem().Kind()
		cursor.fieldValue = reflect.Indirect(cursor.fieldValue)
	} else {
		kind = cursor.fieldType.Type.Kind()
	}

	// Handle allowed types
	fvi := cursor.fieldValue.Interface()

	switch kind {
	case reflect.String:
		node.ID = fvi.(string)
	case reflect.Int:
		node.ID = strconv.FormatInt(int64(fvi.(int)), 10)
	case reflect.Int8:
		node.ID = strconv.FormatInt(int64(fvi.(int8)), 10)
	case reflect.Int16:
		node.ID = strconv.FormatInt(int64(fvi.(int16)), 10)
	case reflect.Int32:
		node.ID = strconv.FormatInt(int64(fvi.(int32)), 10)
	case reflect.Int64:
		node.ID = strconv.FormatInt(fvi.(int64), 10)
	case reflect.Uint:
		node.ID = strconv.FormatUint(uint64(fvi.(uint)), 10)
	case reflect.Uint8:
		node.ID = strconv.FormatUint(uint64(fvi.(uint8)), 10)
	case reflect.Uint16:
		node.ID = strconv.FormatUint(uint64(fvi.(uint16)), 10)
	case reflect.Uint32:
		node.ID = strconv.FormatUint(uint64(fvi.(uint32)), 10)
	case reflect.Uint64:
		node.ID = strconv.FormatUint(fvi.(uint64), 10)
	default:
		// We had a JSON float (numeric), but our field was not one of the
		// allowed numeric types
		return ErrBadJSONAPIID
	}

	node.Type = cursor.currentTag[1]

	return nil
}

func (m *modelVisitor) visitFieldTypeClientID(node *Node, cursor *modelFieldCursor) error {
	clientID := cursor.fieldValue.String()
	if clientID != "" {
		node.ClientID = clientID
	}
	return nil
}

func (m *modelVisitor) visitFieldTypeAttribute(node *Node, cursor *modelFieldCursor) error {
	var omitEmpty, iso8601, rfc3339 bool

	if len(cursor.currentTag) > 2 {
		for _, arg := range cursor.currentTag[2:] {
			switch arg {
			case annotationOmitEmpty:
				omitEmpty = true
			case annotationISO8601:
				iso8601 = true
			case annotationRFC3339:
				rfc3339 = true
			}
		}
	}

	if node.Attributes == nil {
		node.Attributes = make(map[string]interface{})
	}

	// TODO: time.Time and *time.Time handling could be combined
	if cursor.fieldValue.Type() == reflect.TypeOf(time.Time{}) {
		t := cursor.fieldValue.Interface().(time.Time)

		if t.IsZero() {
			return nil
		}

		if iso8601 {
			node.Attributes[cursor.currentTag[1]] = t.UTC().Format(iso8601TimeFormat)
		} else if rfc3339 {
			node.Attributes[cursor.currentTag[1]] = t.UTC().Format(time.RFC3339)
		} else {
			node.Attributes[cursor.currentTag[1]] = t.Unix()
		}
	} else if cursor.fieldValue.Type() == reflect.TypeOf(new(time.Time)) {
		// A time pointer may be nil
		if cursor.fieldValue.IsNil() {
			if omitEmpty {
				return nil
			}

			node.Attributes[cursor.currentTag[1]] = nil
		} else {
			tm := cursor.fieldValue.Interface().(*time.Time)

			if tm.IsZero() && omitEmpty {
				return nil
			}

			if iso8601 {
				node.Attributes[cursor.currentTag[1]] = tm.UTC().Format(iso8601TimeFormat)
			} else if rfc3339 {
				node.Attributes[cursor.currentTag[1]] = tm.UTC().Format(time.RFC3339)
			} else {
				node.Attributes[cursor.currentTag[1]] = tm.Unix()
			}
		}
	} else {
		// Dealing with a fieldValue that is not a time
		emptyValue := reflect.Zero(cursor.fieldValue.Type())

		// See if we need to omit this field
		if omitEmpty && reflect.DeepEqual(cursor.fieldValue.Interface(), emptyValue.Interface()) {
			return nil
		}

		strAttr, ok := cursor.fieldValue.Interface().(string)
		if ok {
			node.Attributes[cursor.currentTag[1]] = strAttr
		} else {
			node.Attributes[cursor.currentTag[1]] = cursor.fieldValue.Interface()
		}
	}

	return nil
}

// selectChoiceTypeStructField returns the first non-nil struct pointer field in the
// specified struct value that has a jsonapi type field defined within it.
// An error is returned if there are no fields matching that definition.
func selectChoiceTypeStructField(structValue reflect.Value) (reflect.Value, error) {
	for i := 0; i < structValue.NumField(); i++ {
		choiceFieldValue := structValue.Field(i)
		choiceTypeField := choiceFieldValue.Type()

		// Must be a pointer
		if choiceTypeField.Kind() != reflect.Ptr {
			continue
		}

		// Must not be nil
		if choiceFieldValue.IsNil() {
			continue
		}

		subtype := choiceTypeField.Elem()
		_, err := jsonapiTypeOfModel(subtype)
		if err == nil {
			return choiceFieldValue, nil
		}
	}

	return reflect.Value{}, errors.New("no non-nil choice field was found in the specified struct")
}

// toShallowNode takes a node and returns a shallow version of the node.
// If the ID is empty, we include attributes into the shallow version.
//
// An example of where this is useful would be if an object
// within a relationship can be created at the same time as
// the root node.
//
// This is not 1.0 jsonapi spec compliant--it's a bespoke variation on
// resource object identifiers discussed in the pending 1.1 spec.
func toShallowNode(node *Node) *Node {
	ret := &Node{Type: node.Type}
	if node.ID == "" {
		ret.Attributes = node.Attributes
	} else {
		ret.ID = node.ID
	}
	return ret
}

func (m *modelVisitor) visitModelNodeRelationships(models reflect.Value) (*RelationshipManyNode, error) {
	nodes := []*Node{}

	for i := 0; i < models.Len(); i++ {
		model := models.Index(i)
		if !model.IsValid() || model.IsNil() {
			return nil, ErrUnexpectedNil
		}

		n := model.Interface()

		node, err := m.Visit(n)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	return &RelationshipManyNode{Data: nodes}, nil
}

func (m *modelVisitor) appendIncluded(nodes ...*Node) {
	for _, n := range nodes {
		k := fmt.Sprintf("%s,%s", n.Type, n.ID)

		if _, hasNode := m.Included[k]; hasNode {
			continue
		}

		m.Included[k] = n
	}
}

func (m *modelVisitor) visitFieldTypeRelation(node *Node, cursor *modelFieldCursor) error {
	var omitEmpty bool

	// add support for 'omitempty' struct tag for marshaling as absent
	if len(cursor.currentTag) > 2 {
		omitEmpty = cursor.currentTag[2] == annotationOmitEmpty
	}

	isSlice := cursor.fieldValue.Type().Kind() == reflect.Slice
	if omitEmpty &&
		(isSlice && cursor.fieldValue.Len() < 1 ||
			(!isSlice && cursor.fieldValue.IsNil())) {
		return nil
	}

	if cursor.TagType() == annotationPolyRelation {
		// for polyrelation, we'll snoop out the actual relation model
		// through the choice type value by choosing the first non-nil
		// field that has a jsonapi type annotation and overwriting
		// `fieldValue` so normal annotation-assisted marshaling
		// can continue
		if !isSlice {
			choiceValue := cursor.fieldValue

			// must be a pointer type
			if choiceValue.Type().Kind() != reflect.Ptr {
				return ErrUnexpectedType
			}

			if choiceValue.IsNil() {
				cursor.fieldValue = reflect.ValueOf(nil)
			}
			structValue := choiceValue.Elem()

			// Short circuit if field is omitted from model
			if !structValue.IsValid() {
				return nil
			}

			if found, err := selectChoiceTypeStructField(structValue); err == nil {
				cursor.fieldValue = found
			}
		} else {
			// A slice polyrelation field can be... polymorphic... meaning
			// that we might snoop different types within each slice element.
			// Each snooped value will added to this collection and then
			// the recursion will take care of the rest. The only special case
			// is nil. For that, we'll just choose the first
			collection := make([]interface{}, 0)

			for i := 0; i < cursor.fieldValue.Len(); i++ {
				itemValue := cursor.fieldValue.Index(i)
				// Once again, must be a pointer type
				if itemValue.Type().Kind() != reflect.Ptr {
					return ErrUnexpectedType
				}

				if itemValue.IsNil() {
					return ErrUnexpectedNil
				}

				structValue := itemValue.Elem()

				if found, err := selectChoiceTypeStructField(structValue); err == nil {
					collection = append(collection, found.Interface())
				}
			}

			cursor.fieldValue = reflect.ValueOf(collection)
		}
	}

	if node.Relationships == nil {
		node.Relationships = make(map[string]interface{})
	}

	var relLinks *Links
	if linkableModel, ok := m.rootModel.(RelationshipLinkable); ok {
		relLinks = linkableModel.JSONAPIRelationshipLinks(cursor.currentTag[1])
	}

	var relMeta *Meta
	if metableModel, ok := m.rootModel.(RelationshipMetable); ok {
		relMeta = metableModel.JSONAPIRelationshipMeta(cursor.currentTag[1])
	}

	if isSlice {
		// to-many relationship
		relationship, err := m.visitModelNodeRelationships(
			cursor.fieldValue,
		)
		if err != nil {
			return err
		}
		relationship.Links = relLinks
		relationship.Meta = relMeta

		if m.Sideload {
			shallowNodes := []*Node{}
			for _, n := range relationship.Data {
				m.appendIncluded(n)
				shallowNodes = append(shallowNodes, toShallowNode(n))
			}

			node.Relationships[cursor.currentTag[1]] = &RelationshipManyNode{
				Data:  shallowNodes,
				Links: relationship.Links,
				Meta:  relationship.Meta,
			}
		} else {
			node.Relationships[cursor.currentTag[1]] = relationship
		}
	} else {
		// to-one relationships

		// Handle null relationship case
		if cursor.fieldValue.IsNil() {
			node.Relationships[cursor.currentTag[1]] = &RelationshipOneNode{Data: nil}
			return nil
		}

		relationship, err := m.Visit(
			cursor.fieldValue.Interface(),
		)
		if err != nil {
			return err
		}

		if m.Sideload {
			m.appendIncluded(relationship)
			node.Relationships[cursor.currentTag[1]] = &RelationshipOneNode{
				Data:  toShallowNode(relationship),
				Links: relLinks,
				Meta:  relMeta,
			}
		} else {
			node.Relationships[cursor.currentTag[1]] = &RelationshipOneNode{
				Data:  relationship,
				Links: relLinks,
				Meta:  relMeta,
			}
		}
	}

	return nil
}

// visitField will populate `node` using data found at the current cursor location
func (m *modelVisitor) visitField(node *Node, cursor *modelFieldCursor) (err error) {
	switch {
	case cursor.FieldTypePrimary():
		err = m.visitFieldTypePrimary(node, cursor)
	case cursor.FieldTypeClientID():
		err = m.visitFieldTypeClientID(node, cursor)
	case cursor.FieldTypeAttribute():
		err = m.visitFieldTypeAttribute(node, cursor)
	case cursor.FieldTypeRelation() || cursor.FieldTypePolyRelation():
		err = m.visitFieldTypeRelation(node, cursor)
	case cursor.FieldTypeLinks():
		// Nothing. Ignore this field, as Links fields are only for unmarshaling requests.
		// The Linkable interface methods are used for marshaling data in a response.
	default:
		err = ErrBadJSONAPIStructTag
	}

	return
}

// newModelCursor creates a new struct field cursor given a model
func newModelCursor(model reflect.Value) *modelFieldCursor {
	modelValue := model.Elem()
	modelType := model.Type().Elem()

	result := &modelFieldCursor{
		structValue: modelValue,
		modelType:   modelType,
		modelValue:  modelValue,
	}

	return result
}

// Visit recursively descends into the given model interface and returns a Node that can
// be Marshaled as json.
func (m *modelVisitor) Visit(next interface{}) (*Node, error) {
	node := new(Node)

	// rootModel tracks the initial model such that recursive calls don't overwrite it
	if m.rootModel == nil {
		m.rootModel = next
		defer func() { m.rootModel = nil }()
	}

	value := reflect.ValueOf(next)
	if value.IsNil() {
		return nil, nil
	}

	cursor := newModelCursor(value)
	for {
		if !cursor.Next() {
			break
		}

		// If this field is not annotated at all, skip it
		if !cursor.HasTag() {
			continue
		}

		// Check to see if the annotation is malformed
		if err := cursor.ValidTag(); err != nil {
			return nil, err
		}

		if err := m.visitField(node, cursor); err != nil {
			return nil, err
		}
	}

	if linkableModel, isLinkable := next.(Linkable); isLinkable {
		jl := linkableModel.JSONAPILinks()
		if er := jl.validate(); er != nil {
			return nil, er
		}
		node.Links = linkableModel.JSONAPILinks()
	}

	if metableModel, ok := next.(Metable); ok {
		node.Meta = metableModel.JSONAPIMeta()
	}

	return node, nil
}
