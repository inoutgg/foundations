package verification

import (
	"strings"

	"github.com/samber/lo"
)

type PasswordRequiredChars []string

func (s *PasswordRequiredChars) Parse(source string) error {
	parts := lo.Filter(strings.Split(source, "::"), func(s string, _ int) bool {
		return len(s) > 0
	})

	*s = PasswordRequiredChars(parts)

	return nil
}

var (
	DefaultPasswordRequiredChars string = ""
)
