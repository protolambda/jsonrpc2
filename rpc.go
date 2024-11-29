package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
)

type RawID string

// 32 hex encoded bytes, with 0x prefix, with quotes, is the maximum length
const maxIDLength = 32*2 + 2 + 2

var _ json.Marshaler = RawID("")
var _ json.Unmarshaler = (*RawID)(nil)

// IsValid checks if the JSON RPC message identifier is valid.
// if not null, a string, or an integer number, then invalid.
// I.e. leading whitespace is not valid, true/false are not valid, floats are not valid, maps and arrays are not valid.
func (id RawID) IsValid() bool {
	if len(id) == 0 { // notifications do not have an ID
		return true
	}
	if len(id) > maxIDLength {
		return false
	}
	if id == "null" {
		return true
	}
	if id[0] == '"' { // any string without whitespace
		return id[len(id)-1] == '"' && json.Valid([]byte(id))
	}
	// any integer number, but no "1e9" etc.
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (id RawID) IsNotification() bool {
	return len(id) == 0
}

func (id RawID) MarshalJSON() ([]byte, error) {
	if !id.IsValid() {
		return nil, fmt.Errorf("invalid ID: %x", []byte(id))
	}
	return []byte(id), nil
}

func (id *RawID) UnmarshalJSON(data []byte) error {
	if id == nil {
		return errors.New("cannot unmarshal into nil RawID")
	}
	*id = RawID(data)
	if !id.IsValid() {
		return fmt.Errorf("invalid ID: %x", data)
	}
	return nil
}

func (id RawID) String() string {
	return string(id)
}

func (id RawID) Equal(other RawID) bool {
	return id == other
}

// Params in JSON-RPC 2.0 can be either ordered or named.
type Params json.RawMessage

func (p Params) MarshalJSON() ([]byte, error) {
	return p, nil
}

func (p *Params) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return errors.New("invalid JSON, empty input")
	}
	if data[0] != '[' && data[0] != '{' {
		return errors.New("JSON-RPC params must be list or map")
	}
	*p = data
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

type ErrorObject struct {
	Code    int64           `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type Response struct {
	Result *json.RawMessage `json:"result,omitempty"`
	Error  *ErrorObject     `json:"error,omitempty"`
}

// V2 is a zero-cost constant type, for encoding/decoding JSON-RPC messages:
// it validates the JSON-RPC version, without allocating it as Go string in every message.
type V2 struct{}

var v2Bytes = []byte("2.0")

func (V2) MarshalText() ([]byte, error) {
	return v2Bytes, nil
}

func (*V2) UnmarshalText(data []byte) error {
	if len(data) != 3 || data[0] != '2' || data[1] != '.' || data[2] != '0' {
		return fmt.Errorf("invalid JSON RPC version: %q", string(data))
	}
	return nil
}

type Message struct {
	*Request
	*Response
	ID RawID // "notification" messages do not require an ID
}

type jsonMessage struct {
	*Request
	*Response
	ID      RawID `json:"id,omitempty"` // "notification" messages do not require an ID
	JSONRPC V2    `json:"jsonrpc"`
}

func (m *jsonMessage) Check() error {
	if m.Response != nil {
		if m.Request != nil {
			return errors.New("message must be either a request or response, but not both")
		}
		if m.ID.IsNotification() {
			return errors.New("responses cannot be notifications")
		}
	} else {
		if m.Request == nil {
			return errors.New("message must be either a request or response")
		}
	}
	return nil
}

func (m *Message) MarshalJSON() ([]byte, error) {
	out := jsonMessage{
		Request:  m.Request,
		Response: m.Response,
		ID:       m.ID,
		JSONRPC:  V2{},
	}
	data, err := json.Marshal(&out)
	if err != nil {
		return data, err
	}
	return data, out.Check()
}

func (m *Message) UnmarshalJSON(data []byte) error {
	var dest jsonMessage
	err := json.Unmarshal(data, &dest)
	if err != nil {
		return err
	}
	if err := dest.Check(); err != nil {
		return err
	}
	*m = Message{
		Request:  dest.Request,
		Response: dest.Response,
		ID:       dest.ID,
	}
	return nil
}

func (m *Message) RespondSuccess(data any) (*Message, error) {
	if m.Response != nil {
		return nil, fmt.Errorf("cannot respond to a response: %s", m.ID)
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

func AnnotatedErrorObj(c ErrorConst, err error) *ErrorObject {
	return &ErrorObject{
		Code:    c.Code(),
		Message: c.Message() + ": " + err.Error(),
		Data:    nil,
	}
}

func (m *Message) Respond(data any) *Message {
	if m.Response != nil {
		panic(fmt.Errorf("cannot respond to a response: %s", m.ID))
	}
	return &Message{
		Request:  nil,
		Response: Respond(data),
		ID:       m.ID,
	}
}

func (m *Message) RespondErr(errObj *ErrorObject) *Message {
	if m.Response != nil {
		panic(fmt.Errorf("cannot respond to a response: %s", m.ID))
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
