package sqlcomment

import (
	"context"
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

func TestTags_WithTags(t *testing.T) {
	t.Parallel()

	t.Run("should merge tags without sharing maps when keys overlap", func(t *testing.T) {
		t.Parallel()

		base := WithTag(context.Background(), KeyApplication, "old")
		supplied := Tags{KeyApplication: "new", KeyRoute: "/orders"}
		ctx := WithTags(base, supplied)

		supplied[KeyRoute] = "/changed"
		returned := FromContext(ctx)
		returned[KeyApplication] = "changed"

		require.Equal(t, Tags{KeyApplication: "new", KeyRoute: "/orders"}, FromContext(ctx))
		require.Equal(t, Tags{KeyApplication: "old"}, FromContext(base))
	})

	t.Run("should skip comments when tags are added after skipping", func(t *testing.T) {
		t.Parallel()

		ctx := WithTags(Skip(context.Background()), Tags{KeyApplication: "api"})
		require.Equal(t, "select 1", Append(ctx, "select 1"))
	})
}
