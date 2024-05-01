package verification

import (
	"strings"
)

var _ error = (*PasswordVerificationError)(nil)

type Reason string

const (
	ReasonPasswordToShort      Reason = "Password is too short"
	ReasonMissingRequiredChars Reason = "Password is missing required characters"
)

type PasswordVerificationError struct {
	message string
	Reasons []Reason
}

func (e *PasswordVerificationError) Error() string {
	return e.message
}

type Option struct {
	// MinLength is the minimum length of the password.
	MinLength int

	// RequiredChars is the list of required characters.
	RequiredChars PasswordRequiredChars
}

// PasswordVerifier verifies strongness of the password.
type PasswordVerifier struct {
	opt *Option
}

// New creates a new PasswordVerifier.
func New(options ...func(*Option)) (*PasswordVerifier, error) {
	var defaultRequiredChars PasswordRequiredChars
	if err := defaultRequiredChars.Parse(DefaultPasswordRequiredChars); err != nil {
		return nil, err
	}

	opt := &Option{
		MinLength:     8,
		RequiredChars: defaultRequiredChars,
	}
	for _, f := range options {
		f(opt)
	}

	return &PasswordVerifier{
		opt,
	}, nil
}

// Verify verifies the strongness password.
func (v *PasswordVerifier) Verify(password string) error {
	var reasons []Reason
	var messages []string

	if len(password) < v.opt.MinLength {
		reasons = append(reasons, ReasonPasswordToShort)
	}

	for _, requiredCharsPart := range v.opt.RequiredChars {
		if !strings.ContainsAny(password, requiredCharsPart) {
			reasons = append(reasons, ReasonMissingRequiredChars)
		}

		if len(reasons) > 0 {
			return &PasswordVerificationError{
				message: strings.Join(messages, ", "),
				Reasons: reasons,
			}
		}
	}

	return nil
}
