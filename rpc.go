// Package jsonrpc2 provides minimal types for JSON-RPC 2.0 messages,
// for servers and proxies that route and manage messages themselves.
// The 2 in the name is the JSON-RPC protocol version, not a module major version.
//
// A [Message] is either a request, or a response with exactly one of a result and an error.
// Decoding validates the message structure, and so does encoding, so an invalid message is never written.
package jsonrpc2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// RawID is a JSON-RPC message ID in its exact JSON encoding: e.g. 1, "abc" (including the quotes), or null.
// The empty RawID is the absent ID of a notification.
//
// Keeping the encoding makes an ID comparable, usable as map key, and echoed exactly as received,
// without decoding it into an interface value. IDs are equal if their encodings are equal:
// "a" and "\u0061" are different IDs, and so are 1 and "1".
//
// A valid ID is a string, an integer, or null, of at most [MaxIDLength] bytes.
// Fractions and exponents are rejected: the spec discourages them, since they may not survive a round trip.
type RawID string

// MaxIDLength bounds the length of an encoded ID, since IDs are retained (e.g. as map keys) while requests are pending.
// It fits common IDs, such as integers, UUIDs, and 0x-prefixed 32-byte hex strings, with room to spare.
const MaxIDLength = 256

// NullID is the null ID. A response to a request whose ID could not be determined
// (a parse error, or an invalid request) has the null ID.
// A request with the null ID is not a notification: it is answered with the null ID, although the spec
// discourages this, since the answer is indistinguishable from such an error response.
const NullID RawID = "null"

var _ json.Marshaler = RawID("")
var _ json.Unmarshaler = (*RawID)(nil)

// IsValid reports whether id is absent (a notification), or a valid ID: see [RawID].
// Surrounding whitespace, booleans, fractions, exponents, objects and arrays are not valid.
func (id RawID) IsValid() bool {
	return id.check() == nil
}

func (id RawID) check() error {
	switch {
	case len(id) == 0, id == NullID:
		return nil
	case len(id) > MaxIDLength:
		return fmt.Errorf("ID too long: %d bytes, max %d", len(id), MaxIDLength)
	case id[0] == '"':
		if id[len(id)-1] == '"' && json.Valid([]byte(id)) {
			return nil
		}
	case isInteger(string(id)):
		return nil
	}
	return fmt.Errorf("invalid ID: %q", string(id))
}

// isInteger reports whether s is a JSON integer: an optional minus sign, and digits without leading zero.
func isInteger(s string) bool {
	if len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	if len(s) == 0 {
		return false
	}
	if s[0] == '0' {
		return len(s) == 1
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (id RawID) IsNotification() bool {
	return len(id) == 0
}

func (id RawID) MarshalJSON() ([]byte, error) {
	if err := id.check(); err != nil {
		return nil, err
	}
	return []byte(id), nil
}

func (id *RawID) UnmarshalJSON(data []byte) error {
	if id == nil {
		return errors.New("cannot unmarshal into nil RawID")
	}
	if err := RawID(data).check(); err != nil {
		return err
	}
	*id = RawID(data)
	return nil
}

func (id RawID) String() string {
	return string(id)
}

func (id RawID) Equal(other RawID) bool {
	return id == other
}

// Params in JSON-RPC 2.0 can be either ordered (an array) or named (an object).
type Params json.RawMessage

func (p Params) MarshalJSON() ([]byte, error) {
	return p, nil
}

// UnmarshalJSON copies an array or object. A null is decoded as omitted params (nil),
// since clients send it for methods without params.
func (p *Params) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("invalid JSON, empty input")
	}
	if string(data) == "null" {
		*p = nil
		return nil
	}
	if data[0] != '[' && data[0] != '{' {
		return errors.New("JSON-RPC params must be list or map")
	}
	// The data may be a buffer of the caller (e.g. of a json.Decoder) that is reused after decoding.
	*p = bytes.Clone(data)
	return nil
}

func (p Params) Count() int {
	var x []json.RawMessage
	if err := json.Unmarshal(p, &x); err == nil {
		return len(x)
	}
	var y map[string]json.RawMessage
	if err := json.Unmarshal(p, &y); err == nil {
		return len(y)
	}
	return 0
}

type Request struct {
	Method string `json:"method"`
	// Can be a map or a list
	Params Params `json:"params,omitempty"`
}

// ErrorObject is the error of a response. It implements error.
type ErrorObject struct {
	Code    int64           `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

var _ error = (*ErrorObject)(nil)

// Error returns the code and message. It leaves out Data, which may be large.
func (e *ErrorObject) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Response is the answer to a request: a valid response has exactly one of Result and Error.
// A null result is a non-nil Result that holds null: decoding keeps "result": null,
// and a Result that holds no data at all is encoded as null.
type Response struct {
	Result *json.RawMessage `json:"result,omitempty"`
	Error  *ErrorObject     `json:"error,omitempty"`
}

// V2 is a zero-size constant type, for encoding/decoding JSON-RPC messages:
// it validates the JSON-RPC version, without allocating it as Go string in every message.
type V2 struct{}

func (V2) MarshalText() ([]byte, error) {
	// A new slice every time: callers may modify it.
	return []byte("2.0"), nil
}

func (*V2) UnmarshalText(data []byte) error {
	if len(data) != 3 || data[0] != '2' || data[1] != '.' || data[2] != '0' {
		return fmt.Errorf("invalid JSON RPC version: %q", string(data))
	}
	return nil
}

// Message is a JSON-RPC request or response: exactly one of Request and Response is set.
// A request without ID is a notification, which must not be answered.
//
// A field of the embedded pointers, such as m.Method, panics if its pointer is nil;
// MethodName is safe for any message.
type Message struct {
	*Request
	*Response
	ID RawID // "notification" messages do not require an ID
}

// MethodName returns the method of a request, or "" if m is not a request.
func (m *Message) MethodName() string {
	if m.Request == nil {
		return ""
	}
	return m.Request.Method
}

// wireMessage is the JSON-RPC 2.0 encoding of a Message, with a pointer or raw member to detect presence.
// Result is not a pointer: encoding/json passes an explicit null to a json.RawMessage,
// but decodes it as a nil pointer, and "result": null is a result.
// The member order is that of earlier versions, which encoded the embedded Request and Response in turn.
type wireMessage struct {
	Method  *string         `json:"method,omitempty"`
	Params  Params          `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
	ID      RawID           `json:"id,omitempty"` // "notification" messages do not require an ID
	JSONRPC V2              `json:"jsonrpc"`
}

func (m *Message) check() error {
	switch {
	case m.Request != nil && m.Response != nil:
		return errors.New("message must be either a request or response, but not both")
	case m.Request != nil:
		return nil
	case m.Response == nil:
		return errors.New("message must be either a request or response")
	case m.ID.IsNotification():
		return errors.New("responses cannot be notifications")
	case m.Response.Result != nil && m.Response.Error != nil:
		return errors.New("response must have either a result or an error, but not both")
	case m.Response.Result == nil && m.Response.Error == nil:
		return errors.New("response must have either a result or an error")
	}
	return nil
}

// MarshalJSON encodes a valid message, and returns an error for an invalid one.
// It has a value receiver, so that a Message value is encoded as a message too, not only a pointer.
func (m Message) MarshalJSON() ([]byte, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	out := wireMessage{ID: m.ID}
	if m.Request != nil {
		out.Method = &m.Request.Method
		out.Params = m.Request.Params
	} else {
		out.Error = m.Response.Error
		if m.Response.Result != nil {
			out.Result = *m.Response.Result
			if len(out.Result) == 0 { // encoded as null, like a nil json.RawMessage
				out.Result = json.RawMessage("null")
			}
		}
	}
	return json.Marshal(&out)
}

// UnmarshalJSON decodes and validates a message. It is lenient where that is unambiguous:
// "params": null is omitted params, and "error": null is an absent error.
// A missing "jsonrpc" member is accepted too.
func (m *Message) UnmarshalJSON(data []byte) error {
	var in wireMessage
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	out := Message{ID: in.ID}
	if in.Method != nil {
		out.Request = &Request{Method: *in.Method, Params: in.Params}
	} else if in.Params != nil {
		return errors.New("message has params, but no method")
	}
	if in.Result != nil || in.Error != nil {
		out.Response = &Response{Error: in.Error}
		if in.Result != nil {
			out.Response.Result = &in.Result
		}
	}
	if err := out.check(); err != nil {
		return err
	}
	*m = out
	return nil
}

// RespondSuccess returns a response to the request m, with data encoded as JSON as result.
// It returns an error if m is not a request, or if data cannot be encoded.
func (m *Message) RespondSuccess(data any) (*Message, error) {
	if m.Request == nil {
		return nil, fmt.Errorf("cannot respond to a message that is not a request: %s", m.ID)
	}
	resp, err := RespondSuccess(data)
	if err != nil {
		return nil, err
	}
	return &Message{
		Request:  nil,
		Response: resp,
		ID:       m.ID,
	}, nil
}

// RespondSuccess returns a response with data encoded as JSON as result.
// Unlike [Respond], it returns an encoding error, so the caller may choose the error response.
func RespondSuccess(data any) (*Response, error) {
	x, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode response: %w", err)
	}
	result := json.RawMessage(x)
	return &Response{
		Result: &result,
		Error:  nil,
	}, nil
}

// Respond returns a response with data encoded as JSON as result.
// If data cannot be encoded, it returns an [InternalError] response that describes the encoding error.
func Respond(data any) *Response {
	resp, err := RespondSuccess(data)
	if err != nil {
		return &Response{
			Result: nil,
			Error:  AnnotatedErrorObj(InternalError, err),
		}
	}
	return resp
}

func ConstErrorObj(c ErrorConst) *ErrorObject {
	return &ErrorObject{
		Code:    c.Code(),
		Message: c.Message(),
		Data:    nil,
	}
}

// AnnotatedErrorObj returns an error object with the message of c, followed by err.
// A nil err adds nothing, as in [ConstErrorObj].
func AnnotatedErrorObj(c ErrorConst, err error) *ErrorObject {
	if err == nil {
		return ConstErrorObj(c)
	}
	return &ErrorObject{
		Code:    c.Code(),
		Message: c.Message() + ": " + err.Error(),
		Data:    nil,
	}
}

// Respond returns a response to the request m, like the package-level [Respond]:
// if data cannot be encoded, the response is an [InternalError].
// It panics if m is not a request: whether to respond is up to the caller, not the input.
func (m *Message) Respond(data any) *Message {
	if m.Request == nil {
		panic(fmt.Errorf("cannot respond to a message that is not a request: %s", m.ID))
	}
	return &Message{
		Request:  nil,
		Response: Respond(data),
		ID:       m.ID,
	}
}

// RespondErr returns an error response to the request m.
// It panics if m is not a request: whether to respond is up to the caller, not the input.
func (m *Message) RespondErr(errObj *ErrorObject) *Message {
	if m.Request == nil {
		panic(fmt.Errorf("cannot respond to a message that is not a request: %s", m.ID))
	}
	return &Message{
		Request: nil,
		Response: &Response{
			Result: nil,
			Error:  errObj,
		},
		ID: m.ID,
	}
}
