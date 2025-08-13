package jsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

var emptyStruct = reflect.TypeFor[struct{}]()

func ParamsDecoder[E any]() func(p Params) (E, error) {
	typ := reflect.TypeFor[E]()
	kind := typ.Kind()
	pointerTo := false
	if kind == reflect.Pointer {
		kind = typ.Elem().Kind()
		pointerTo = true
	}
	return func(p Params) (dest E, err error) {
		p = bytes.TrimLeft(p, "\t\n\r ")
		if len(p) == 0 {
			// params may be omitted if we are not decoding into anything
			if typ == emptyStruct {
				return
			}
			err = errors.New("empty params data")
			return
		}
		switch kind {
		case reflect.Slice:
			switch p[0] {
			case '{':
				err = fmt.Errorf("cannot decode named RPC params into list")
				return
			case '[':
				dec := json.NewDecoder(bytes.NewReader(p))
				dec.UseNumber()
				err = dec.Decode(&dest)
				return
			default:
				err = errors.New("invalid params")
				return
			}
		case reflect.Struct:
			switch p[0] {
			case '{':
				err = json.Unmarshal(p, &dest)
				return
			case '[':
				var items []json.RawMessage
				err = json.Unmarshal(p, &items)
				if err != nil {
					return
				}
				v := reflect.ValueOf(&dest).Elem()
				if pointerTo { // allocate a value if the dest is just a pointer type
					v.Set(reflect.New(typ.Elem()))
					v = v.Elem()
				}
				if expected := v.NumField(); expected != len(items) {
					err = fmt.Errorf("expected %d params, got %d params", expected, len(items))
					return
				}
				for i, itemData := range items {
					if fErr := json.Unmarshal(itemData, v.Field(i).Addr().Interface()); fErr != nil {
						err = errors.Join(err, fmt.Errorf("failed to decode field %d: %w", i, fErr))
					}
				}
				return
			default:
				err = errors.New("invalid params")
				return
			}
		default:
			err = json.Unmarshal(p, &dest)
			return
		}
	}
}
