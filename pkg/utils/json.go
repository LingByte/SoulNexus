package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// EnableDecoderUseNumber is used to call the UseNumber method on the JSON
// Decoder instance. UseNumber causes the Decoder to unmarshal a number into an
// any as a Number instead of as a float64.
var EnableDecoderUseNumber = false

// EnableDecoderDisallowUnknownFields is used to call the DisallowUnknownFields method
// on the JSON Decoder instance. DisallowUnknownFields causes the Decoder to
// return an error when the destination is a struct and the input contains object
// keys which do not match any non-ignored, exported fields in the destination.
var EnableDecoderDisallowUnknownFields = false

type Decoder interface {
	Decode(v any) error
}

type Encoder interface {
	Encode(v any) error
}

var (
	Marshal                                 = json.Marshal
	Unmarshal                               = json.Unmarshal
	MarshalIndent                           = json.MarshalIndent
	NewDecoder    func(r io.Reader) Decoder = useJsonNewDecoder
	NewEncoder    func(w io.Writer) Encoder = useJsonNewEncoder
)

func init() {
	binding.JSON = CarrotJsonBinding{}
}

var (
	JsonContentType = []string{"application/json; charset=utf-8"}
)

type CarrotJSON struct {
	Data any
}

type CarrotJsonBinding struct{}

func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}

func RenderJSON(c *gin.Context, code int, obj any) {
	c.Render(code, CarrotJSON{Data: obj})
}

// Render (JSON) writes data with custom ContentType.
func (r CarrotJSON) Render(w http.ResponseWriter) error {
	return WriteJSON(w, r.Data)
}

// WriteContentType (JSON) writes JSON ContentType.
func (r CarrotJSON) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, JsonContentType)
}

// WriteJSON marshals the given interface object and writes it with custom ContentType.
func WriteJSON(w http.ResponseWriter, obj any) error {
	writeContentType(w, JsonContentType)
	jsonBytes, err := Marshal(obj)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonBytes)
	return err
}

func (CarrotJsonBinding) Name() string {
	return "json"
}

func (CarrotJsonBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	return decodeJSON(req.Body, obj)
}

func (CarrotJsonBinding) BindBody(body []byte, obj any) error {
	return decodeJSON(bytes.NewReader(body), obj)
}

func decodeJSON(r io.Reader, obj any) error {
	if err := NewDecoder(r).Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

func validate(obj any) error {
	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(obj)
}

func useJsonNewDecoder(r io.Reader) Decoder {
	decoder := json.NewDecoder(r)
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder
}

func useJsonNewEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}
