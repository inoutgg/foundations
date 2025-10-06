package pgxtypeid

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/must"
)

var pool *pgxpool.Pool //nolint:gochecknoglobals

func typeidCodec(c *pgxpool.Config) {
	origAfterConnect := c.AfterConnect
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		Register(conn.TypeMap())
		if origAfterConnect != nil {
			return origAfterConnect(ctx, conn)
		}
		return nil
	}
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	pool = must.Must(pgxpool.New(ctx, "postgresql://test:test@localhost:5432/postgres"))
	defer pool.Close()

	must.Must1(pool.Ping(ctx))

	code := m.Run()
	os.Exit(code)
}
