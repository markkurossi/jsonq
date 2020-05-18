//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package jsonq

import (
	"errors"
	"fmt"
	"reflect"
)

// Context filters JSON object with Select and extracts values with
// Extract.
type Context struct {
	selection []interface{}
	err       error
}

// Ctx creates a new selection context for the argument JSON root
// value.
func Ctx(root interface{}) *Context {
	return &Context{
		selection: []interface{}{root},
	}
}

// Select selects elements from the context.
func (ctx *Context) Select(q string) *Context {
	if ctx.err != nil {
		return ctx
	}
	var result []interface{}
	for _, sel := range ctx.selection {
		elements, err := Get(sel, q)
		if err != nil {
			ctx.err = err
			return ctx
		}
		result = append(result, elements...)
	}
	ctx.selection = result

	return ctx
}

// Error describes an invalid argument passed to Extract.
type Error struct {
	Type reflect.Type
}

func (e *Error) Error() string {
	if e.Type == nil {
		return "jsonq: Extract(nil)"
	}
	if e.Type.Kind() != reflect.Ptr {
		return fmt.Sprintf("jsonq: Extract(non-pointer %s)", e.Type)
	}
	return fmt.Sprintf("jsonq: Extract(nil %s)", e.Type)
}

// Extract extracts values from the current selection into the
// argument value object.
func (ctx *Context) Extract(v interface{}) error {
	if ctx.err != nil {
		return ctx.err
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &Error{
			Type: reflect.TypeOf(v),
		}
	}
	if len(ctx.selection) == 0 {
		return errors.New("jsonq: empty selection")
	}

	// Check the pointed value's type.
	pointed := reflect.Indirect(rv)
	switch pointed.Type().Kind() {
	case reflect.Ptr:
		return fmt.Errorf("jsonq: pointers not implemented yet")

	case reflect.Struct:
		if len(ctx.selection) != 1 {
			return errors.New("jsonq: selection matches more than one item")
		}
		return extractStruct(ctx.selection[0], pointed)

	default:
		return fmt.Errorf("jsonq: pointed: %s", pointed.Type())
	}
}

func extractStruct(sel interface{}, value reflect.Value) error {
	for i := 0; i < value.NumField(); i++ {
		tag := value.Type().Field(i).Tag.Get("jsonq")
		if len(tag) == 0 {
			continue
		}
		field := value.Field(i)
		switch field.Type().Kind() {
		case reflect.String:
			val, err := GetString(sel, tag)
			if err != nil {
				return err
			}
			field.SetString(val)

		default:
			return fmt.Errorf("jsonq: field type %s not supported",
				field.Type())
		}

	}
	return nil
}
