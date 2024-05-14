package passwordreset

import (
	"errors"
	"log/slog"
	"net/http"

	"go.inout.gg/common/authentication/db/driverpgxv5"
	"go.inout.gg/common/authentication/password"
	"go.inout.gg/common/authentication/sender"
	httperror "go.inout.gg/common/http/error"
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
	*Handler
}

// NewFormConfig creates a new FormConfig with the given configuration options.
func NewFormConfig(
	config ...func(*FormConfig),
) *FormConfig {
	cfg := &FormConfig{
		EmailFieldName:      FieldNameEmail,
		ResetTokenFieldName: FieldNameResetToken,
		PasswordFieldName:   FieldNamePassword,
		Config: &Config{
			TokenExpiryIn: TokenExpiry,
			TokenLength:   TokenLength,
		},
	}
	for _, f := range config {
		f(cfg)
	}

	return cfg
}

// RequestForm is the form used to request a password reset.
type RequestForm struct {
	Email string
}

// ConfirmForm is the form used to confirm a password reset.
type ConfirmForm struct {
	Password   string
	ResetToken string
}

// NewFormHandler creates a new FormHandler with the given configuration.
func NewFormHandler(
	logger *slog.Logger,
	driver *driverpgxv5.Driver,
	hasher password.PasswordHasher,
	sender sender.Sender[any],
	config *FormConfig,
) *FormHandler {
	handler := &Handler{
		config: config.Config,
		logger: logger,
		driver: driver,

		PasswordHasher: hasher,
		Sender:         sender,
	}

	return &FormHandler{handler}
}

func (h *FormHandler) parsePasswordResetRequestForm(
	req *http.Request,
) (RequestForm, error) {
	return RequestForm{}, nil
}

// HandlePasswordResetRequest handles a password reset request.
func (h *FormHandler) HandlePasswordResetRequest(req *http.Request) error {
	ctx := req.Context()
	form, err := h.parsePasswordResetRequestForm(req)
	if err != nil {
		return httperror.FromError(err, http.StatusBadRequest)
	}

	if err := h.Handler.HandlePasswordResetRequest(ctx, form.Email); err != nil {
		if errors.Is(err, ErrAuthorizedUser) {
			return httperror.FromError(err, http.StatusForbidden)
		}
		return httperror.FromError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FormHandler) parsePasswordResetConfirmForm(
	req *http.Request,
) (ConfirmForm, error) {
	return ConfirmForm{}, nil
}

func (h *FormHandler) HandlePasswordResetConfirm(req *http.Request) error {
	ctx := req.Context()
	form, err := h.parsePasswordResetConfirmForm(req)
	if err != nil {
		return httperror.FromError(err, http.StatusBadRequest)
	}

	if err := h.Handler.HandlePasswordResetConfirm(ctx, form.Password, form.ResetToken); err != nil {
		if errors.Is(err, ErrAuthorizedUser) {
			return httperror.FromError(err, http.StatusForbidden)
		} else if errors.Is(err, ErrUsedPasswordResetToken) {
			return httperror.FromError(err, http.StatusBadRequest)
		}

		return httperror.FromError(err, http.StatusInternalServerError)
	}

	return nil
}
