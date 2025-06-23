package httprequest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/form/v4"
	"github.com/go-playground/validator/v10"
)

var (
	// DefaultFormDecoder is used to decode url.Values in the DecodeForm.
	DefaultFormDecoder = form.NewDecoder()

	// DefaultValidator is used both for JSON and Form validation.
	DefaultValidator = validator.New(validator.WithRequiredStructEnabled())
)

type DecodeJSONOptions struct {
	// Validator is the validator to use for validation.
	Validator *validator.Validate
}

// DecodeJSON converts a JSON body from the incoming request r into the struct v.
//
// The decoded struct is validated with the Validator.
//
// Use validation tags from the github.com/go-playground/validator/v10 for
// setting validation.
func DecodeJSON[T any](r *http.Request, opts *DecodeJSONOptions) (*T, error) {
	ctx := r.Context()

	validator := DefaultValidator
	if opts != nil {
		if opts.Validator != nil {
			validator = opts.Validator
		}
	}

	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("foundations/httprequest: unable to decode JSON: %w", err)
	}

	if err := validator.StructCtx(ctx, v); err != nil {
		return nil, err
	}

	return &v, nil
}

type DecodeFormOptions struct {
	// Validator is the validator to use for validation.
	Validator *validator.Validate

	// Decoder is the decoder to use for decoding.
	Decoder *form.Decoder
}

// DecodeForm converts a url.Values (including form values) from the incoming
// request r into the struct v.
//
// The decoded struct is validated with the Validator.
//
// Use validation tags from the github.com/go-playground/validator/v10 for
// setting validation.
func DecodeForm[T any](r *http.Request, opts *DecodeFormOptions) (*T, error) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("foundations/httprequest: unable to parse form request: %w", err)
	}

	decoder := DefaultFormDecoder
	validator := DefaultValidator
	if opts != nil {
		if opts.Decoder != nil {
			decoder = opts.Decoder
		}

		if opts.Validator != nil {
			validator = opts.Validator
		}
	}

	var v T
	values := r.Form

	if err := decoder.Decode(&v, values); err != nil {
		return nil, err
	}

	if err := validator.StructCtx(ctx, &v); err != nil {
		return nil, err
	}

	return &v, nil
}
