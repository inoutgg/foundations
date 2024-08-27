package passwordreset

import (
	"errors"
	"fmt"
	"net/http"

	"go.inout.gg/foundations/authentication"
	"go.inout.gg/foundations/authentication/db/driver"
	"go.inout.gg/foundations/authentication/password"
	"go.inout.gg/foundations/authentication/sender"
	httperror "go.inout.gg/foundations/http/error"
)

const (
	FieldNameEmail      = "email"
	FieldNameResetToken = "reset_token"
	FieldNamePassword   = "password"
)

// FormConfig is the configuration for form-based password reset.
type FormConfig struct {
	*Config

	EmailFieldName      string
	ResetTokenFieldName string
	PasswordFieldName   string
}

// FormHandler is a wrapper around Handler handling HTTP form requests.
type FormHandler struct {
	config  *FormConfig
	handler *Handler
}

// NewFormConfig creates a new FormConfig with the given configuration options.
func NewFormConfig(
	config ...func(*FormConfig),
) *FormConfig {
	cfg := &FormConfig{
		EmailFieldName:      FieldNameEmail,
		ResetTokenFieldName: FieldNameResetToken,
		PasswordFieldName:   FieldNamePassword,
	}

	for _, f := range config {
		f(cfg)
	}

	// Set defaults.
	if cfg.Config == nil {
		cfg.Config = NewConfig()
	}

	return cfg
}

// WithConfig sets the configuration for the underlying Handler for FormHandler.
func WithConfig(config *Config) func(*FormConfig) {
	return func(cfg *FormConfig) { cfg.Config = config }
}

// requestForm is the form used to request a password reset.
type requestForm struct {
	Email string `mod:"trim" validate:"required,email" scrub:"emails"`
}

// confirmForm is the form used to confirm a password reset.
type confirmForm struct {
	Password   string `mod:"trim" validate:"required"`
	ResetToken string `mod:"trim" validate:"required"`
}

// NewFormHandler creates a new FormHandler with the given configuration.
func NewFormHandler(
	driver driver.Driver,
	sender sender.Sender,
	config *FormConfig,
) *FormHandler {
	handler := &Handler{
		config.Config,
		driver,
		sender,
	}

	return &FormHandler{config, handler}
}

func (h *FormHandler) parsePasswordResetRequestForm(
	r *http.Request,
) (*requestForm, error) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	form := &requestForm{
		Email: r.PostFormValue(h.config.EmailFieldName),
	}

	if err := password.FormModifier.Struct(ctx, form); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	if err := password.FormValidator.Struct(form); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	return form, nil
}

// HandlePasswordReset handles a password reset request.
func (h *FormHandler) HandlePasswordReset(req *http.Request) error {
	ctx := req.Context()
	form, err := h.parsePasswordResetRequestForm(req)
	if err != nil {
		return httperror.FromError(err, http.StatusBadRequest)
	}

	if err := h.handler.HandlePasswordReset(ctx, form.Email); err != nil {
		if errors.Is(err, authentication.ErrAuthorizedUser) {
			return httperror.FromError(err, http.StatusForbidden)
		}
		return httperror.FromError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FormHandler) parsePasswordResetConfirmForm(
	req *http.Request,
) (*confirmForm, error) {
	ctx := req.Context()

	if err := req.ParseForm(); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	form := &confirmForm{
		Password:   req.PostFormValue(h.config.PasswordFieldName),
		ResetToken: req.FormValue(h.config.ResetTokenFieldName),
	}

	if err := password.FormModifier.Struct(ctx, form); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	if err := password.FormValidator.Struct(form); err != nil {
		return nil, fmt.Errorf("password/reset: failed to parse request form: %w", err)
	}

	return form, nil
}

// HandlePasswordResetConfirm handles a password reset confirmation.
func (h *FormHandler) HandlePasswordResetConfirm(req *http.Request) error {
	ctx := req.Context()
	form, err := h.parsePasswordResetConfirmForm(req)
	if err != nil {
		return httperror.FromError(err, http.StatusBadRequest)
	}

	if err := h.handler.HandlePasswordResetConfirm(ctx, form.Password, form.ResetToken); err != nil {
		// Don't allow to change password for logged in users.
		if errors.Is(err, authentication.ErrAuthorizedUser) {
			return httperror.FromError(err, http.StatusForbidden)
		} else if errors.Is(err, ErrUsedPasswordResetToken) {
			return httperror.FromError(err, http.StatusBadRequest)
		}

		return httperror.FromError(err, http.StatusInternalServerError)
	}

	return nil
}
