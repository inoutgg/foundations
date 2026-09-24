package sqlcomment

import "context"

// Tagger supplies tags for SQL comments.
type Tagger interface {
	Tag(context.Context) Tags
}

// Option configures the manual SQL-comment wrapper.
type Option func(*commenter)

// WithTagger adds taggers used to populate the SQL comment.
func WithTagger(taggers ...Tagger) Option {
	return func(c *commenter) {
		c.taggers = append(c.taggers, taggers...)
	}
}

// WithTags adds the given tags to every SQL query through the manual wrapper.
func WithTags(tags Tags) Option {
	return WithTagger(NewStaticTagger(tags))
}

// WithDriverVerTag adds a db_driver tag with the current pgx version.
func WithDriverVerTag() Option {
	return WithTagger(NewDriverVersionTagger())
}
