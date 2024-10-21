package templcomponent

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// Empty component renders nothing.
func Empty() templ.ComponentFunc {
	return func(ctx context.Context, w io.Writer) error {
		return nil
	}
}
