package dbsqltest

import (
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.segfaultmedaddy.com/pgxephemeraltest"
)

type PoolFactoryConstructor func(testing.TB) *pgxpool.Pool

func NewPoolFactoryConstructor(
	m pgxephemeraltest.Migrator,
	config *pgxpool.Config,
	opts ...pgxephemeraltest.FactoryOption,
) PoolFactoryConstructor {
	var once sync.Once

	var err error

	var factory *pgxephemeraltest.PoolFactory

	return func(tb testing.TB) *pgxpool.Pool {
		tb.Helper()

		once.Do(func() {
			factory, err = pgxephemeraltest.NewPoolFactory(
				tb.Context(),
				config,
				m,
				opts...,
			)
		})

		if err != nil {
			tb.Fatal(err)
		}

		return factory.Pool(tb)
	}
}
