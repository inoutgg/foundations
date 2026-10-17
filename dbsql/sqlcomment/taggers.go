package sqlcomment

import (
	"context"
	"runtime/debug"
)

// Tagger supplies tags for SQL comments.
type Tagger interface {
	Tag(context.Context) Tags
}

type driverVersionTagger struct {
	version string
}

// NewDriverVersionTagger returns a Tagger which adds the current pgx version.
func NewDriverVersionTagger() Tagger {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return driverVersionTagger{version: "pgx"}
	}

	for _, d := range info.Deps {
		if d.Path == "github.com/jackc/pgx/v5" {
			return driverVersionTagger{version: "pgx:" + d.Version}
		}
	}

	return driverVersionTagger{version: "pgx"}
}

func (dv driverVersionTagger) Tag(_ context.Context) Tags {
	return Tags{KeyDBDriver: dv.version}
}

type contextTagger struct{}

func (ct contextTagger) Tag(ctx context.Context) Tags {
	return FromContext(ctx)
}
