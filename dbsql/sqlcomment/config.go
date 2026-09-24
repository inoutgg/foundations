package sqlcomment

import "context"

// Tagger is used by the DB wrapper to add tags to SQL queries.
type Tagger interface {
	Tag(context.Context) Tags
}

type (
	// Option configures SQL commenting.
	Option func(*commenter)
)

// WithTagger sets the taggers used to populate the SQL comment.
func WithTagger(taggers ...Tagger) Option {
	return func(c *commenter) {
		c.taggers = append(c.taggers, taggers...)
	}
}

// WithTags appends the given tags to every SQL query.
func WithTags(tags Tags) Option {
	return func(c *commenter) {
		c.taggers = append(c.taggers, NewStaticTagger(tags))
	}
}

// WithDriverVerTag adds a db_driver tag with the current pgx version.
func WithDriverVerTag() Option {
	return func(c *commenter) {
		c.taggers = append(c.taggers, NewDriverVersionTagger())
	}
}
