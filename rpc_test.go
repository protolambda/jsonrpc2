package jsonrpc

import (
	"encoding/json"
	"fmt"
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
