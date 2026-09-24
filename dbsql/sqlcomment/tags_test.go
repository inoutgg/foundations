package sqlcomment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTags(t *testing.T) {
	t.Parallel()

	t.Run("marshal", func(t *testing.T) {
		t.Parallel()

		// Arrange
		tags := Tags{
			"route":      "/users/:id",
			"request id": "a b",
		}
		want := "request+id='a%20b',route='%2Fusers%2F:id'"

		// Act
		got := tags.Marshal()

		// Assert
		require.Equal(t, want, got)
	})

	t.Run("merge", func(t *testing.T) {
		t.Parallel()

		// Arrange
		tags := Tags{"application": "api", "route": "static"}

		// Act
		tags.Merge(Tags{"route": "context"})

		// Assert
		require.Equal(t, "api", tags["application"])
		require.Equal(t, "context", tags["route"])
	})
}
