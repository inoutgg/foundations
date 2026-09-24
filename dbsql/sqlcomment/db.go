package sqlcomment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"go.inout.gg/foundations/dbsql/dbtx"
)

var (
	_ dbtx.DB = (*DB)(nil)
	_ pgx.Tx  = (*Tx)(nil)
)

// DB adds SQL comments to calls through a pgx DB without requiring otelc.
type DB struct {
	dbtx.DB
	commenter
}

// NewWithDefaults adds the provided taggers, followed by OTel and context tags.
func NewWithDefaults(db dbtx.DB, options ...Option) *DB {
	var comment commenter
	for _, option := range options {
		option(&comment)
	}

	comment.taggers = append(comment.taggers, NewOTELTagger(), contextTagger{})

	return &DB{DB: db, commenter: comment}
}

// Pass a derived context to pgx so an otelc hook does not comment the same
// query a second time. Skip does not modify the caller's context.
func withoutHook(ctx context.Context) context.Context { return Skip(ctx) }

func (d *DB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tag, err := d.DB.Exec(withoutHook(ctx), d.withComment(ctx, query), args...)
	if err != nil {
		return tag, fmt.Errorf("sqlcomment: failed to execute query: %w", err)
	}

	return tag, nil
}

func (d *DB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := d.DB.Query(withoutHook(ctx), d.withComment(ctx, query), args...)
	if err != nil {
		return rows, fmt.Errorf("sqlcomment: failed to query: %w", err)
	}

	return rows, nil
}

func (d *DB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.DB.QueryRow(withoutHook(ctx), d.withComment(ctx, query), args...)
}

func (d *DB) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return d.DB.SendBatch(withoutHook(ctx), d.commentBatch(ctx, batch))
}

func (d *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("sqlcomment: failed to begin transaction: %w", err)
	}

	return &Tx{Tx: tx, commenter: d.commenter}, nil
}

func (d *DB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("sqlcomment: failed to begin transaction: %w", err)
	}

	return &Tx{Tx: tx, commenter: d.commenter}, nil
}

// Tx adds SQL comments to queries executed inside a transaction.
type Tx struct {
	pgx.Tx
	commenter
}

func (d *Tx) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.Tx.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("sqlcomment: failed to begin nested transaction: %w", err)
	}

	return &Tx{Tx: tx, commenter: d.commenter}, nil
}

func (d *Tx) Prepare(ctx context.Context, name, query string) (*pgconn.StatementDescription, error) {
	statement, err := d.Tx.Prepare(withoutHook(ctx), name, d.withComment(ctx, query))
	if err != nil {
		return statement, fmt.Errorf("sqlcomment: failed to prepare statement: %w", err)
	}

	return statement, nil
}

func (d *Tx) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tag, err := d.Tx.Exec(withoutHook(ctx), d.withComment(ctx, query), args...)
	if err != nil {
		return tag, fmt.Errorf("sqlcomment: failed to execute transaction query: %w", err)
	}

	return tag, nil
}

func (d *Tx) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := d.Tx.Query(withoutHook(ctx), d.withComment(ctx, query), args...)
	if err != nil {
		return rows, fmt.Errorf("sqlcomment: failed to query transaction: %w", err)
	}

	return rows, nil
}

func (d *Tx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.Tx.QueryRow(withoutHook(ctx), d.withComment(ctx, query), args...)
}

func (d *Tx) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	return d.Tx.SendBatch(withoutHook(ctx), d.commentBatch(ctx, batch))
}
