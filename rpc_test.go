package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRPC(t *testing.T) {
	valid := []string{
		`{"jsonrpc": "2.0", "method": "subtract", "params": [42,23], "id": 1}`,
		`{"jsonrpc": "2.0", "result": 19, "id": 1}`,
		`{"jsonrpc": "2.0", "method": "subtract", "params": [23,42], "id": 2}`,
		`{"jsonrpc": "2.0", "result": -19, "id": 2}`,
		`{"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend":23,"minuend":42}, "id": 3}`,
		`{"jsonrpc": "2.0", "result": 19, "id": 3}`,
		`{"jsonrpc": "2.0", "method": "subtract", "params": {"minuend":42,"subtrahend":23}, "id": 4}`,
		`{"jsonrpc": "2.0", "result": 19, "id": 4}`,
		`{"jsonrpc": "2.0", "method": "update", "params": [1,2,3,4,5]}`,
		`{"jsonrpc": "2.0", "method": "foobar"}`,
		`{"jsonrpc": "2.0", "method": "foobar", "id": "1"}`,
		`{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "1"}`,
		`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`,
		`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		`{"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"}`,
		`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`,
		`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		`{"jsonrpc": "2.0", "result": null, "id": 1}`,              // null result
		`{"jsonrpc": "2.0", "result": 19, "error": null, "id": 1}`, // null error is absent
		`{"jsonrpc": "2.0", "method": "foobar", "params": null, "id": 1}`,
		`{"jsonrpc": "2.0", "method": "foobar", "id": -1}`,
		`{"jsonrpc": "2.0", "method": "foobar", "id": 0}`,
		`{"jsonrpc": "2.0", "method": "foobar", "id": null}`,
		`{"jsonrpc": "2.0", "method": "foobar", "id": "with space"}`,
	}
	for i, tc := range valid {
		t.Run(fmt.Sprintf("valid_%d", i), func(t *testing.T) {
			var m Message
			err := json.Unmarshal([]byte(tc), &m)
			if err != nil {
				t.Fatalf("failed to decode: %v\n data: %s\n", err, tc)
			}
			out, err := json.Marshal(&m)
			if err != nil {
				t.Fatalf("failed to re-encode: %v\n", err)
			}
			var m2 Message
			err = json.Unmarshal(out, &m2)
			if err != nil {
				t.Fatalf("failed to re-decode: %v\n data: %s\n", err, tc)
			}
			if m.Request != nil {
				if m2.Request == nil {
					t.Fatal("lost request")
				}
				if m.Request.Method != m2.Request.Method {
					t.Fatalf("different method: %s <> %s", m.Request.Method, m2.Request.Method)
				}
				if string(m.Request.Params) != string(m2.Request.Params) {
					t.Fatalf("different params: %s <> %s", string(m.Request.Params), string(m2.Request.Params))
				}
			} else {
				if m2.Request != nil {
					t.Fatal("unexpected request")
				}
			}
			if m.Response != nil {
				if m2.Response == nil {
					t.Fatal("lost response")
				}
				if m.Response.Result != nil {
					if m2.Response.Result == nil {
						t.Fatal("lost result")
					}
					if string(*m.Response.Result) != string(*m2.Response.Result) {
						t.Fatalf("different result: %s <> %s", string(*m.Response.Result), string(*m2.Response.Result))
					}
				} else {
					if m2.Response.Result != nil {
						t.Fatal("unexpected result")
					}
				}
				if m.Response.Error != nil {
					if m2.Response.Error == nil {
						t.Fatal("lost error")
					}
					if m.Response.Error.Code != m2.Response.Error.Code {
						t.Fatalf("different error code: %d <> %d", m.Response.Error.Code, m2.Response.Error.Code)
					}
					if m.Response.Error.Message != m2.Response.Error.Message {
						t.Fatalf("different error message: %s <> %s", m.Response.Error.Message, m2.Response.Error.Message)
					}
					if string(m.Response.Error.Data) != string(m2.Response.Error.Data) {
						t.Fatalf("different error data: %s <> %s", string(m.Response.Error.Data), string(m2.Response.Error.Data))
					}
				} else {
					if m2.Response.Error != nil {
						t.Fatal("unexpected error")
					}
				}
			} else {
				if m2.Response != nil {
					t.Fatal("unexpected response")
				}
			}
		})
	}
	invalid := []string{
		`{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`, // invalid JSON
		`{"jsonrpc": "2.0", "method": 1, "params": "bar"}`,             // invalid method type
		`{"jsonrpc": "2.0", "method": "foobar", "params": "bar"}`,      // invalid params type
		`{"jsonrpc": "1.0", "method": "foobar", "params": []}`,         // invalid version
		`{"jsonrpc": "2.0"}`, // not a request or response
		`{"jsonrpc": "2.0", "result": 19, "error": {"code": -32601, "message": "Method not found"}, "id": 1}`,
		`{"jsonrpc": "2.0", "result": null, "error": {"code": -32601, "message": "Method not found"}, "id": 1}`,
		`{"jsonrpc": "2.0", "id": 1}`,                                   // neither result nor error
		`{"jsonrpc": "2.0", "result": 19}`,                              // response without ID
		`{"jsonrpc": "2.0", "params": [1], "id": 1}`,                    // params without method
		`{"jsonrpc": "2.0", "params": [1], "result": 19, "id": 1}`,      // params in a response
		`{"jsonrpc": "2.0", "method": "foobar", "result": 19, "id": 1}`, // request and response
		`{"jsonrpc": "2.0", "method": "foobar", "id": 1.5}`,             // fractional ID
		`{"jsonrpc": "2.0", "method": "foobar", "id": 1e3}`,             // exponent ID
		`{"jsonrpc": "2.0", "method": "foobar", "id": true}`,            // bool ID
		`{"jsonrpc": "2.0", "method": "foobar", "id": [1]}`,             // array ID
	}
	for i, tc := range invalid {
		t.Run(fmt.Sprintf("invalid_%d", i), func(t *testing.T) {
			var m Message
			err := json.Unmarshal([]byte(tc), &m)
			if err == nil {
				t.Errorf("expected error, but got none, for data: %s\n", tc)
			}
		})
	}
}

func TestRawIDIsValid(t *testing.T) {
	longString := RawID(`"` + strings.Repeat("a", MaxIDLength-2) + `"`)
	valid := []RawID{"", NullID, "0", "-0", "1", "-1", "1234567890", `""`, `"a"`, `"a b"`, `"\u0061"`, longString}
	for _, id := range valid {
		if !id.IsValid() {
			t.Errorf("expected valid ID: %s", id)
		}
	}
	invalid := []RawID{
		"-", "01", "-01", "1.5", "1e3", "+1", " 1", "1 ", "true", "nil", "[1]", "{}",
		`"`, `"a`, `a"`, `"a" `, ` "a"`, `"a" "b"`,
		longString[:len(longString)-1] + `a"`, // one byte too long
		RawID(strings.Repeat("1", MaxIDLength+1)),
	}
	for _, id := range invalid {
		if id.IsValid() {
			t.Errorf("expected invalid ID: %s", id)
		}
		if _, err := id.MarshalJSON(); err == nil {
			t.Errorf("expected encoding error for invalid ID: %s", id)
		}
	}
}

func TestNullID(t *testing.T) {
	var m Message
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"foobar","id":null}`), &m); err != nil {
		t.Fatal(err)
	}
	if m.ID != NullID {
		t.Fatalf("expected null ID, got %q", m.ID)
	}
	if m.ID.IsNotification() {
		t.Fatal("a request with null ID is not a notification")
	}
	out, err := json.Marshal(m.Respond(true))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `{"result":true,"id":null,"jsonrpc":"2.0"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"foobar"}`), &m); err != nil {
		t.Fatal(err)
	}
	if !m.ID.IsNotification() {
		t.Fatal("a request without ID is a notification")
	}
}

func TestNullResult(t *testing.T) {
	var m Message
	if err := json.Unmarshal([]byte(`{"jsonrpc": "2.0", "result": null, "id": 1}`), &m); err != nil {
		t.Fatal(err)
	}
	if m.Response == nil || m.Response.Result == nil {
		t.Fatal("expected a null result")
	}
	if got := string(*m.Response.Result); got != "null" {
		t.Fatalf("expected null result, got %s", got)
	}
	out, err := json.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `{"result":null,"id":1,"jsonrpc":"2.0"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	// A result without any data is encoded as null too, like a nil json.RawMessage.
	empty := json.RawMessage(nil)
	out, err = json.Marshal(&Message{Response: &Response{Result: &empty}, ID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `{"result":null,"id":1,"jsonrpc":"2.0"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	resp := Respond(nil)
	if resp.Result == nil || string(*resp.Result) != "null" {
		t.Fatal("expected Respond(nil) to have a null result")
	}
}

func TestMarshalInvalid(t *testing.T) {
	result := json.RawMessage("1")
	invalid := map[string]Message{
		"neither":                          {ID: "1"},
		"both":                             {Request: &Request{Method: "foo"}, Response: &Response{Result: &result}, ID: "1"},
		"response without ID":              {Response: &Response{Result: &result}},
		"response without result or error": {Response: &Response{}, ID: "1"},
		"response with result and error": {
			Response: &Response{Result: &result, Error: ConstErrorObj(InternalError)},
			ID:       "1",
		},
		"invalid ID": {Request: &Request{Method: "foo"}, ID: "1.5"},
	}
	for name, m := range invalid {
		t.Run(name, func(t *testing.T) {
			out, err := json.Marshal(&m)
			if err == nil {
				t.Fatalf("expected error, got %s", out)
			}
			if out != nil {
				t.Fatalf("expected no data with the error, got %s", out)
			}
		})
	}
}

func TestMarshalValue(t *testing.T) {
	m := Message{Request: &Request{Method: "foo", Params: Params(`[1]`)}, ID: "1"}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `{"method":"foo","params":[1],"id":1,"jsonrpc":"2.0"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestParamsCopy(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","method":"foo","params":[1,2],"id":1}`)
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for i := range data {
		data[i] = ' '
	}
	if got := string(m.Params); got != "[1,2]" {
		t.Fatalf("params changed with the input buffer: %q", got)
	}
}

func TestV2MarshalText(t *testing.T) {
	out, err := V2{}.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	out[0] = '3'
	out, err = json.Marshal(&Message{Request: &Request{Method: "foo"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `{"method":"foo","jsonrpc":"2.0"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestMethodName(t *testing.T) {
	req := Message{Request: &Request{Method: "foo"}, ID: "1"}
	if got := req.MethodName(); got != "foo" {
		t.Fatalf("expected method foo, got %q", got)
	}
	resp := req.Respond(1)
	if got := resp.MethodName(); got != "" {
		t.Fatalf("expected no method, got %q", got)
	}
}

func TestRespond(t *testing.T) {
	req := &Message{Request: &Request{Method: "foo"}, ID: "1"}
	resp, err := req.RespondSuccess(19)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != req.ID || resp.Result == nil || string(*resp.Result) != "19" {
		t.Fatalf("unexpected response: %v", resp)
	}
	if _, err := resp.RespondSuccess(19); err == nil {
		t.Fatal("expected error when responding to a response")
	}
	if _, err := req.RespondSuccess(make(chan int)); err == nil {
		t.Fatal("expected encoding error")
	}
	internal := req.Respond(make(chan int))
	if internal.Error == nil || internal.Error.Code != InternalError.Code() || internal.Result != nil {
		t.Fatalf("expected internal error response, got %v", internal.Response)
	}
	failed := req.RespondErr(ConstErrorObj(MethodNotFound))
	if failed.Error == nil || failed.Error.Code != MethodNotFound.Code() {
		t.Fatalf("expected method not found response, got %v", failed.Response)
	}
	mustPanic(t, "Respond to response", func() { resp.Respond(19) })
	mustPanic(t, "RespondErr to response", func() { resp.RespondErr(ConstErrorObj(InternalError)) })
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	fn()
}

func TestErrorObject(t *testing.T) {
	var err error = &ErrorObject{Code: 4001, Message: "User Rejected Request"}
	if got, want := err.Error(), "JSON-RPC error 4001: User Rejected Request"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	var obj *ErrorObject
	if !errors.As(fmt.Errorf("wrapped: %w", err), &obj) || obj.Code != 4001 {
		t.Fatal("expected to find the error object")
	}
	if got := AnnotatedErrorObj(InternalError, nil); got.Code != InternalError.Code() || got.Message != "Internal error" {
		t.Fatalf("expected plain error object for nil error, got %v", got)
	}
	if got, want := AnnotatedErrorObj(InternalError, errors.New("oops")).Message, "Internal error: oops"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
