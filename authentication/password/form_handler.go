package password

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/mold/v4/scrubbers"
	"github.com/go-playground/validator/v10"
	httperror "go.inout.gg/common/http/error"
)

var (
	DefaultFormValidate = validator.New()
	DefaultFormScrubber = scrubbers.New()
	DefaultFormModifier = modifiers.New()
)

var (
	FieldNameEmail    = "email"
	FieldNamePassword = "password"
)

type FormConfig struct {
	*Config

	Validator    *validator.Validate
	FormScrubber *mold.Transformer
	FormModifier *mold.Transformer

	EmailFieldName    string
	PasswordFieldName string
}

// NewFormConfig creates a new FormConfig with the given configuration options.
func NewFormConfig(config ...func(*FormConfig)) *FormConfig {
	cfg := &FormConfig{
		Config: NewConfig(),
	}

	for _, f := range config {
		f(cfg)
	}

	// Set defaults.
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

type FormHandler struct {
	config  *FormConfig
	handler *Handler
}

// userLoginForm is the form for user login.
type userLoginForm struct {
	Email    string `mod:"trim" validate:"required,email" scrub:"emails"`
	Password string `mod:"trim" validate:"required"`
}

func (h *FormHandler) parseUserLoginForm(req *http.Request) (*userLoginForm, error) {
	ctx := req.Context()

	if err := req.ParseForm(); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	form := &userLoginForm{
		Email:    req.PostFormValue(h.config.EmailFieldName),
		Password: req.PostFormValue(h.config.PasswordFieldName),
	}

	if err := h.config.FormModifier.Struct(ctx, form); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	if err := h.config.Validator.Struct(form); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	return form, nil
}

// HandleUserLoginRequest handles a user login request.
func (h *FormHandler) HandleUserLoginRequest(w http.ResponseWriter, r *http.Request) error {
	form, err := h.parseUserLoginForm(r)
	if err != nil {
		return httperror.FromError(err, http.StatusBadRequest)
	}

	_, err = h.handler.HandleUserLoginRequest(r.Context(), form.Email, form.Password)
	if err != nil {
		if errors.Is(err, ErrAuthorizedUser) {
			return httperror.FromError(err, http.StatusForbidden)
		}

		return httperror.FromError(err, http.StatusInternalServerError)
	}

	return nil
}
