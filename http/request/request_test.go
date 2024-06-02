package request

import (
	"bytes"
	"encoding/json"
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
	BoolField bool `json:"bool_field" form:"bool_field"`
}

type Body struct {
	StringField string      `json:"string_field" form:"string_field" validate:"required"`
	IntField    int         `json:"int_field"    form:"int_field"`
	NestedField NestedField `json:"nested_field" form:"nested_field"`
}

func encodeJSON(t *testing.T, val any) []byte {
	bytes, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}

	return bytes
}

func requestJSON(t *testing.T, val any) *http.Request {
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

		body, err := DecodeJSON[Body](r)
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
		)

		if err == nil || body != nil {
			t.Fatal(fmt.Errorf("expected body to be empty got %+v", body))
		}
	})

	t.Run("it validates result", func(t *testing.T) {
		r := requestJSON(t, Body{})
		_, err := DecodeJSON[Body](r)
		if err == nil {
			t.Fatal("expected validation error")
		}

		validationErr, ok := err.(validator.ValidationErrors)
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
		body, err := DecodeForm[Body](r)
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
		)

		if err == nil || body != nil {
			t.Fatal(fmt.Errorf("expected body to be empty got %+v", body))
		}
	})

	t.Run("it validates result", func(t *testing.T) {
		r := requestForm(url.Values{})
		_, err := DecodeForm[Body](r)
		if err == nil {
			t.Fatal("expected validation error")
		}

		validationErr, ok := err.(validator.ValidationErrors)
		if !ok {
			t.Fatal(fmt.Errorf("expected validation error got %T", err))
		}

		if validationErr[0].Field() != "StringField" {
			t.Fatal(fmt.Errorf("expected field to be StringField got %s", validationErr[0].Field()))
		}
	})
}
