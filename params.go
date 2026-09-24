package jsonrpc2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// ParamsDecoder returns a function that decodes request params into E.
//
// A slice E (or pointer to a slice) decodes positional params: an array.
//
// A struct E (or pointer to a struct) decodes named params, an object, with encoding/json.
// It also decodes positional params, one array item into each field, in the order of declaration.
// Trailing fields tagged omitempty (or omitzero) are optional: the array may end before them,
// and they keep their zero value, e.g. an optional block tag after a call object.
// Positional params cannot decode into unexported fields.
// Omitted params decode like an empty array: into a struct without required fields, such as struct{}.
//
// Other types decode the params with encoding/json.
//
// Numbers decode into interface values as json.Number, and data after the params value is an error.
// On error, the decoder returns the zero E.
func ParamsDecoder[E any]() func(p Params) (E, error) {
	typ := reflect.TypeFor[E]()
	elem := typ
	if elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	decode := func(p Params) (E, error) {
		var dest E
		if err := decodeJSON(p, &dest); err != nil {
			return *new(E), err
		}
		return dest, nil
	}
	switch elem.Kind() {
	case reflect.Slice:
		return func(p Params) (E, error) {
			p = bytes.TrimLeft(p, "\t\n\r ")
			if len(p) == 0 {
				return *new(E), errors.New("empty params data")
			}
			if p[0] != '[' {
				return *new(E), errors.New("cannot decode named RPC params into list")
			}
			return decode(p)
		}
	case reflect.Struct:
		pos := positionalOf(elem)
		decodePositional := func(items []json.RawMessage) (E, error) {
			var dest E
			v := reflect.ValueOf(&dest).Elem()
			if typ != elem { // allocate a value if the dest is just a pointer type
				v.Set(reflect.New(elem))
				v = v.Elem()
			}
			if err := pos.decode(items, v); err != nil {
				return *new(E), err
			}
			return dest, nil
		}
		return func(p Params) (E, error) {
			p = bytes.TrimLeft(p, "\t\n\r ")
			if len(p) == 0 {
				return decodePositional(nil)
			}
			switch p[0] {
			case '{':
				return decode(p)
			case '[':
				var items []json.RawMessage
				if err := decodeJSON(p, &items); err != nil {
					return *new(E), err
				}
				return decodePositional(items)
			default:
				return *new(E), errors.New("invalid params")
			}
		}
	default:
		return func(p Params) (E, error) {
			p = bytes.TrimLeft(p, "\t\n\r ")
			if len(p) == 0 {
				return *new(E), errors.New("empty params data")
			}
			return decode(p)
		}
	}
}

// positional describes the decoding of positional params into the fields of a struct.
type positional struct {
	typ      reflect.Type // the struct type
	required int          // the minimum number of params: the fields up to the last one that is not optional
}

func positionalOf(typ reflect.Type) positional {
	out := positional{typ: typ}
	for i := range typ.NumField() {
		if !isOptional(typ.Field(i)) {
			out.required = i + 1
		}
	}
	return out
}

// isOptional reports whether the json tag of f has the omitempty or omitzero option.
func isOptional(f reflect.StructField) bool {
	_, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
	for opts != "" {
		var opt string
		opt, opts, _ = strings.Cut(opts, ",")
		if opt == "omitempty" || opt == "omitzero" {
			return true
		}
	}
	return false
}

// decode decodes items into the fields of the struct v.
func (pos positional) decode(items []json.RawMessage, v reflect.Value) error {
	if n := pos.typ.NumField(); len(items) < pos.required || len(items) > n {
		if pos.required == n {
			return fmt.Errorf("expected %d params, got %d params", n, len(items))
		}
		return fmt.Errorf("expected %d to %d params, got %d params", pos.required, n, len(items))
	}
	var err error
	for i, item := range items {
		if f := pos.typ.Field(i); !f.IsExported() {
			// Addr().Interface() would panic.
			err = errors.Join(err, fmt.Errorf("cannot decode param %d into unexported field %s of %s", i, f.Name, pos.typ))
			continue
		}
		if fErr := decodeJSON(item, v.Field(i).Addr().Interface()); fErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to decode field %d: %w", i, fErr))
		}
	}
	return err
}

// decodeJSON decodes a single JSON value into dest, with numbers in interface values as json.Number.
// Data after the value, other than whitespace, is an error.
func decodeJSON(data []byte, dest any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(dest); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("unexpected data after JSON value")
	}
	return nil
}
