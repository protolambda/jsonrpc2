package jsonrpc

import (
	"encoding/json"
	"testing"
)

type TestObj struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func TestParamsDecoder(t *testing.T) {
	t.Run("empty data empty dest", func(t *testing.T) {
		dec := ParamsDecoder[struct{}]()
		_, err := dec(Params(""))
		if err != nil {
			t.Fatal("expected to decode empty params data into empty struct")
		}
	})
	t.Run("empty data into struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		_, err := dec(Params(""))
		if err == nil {
			t.Fatal("cannot accept empty data into struct")
		}
	})
	t.Run("empty data into slice", func(t *testing.T) {
		dec := ParamsDecoder[[]any]()
		_, err := dec(Params(""))
		if err == nil {
			t.Fatal("cannot accept empty data into slice")
		}
	})
	t.Run("empty data into string", func(t *testing.T) {
		dec := ParamsDecoder[string]()
		_, err := dec(Params(""))
		if err == nil {
			t.Fatal("cannot accept empty data into string")
		}
	})
	t.Run("empty string", func(t *testing.T) {
		dec := ParamsDecoder[string]()
		_, err := dec(Params(`""`))
		if err != nil {
			t.Fatal("must accept empty string")
		}
	})
	t.Run("basic string", func(t *testing.T) {
		dec := ParamsDecoder[string]()
		got, err := dec(Params(`"hello world"`))
		if err != nil {
			t.Fatal("must accept empty string")
		}
		if got != "hello world" {
			t.Fatal("expected hello message")
		}
	})
	t.Run("number", func(t *testing.T) {
		dec := ParamsDecoder[int]()
		got, err := dec(Params(`1234`))
		if err != nil {
			t.Fatal("must accept empty string")
		}
		if got != 1234 {
			t.Fatal("expected int value")
		}
	})
	t.Run("empty object into struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		got, err := dec(Params(`{}`))
		if err != nil {
			t.Fatal("must accept empty object", err)
		}
		if got.Foo != "" {
			t.Fatal("unexpected Foo")
		}
		if got.Bar != 0 {
			t.Fatal("unexpected Bar")
		}
	})
	t.Run("named data into pointer struct", func(t *testing.T) {
		dec := ParamsDecoder[*TestObj]()
		got, err := dec(Params(`{"foo": "hello", "bar": 1234}`))
		if err != nil {
			t.Fatal("must accept proper object. Err:", err)
		}
		if got.Foo != "hello" {
			t.Fatal("unexpected Foo")
		}
		if got.Bar != 1234 {
			t.Fatal("unexpected Bar")
		}
	})
	t.Run("named data into struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		got, err := dec(Params(`{"foo": "hello", "bar": 1234}`))
		if err != nil {
			t.Fatal("must accept proper object. Err:", err)
		}
		if got.Foo != "hello" {
			t.Fatal("unexpected Foo")
		}
		if got.Bar != 1234 {
			t.Fatal("unexpected Bar")
		}
	})
	t.Run("partial named data into struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		got, err := dec(Params(`{"bar": 1234}`))
		if err != nil {
			t.Fatal("must accept proper object. Err:", err)
		}
		if got.Foo != "" {
			t.Fatal("unexpected Foo")
		}
		if got.Bar != 1234 {
			t.Fatal("unexpected Bar")
		}
	})
	t.Run("list data into bigger struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		_, err := dec(Params(`["hello"]`))
		if err == nil {
			t.Fatal("cannot accept partial list")
		}
	})
	t.Run("list data into smaller struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		_, err := dec(Params(`["hello", 1234, "extra"]`))
		if err == nil {
			t.Fatal("cannot accept partial list")
		}
	})
	t.Run("list data into struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		got, err := dec(Params(`["hello", 1234]`))
		if err != nil {
			t.Fatal("must accept proper list. Err:", err)
		}
		if got.Foo != "hello" {
			t.Fatal("unexpected Foo")
		}
		if got.Bar != 1234 {
			t.Fatal("unexpected Bar")
		}
	})
	t.Run("list data into pointer struct", func(t *testing.T) {
		dec := ParamsDecoder[*TestObj]()
		got, err := dec(Params(`["hello", 1234]`))
		if err != nil {
			t.Fatal("must accept proper list. Err:", err)
		}
		if got.Foo != "hello" {
			t.Fatal("unexpected Foo")
		}
		if got.Bar != 1234 {
			t.Fatal("unexpected Bar")
		}
	})
	t.Run("list data into list", func(t *testing.T) {
		dec := ParamsDecoder[[]any]()
		got, err := dec(Params(`["hello", 1234]`))
		if err != nil {
			t.Fatal("must accept proper list. Err:", err)
		}
		if len(got) != 2 {
			t.Fatal("expected two values")
		}
		if got[0].(string) != "hello" {
			t.Fatal("unexpected Foo")
		}
		if got[1].(json.Number) != "1234" {
			t.Fatal("unexpected Bar")
		}
	})
	t.Run("empty list data into list", func(t *testing.T) {
		dec := ParamsDecoder[[]any]()
		got, err := dec(Params(`[]`))
		if err != nil {
			t.Fatal("must accept proper list. Err:", err)
		}
		if len(got) != 0 {
			t.Fatal("expected no values")
		}
		if got == nil {
			t.Fatal("expected 0-length but allocated slice")
		}
	})
	t.Run("object into list", func(t *testing.T) {
		dec := ParamsDecoder[[]any]()
		_, err := dec(Params(`{"foo": "hello"}`))
		if err == nil {
			t.Fatal("should not accept object into list params type")
		}
	})
	t.Run("int into struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		_, err := dec(Params(`1234`))
		if err == nil {
			t.Fatal("Cannot accept non-list/object into struct")
		}
	})
	t.Run("int into list", func(t *testing.T) {
		dec := ParamsDecoder[[]any]()
		_, err := dec(Params(`1234`))
		if err == nil {
			t.Fatal("Cannot accept non-list/object into list")
		}
	})
	t.Run("list-ish junk into struct", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		_, err := dec(Params(`["`))
		if err == nil {
			t.Fatal("Cannot accept junk into struct")
		}
	})
	t.Run("list-ish junk into list", func(t *testing.T) {
		dec := ParamsDecoder[[]any]()
		_, err := dec(Params(`["`))
		if err == nil {
			t.Fatal("Cannot accept junk into list")
		}
	})
	t.Run("string list into int list", func(t *testing.T) {
		dec := ParamsDecoder[[]int]()
		_, err := dec(Params(`["hello"]`))
		if err == nil {
			t.Fatal("Cannot accept junk into list")
		}
	})
	t.Run("object value number into string field", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		_, err := dec(Params(`{"foo": 9000, "bar": 1234}`))
		if err == nil {
			t.Fatal("must not accept number into string field")
		}
	})
	t.Run("list value number into string field", func(t *testing.T) {
		dec := ParamsDecoder[TestObj]()
		_, err := dec(Params(`[9000, 1234]`))
		if err == nil {
			t.Fatal("must not accept number into string field")
		}
	})
}
