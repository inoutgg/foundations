package dbsql

import "go.inout.gg/foundations/dbsql/dbtx"

// DBTX is an interface common to pgx.Conn, pgx.Tx and pgxpool.Pool.
type DBTX = dbtx.DBTX

// DB is an interface common to pgx.Conn and pgxpool.Pool.
type DB = dbtx.DB
