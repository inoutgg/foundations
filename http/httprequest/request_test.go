package httprequest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

type NestedField struct {
	BoolField bool `form:"bool_field" json:"bool_field"` //nolint:tagliatelle // test
}

type Body struct {
	StringField string      `form:"string_field" json:"string_field" validate:"required"` //nolint:tagliatelle // test
	IntField    int         `form:"int_field"    json:"int_field"`                        //nolint:tagliatelle // test
	NestedField NestedField `form:"nested_field" json:"nested_field"`                     //nolint:tagliatelle // test
}

func encodeJSON(t *testing.T, val any) []byte {
	t.Helper()

	bytes, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}

	return bytes
}

func requestJSON(t *testing.T, val any) *http.Request {
	t.Helper()

	return httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(encodeJSON(t, val)))
}

func requestForm(val url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(val.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return r
}

func TestJSON(t *testing.T) {
	t.Run("it works", func(t *testing.T) {
		expected := &Body{
			StringField: "string_test",
			IntField:    1,
			NestedField: NestedField{BoolField: true},
		}
		r := requestJSON(t, expected)

		body, err := DecodeJSON[Body](r, nil)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(body, expected) {
			t.Fatal(fmt.Errorf("expected body to equal %+v", expected))
		}
	})

	t.Run("it handles error correctly", func(t *testing.T) {
		body, err := DecodeJSON[Body](
			httptest.NewRequest(http.MethodPost, "/", nil),
			nil,
		)

		if err == nil || body != nil {
			t.Fatal(fmt.Errorf("expected body to be empty got %+v", body))
		}
	})

	t.Run("it validates result", func(t *testing.T) {
		r := requestJSON(t, Body{})

		_, err := DecodeJSON[Body](r, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}

		var validationErr validator.ValidationErrors

		ok := errors.As(err, &validationErr)
		if !ok {
			t.Fatal(fmt.Errorf("expected validation error got %T", err))
		}

		if validationErr[0].Field() != "StringField" {
			t.Fatal(fmt.Errorf("expected field to be StringField got %s", validationErr[0].Field()))
		}
	})
}

func TestForm(t *testing.T) {
	t.Run("it works", func(t *testing.T) {
		r := requestForm(url.Values{
			"string_field":            {"string_test"},
			"int_field":               {"1"},
			"nested_field.bool_field": {"true"},
		})

		body, err := DecodeForm[Body](r, nil)
		if err != nil {
			t.Fatal(err)
		}

		expected := &Body{
			StringField: "string_test",
			IntField:    1,
			NestedField: NestedField{BoolField: true},
		}

		if !reflect.DeepEqual(body, expected) {
			t.Fatal(fmt.Errorf("expected body to equal %+v", expected))
		}
	})

	t.Run("it handles error correctly", func(t *testing.T) {
		body, err := DecodeForm[Body](
			httptest.NewRequest(http.MethodPost, "/", nil),
			nil,
		)

		if err == nil || body != nil {
			t.Fatal(fmt.Errorf("expected body to be empty got %+v", body))
		}
	})

	t.Run("it validates result", func(t *testing.T) {
		r := requestForm(url.Values{})

		_, err := DecodeForm[Body](r, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}

		var validationErr validator.ValidationErrors

		ok := errors.As(err, &validationErr)
		if !ok {
			t.Fatal(fmt.Errorf("expected validation error got %T", err))
		}

		if validationErr[0].Field() != "StringField" {
			t.Fatal(fmt.Errorf("expected field to be StringField got %s", validationErr[0].Field()))
		}
	})
}
