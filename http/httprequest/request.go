package httprequest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/form/v4"
	"github.com/go-playground/validator/v10"
)

var (
	// FormDecoder is used to decode url.Values in the DecodeForm.
	FormDecoder = form.NewDecoder()

	// Validator is used both for JSON and Form validation.
	Validator = validator.New(validator.WithRequiredStructEnabled())
)

// DecodeJSON converts a JSON body from the incoming request r into the struct v.
//
// The decoded struct is validated with the Validator.
//
// Use validation tags from the github.com/go-playground/validator/v10 for
// setting validation.
func DecodeJSON[T any](r *http.Request) (*T, error) {
	ctx := r.Context()

	var v *T
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return nil, fmt.Errorf("http/request: unable to decode JSON: %w", err)
	}

	if err := Validator.StructCtx(ctx, v); err != nil {
		return nil, err
	}

	return v, nil
}

// DecodeForm converts a url.Values (including form values) from the incoming
// request r into the struct v.
//
// The decoded struct is validated with the Validator.
//
// Use validation tags from the github.com/go-playground/validator/v10 for
// setting validation.
func DecodeForm[T any](r *http.Request) (*T, error) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("http/request: unable to parse form request: %w", err)
	}

	var v T
	values := r.Form

	if err := FormDecoder.Decode(&v, values); err != nil {
		return nil, err
	}

	if err := Validator.StructCtx(ctx, &v); err != nil {
		return nil, err
	}

	return &v, nil
}
