package password

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/mold/v4/scrubbers"
	"github.com/go-playground/validator/v10"
	"go.inout.gg/common/authentication/db/driver"
	httperror "go.inout.gg/common/http/error"
)

var (
	DefaultFormValidate = validator.New()
	DefaultFormScrubber = scrubbers.New()
	DefaultFormModifier = modifiers.New()
)

var (
	FieldNameFirstName = "first_name"
	FieldNameLastName  = "last_name"
	FieldNameEmail     = "email"
	FieldNamePassword  = "password"
)

type FormConfig struct {
	*Config

	Validator    *validator.Validate
	FormScrubber *mold.Transformer
	FormModifier *mold.Transformer

	FirstNameFieldName string
	LastNameFieldName  string
	EmailFieldName     string
	PasswordFieldName  string
}

// NewFormConfig creates a new FormConfig with the given configuration options.
func NewFormConfig(config ...func(*FormConfig)) *FormConfig {
	cfg := &FormConfig{
		EmailFieldName:    FieldNameEmail,
		PasswordFieldName: FieldNamePassword,
	}

	for _, f := range config {
		f(cfg)
	}

	// Set defaults.
	if cfg.Config == nil {
		cfg.Config = NewConfig()
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

func WithConfig(config *Config) func(*FormConfig) {
	return func(cfg *FormConfig) { cfg.Config = config }
}

// FormHandler is a wrapper around Handler handling HTTP form requests.
type FormHandler struct {
	config  *FormConfig
	handler *Handler
}

// NewFormHandler creates a new FormHandler with the given configuration.
func NewFormHandler(driver driver.Driver, config *FormConfig) *FormHandler {
	return &FormHandler{
		handler: &Handler{
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

func (h *FormHandler) parseUserRegistrationForm(req *http.Request) (*userRegistrationForm, error) {
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
func (h *FormHandler) HandleUserRegistration(w http.ResponseWriter, r *http.Request) error {
	form, err := h.parseUserRegistrationForm(r)
	if err != nil {
		return httperror.FromError(err, http.StatusBadRequest)
	}

	_, err = h.handler.HandleUserRegistration(r.Context(), form.Email, form.Password)
	if err != nil {
		if errors.Is(err, ErrAuthorizedUser) {
			return httperror.FromError(err, http.StatusForbidden)
		}

		return httperror.FromError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FormHandler) parseUserLoginForm(req *http.Request) (*userLoginForm, error) {
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
func (h *FormHandler) HandleUserLogin(w http.ResponseWriter, r *http.Request) error {
	form, err := h.parseUserLoginForm(r)
	if err != nil {
		return httperror.FromError(err, http.StatusBadRequest)
	}

	_, err = h.handler.HandleUserLogin(r.Context(), form.Email, form.Password)
	if err != nil {
		if errors.Is(err, ErrAuthorizedUser) {
			return httperror.FromError(err, http.StatusForbidden)
		}

		return httperror.FromError(err, http.StatusInternalServerError)
	}

	return nil
}
