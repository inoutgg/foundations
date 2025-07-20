package pgxuuid

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/dbsql/dbsqltest"
)

var db *dbsqltest.DB

func uuidCodec(c *pgxpool.Config) {
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
	var err error
	var close func(context.Context) error

	ctx := context.Background()
	db, close, err = dbsqltest.NewDBWithContainer(ctx, uuidCodec)
	if err != nil {
		panic(err)
	}

	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	code := m.Run()

	// Teardown
	closeErr := close(closeCtx)
	if closeErr != nil {
		panic(closeErr)
	}

	os.Exit(code)
}
