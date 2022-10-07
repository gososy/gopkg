package binding

import (
	"net/http"
	"net/url"

	"github.com/gososy/gopkg/errors"

	"github.com/gososy/gopkg/encoding"
	"github.com/gososy/gopkg/encoding/form"
)

// BindQuery bind vars parameters to target.
func BindQuery(vars url.Values, target interface{}) error {
	if err := encoding.GetCodec(form.Name).Unmarshal([]byte(vars.Encode()), target); err != nil {
		return errors.BadRequest("CODEC", err.Error())
	}
	return nil
}

// BindForm bind form parameters to target.
func BindForm(req *http.Request, target interface{}) error {
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := encoding.GetCodec(form.Name).Unmarshal([]byte(req.Form.Encode()), target); err != nil {
		return errors.BadRequest("CODEC", err.Error())
	}
	return nil
}
