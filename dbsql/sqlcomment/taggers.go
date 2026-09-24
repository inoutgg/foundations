package sqlcomment

import (
	"context"
	"fmt"
	"runtime/debug"
)

type driverVersionTagger struct {
	version string
}

// NewDriverVersionTagger returns a Tagger which adds the current pgx version.
func NewDriverVersionTagger() driverVersionTagger {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return driverVersionTagger{version: "pgx"}
	}

	for _, d := range info.Deps {
		if d.Path == "github.com/jackc/pgx/v5" {
			return driverVersionTagger{version: fmt.Sprintf("pgx:%s", d.Version)}
		}
	}

	return driverVersionTagger{version: "pgx"}
}

func (dv driverVersionTagger) Tag(ctx context.Context) Tags {
	return Tags{KeyDBDriver: dv.version}
}

type contextTagger struct{}

func (ct contextTagger) Tag(ctx context.Context) Tags {
	return FromContext(ctx)
}

type staticTagger struct {
	tags Tags
}

// NewStaticTagger returns a Tagger which adds tags to every SQL comment.
func NewStaticTagger(tags Tags) staticTagger {
	return staticTagger{tags: tags}
}

func (st staticTagger) Tag(ctx context.Context) Tags {
	return st.tags
}
