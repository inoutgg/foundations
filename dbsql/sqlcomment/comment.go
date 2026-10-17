package sqlcomment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type commenter struct {
	taggers []Tagger
}

var defaultCommenter = commenter{taggers: []Tagger{ //nolint:gochecknoglobals
	NewDriverVersionTagger(), NewOTELTagger(), contextTagger{},
}}

// Append adds SQL comment tags from the context and current OTel propagator.
func Append(ctx context.Context, query string) string {
	return defaultCommenter.withComment(ctx, query)
}

func (c commenter) withComment(ctx context.Context, query string) string {
	if skipFromContext(ctx) {
		return query
	}

	tags := make(Tags)
	for _, tagger := range c.taggers {
		tags.Merge(tagger.Tag(ctx))
	}

	return fmt.Sprintf("%s /*%s*/", query, tags.Marshal())
}

// CommentBatch returns a batch with comments without modifying the queued queries.
func CommentBatch(ctx context.Context, b *pgx.Batch) *pgx.Batch {
	return defaultCommenter.commentBatch(ctx, b)
}

func (c commenter) commentBatch(ctx context.Context, b *pgx.Batch) *pgx.Batch {
	if skipFromContext(ctx) {
		return b
	}

	commented := &pgx.Batch{QueuedQueries: make([]*pgx.QueuedQuery, 0, len(b.QueuedQueries))}
	for _, query := range b.QueuedQueries {
		queued := *query
		queued.SQL = c.withComment(ctx, queued.SQL)
		commented.QueuedQueries = append(commented.QueuedQueries, &queued)
	}

	return commented
}
