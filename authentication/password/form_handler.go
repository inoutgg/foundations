package password

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/mold/v4/scrubbers"
	"github.com/go-playground/validator/v10"
	"go.inout.gg/common/authentication"
	"go.inout.gg/common/authentication/db/driver"
	"go.inout.gg/common/authentication/strategy"
	httperror "go.inout.gg/common/http/error"
)

var (
	DefaultFormValidate = validator.New()
	DefaultFormScrubber = scrubbers.New()
	DefaultFormModifier = modifiers.New()
)

const (
	DefaultFieldNameFirstName = "first_name"
	DefaultFieldNameLastName  = "last_name"
	DefaultFieldNameEmail     = "email"
	DefaultFieldNamePassword  = "password"
)

type FormConfig[T any] struct {
	*Config[T]

	Validator    *validator.Validate
	FormScrubber *mold.Transformer
	FormModifier *mold.Transformer

	FirstNameFieldName string
	LastNameFieldName  string
	EmailFieldName     string
	PasswordFieldName  string
}

// NewFormConfig[T] creates a new FormConfig[T] with the given configuration options.
func NewFormConfig[T any](config ...func(*FormConfig[T])) *FormConfig[T] {
	cfg := &FormConfig[T]{
		FirstNameFieldName: DefaultFieldNameFirstName,
		LastNameFieldName:  DefaultFieldNameLastName,
		EmailFieldName:     DefaultFieldNameEmail,
		PasswordFieldName:  DefaultFieldNamePassword,
	}

	for _, f := range config {
		f(cfg)
	}

	// Set defaults.
	if cfg.Config == nil {
		cfg.Config = NewConfig[T]()
	}

	if cfg.Validator == nil {
		cfg.Validator = DefaultFormValidate
	}

	if cfg.FormScrubber == nil {
		cfg.FormScrubber = DefaultFormScrubber
	}

	if cfg.FormModifier == nil {
		cfg.FormModifier = DefaultFormModifier
	}

	return cfg
}

func WithConfig[T any](config *Config[T]) func(*FormConfig[T]) {
	return func(cfg *FormConfig[T]) { cfg.Config = config }
}

// FormHandler[T] is a wrapper around Handler handling HTTP form requests.
type FormHandler[T any] struct {
	config  *FormConfig[T]
	handler *Handler[T]
}

// NewFormHandler[T] creates a new FormHandler[T] with the given configuration.
func NewFormHandler[T any](driver driver.Driver, config *FormConfig[T]) *FormHandler[T] {
	return &FormHandler[T]{
		handler: &Handler[T]{
			config: config.Config,
			driver: driver,
		},
		config: config,
	}
}

// userRegistrationForm is the form for user login.
type userRegistrationForm struct {
	FirstName string `mod:"trim"`
	LastName  string `mod:"trim"`
	Email     string `mod:"trim" validate:"required,email" scrub:"emails"`
	Password  string `mod:"trim" validate:"required"`
}

// userLoginForm is the form for user login.
type userLoginForm struct {
	Email    string `mod:"trim" validate:"required,email" scrub:"emails"`
	Password string `mod:"trim" validate:"required"`
}

func (h *FormHandler[T]) parseUserRegistrationForm(
	req *http.Request,
) (*userRegistrationForm, error) {
	ctx := req.Context()

	if err := req.ParseForm(); err != nil {
		return nil, fmt.Errorf("password: failed to parse request form: %w", err)
	}

	form := &userRegistrationForm{
		FirstName: req.PostFormValue(h.config.FirstNameFieldName),
		LastName:  req.PostFormValue(h.config.LastNameFieldName),
		Email:     req.PostFormValue(h.config.EmailFieldName),
		Password:  req.PostFormValue(h.config.PasswordFieldName),
	}

	if err := h.config.FormModifier.Struct(ctx, form); err != nil {
		return nil, fmt.Errorf("password: failed to parse request form: %w", err)
	}

	if err := h.config.Validator.Struct(form); err != nil {
		return nil, fmt.Errorf("password: failed to parse request form: %w", err)
	}

	return form, nil
}

// HandleUserRegistration handles a user registration request.
func (h *FormHandler[T]) HandleUserRegistration(
	w http.ResponseWriter,
	r *http.Request,
) (*strategy.User[T], error) {
	form, err := h.parseUserRegistrationForm(r)
	if err != nil {
		return nil, httperror.FromError(err, http.StatusBadRequest)
	}

	result, err := h.handler.HandleUserRegistration(r.Context(), form.Email, form.Password)
	if err != nil {
		if errors.Is(err, authentication.ErrAuthorizedUser) {
			return nil, httperror.FromError(err, http.StatusForbidden)
		}

		return nil, httperror.FromError(err, http.StatusInternalServerError)
	}

	return result, nil
}

func (h *FormHandler[T]) parseUserLoginForm(req *http.Request) (*userLoginForm, error) {
	ctx := req.Context()

	if err := req.ParseForm(); err != nil {
		return nil, fmt.Errorf("password: failed to parse request form: %w", err)
	}

	form := &userLoginForm{
		Email:    req.PostFormValue(h.config.EmailFieldName),
		Password: req.PostFormValue(h.config.PasswordFieldName),
	}

	if err := h.config.FormModifier.Struct(ctx, form); err != nil {
		return nil, fmt.Errorf("password: failed to parse request form: %w", err)
	}

	if err := h.config.Validator.Struct(form); err != nil {
		return nil, fmt.Errorf("password: failed to parse request form: %w", err)
	}

	return form, nil
}

// HandleUserLogin handles a user login request.
func (h *FormHandler[T]) HandleUserLogin(
	w http.ResponseWriter,
	r *http.Request,
) (*strategy.User[T], error) {
	form, err := h.parseUserLoginForm(r)
	if err != nil {
		return nil, httperror.FromError(err, http.StatusBadRequest)
	}

	result, err := h.handler.HandleUserLogin(r.Context(), form.Email, form.Password)
	if err != nil {
		if errors.Is(err, authentication.ErrAuthorizedUser) {
			return nil, httperror.FromError(err, http.StatusForbidden)
		}

		return nil, httperror.FromError(err, http.StatusInternalServerError)
	}

	return result, nil
}
