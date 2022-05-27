package db

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"

	"estonia-news/config"
	"estonia-news/migrations"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
)

// Connect return db connection
func Connect(host, user, password, name string) *bun.DB {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", user, password, host, name)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	return bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())
}

// Migrate is application of a migration to a database
func Migrate(ctx context.Context) {
	dbConnect := ctx.Value(config.CtxDBKey).(*bun.DB)
	migrator := migrate.NewMigrator(dbConnect, migrations.Migrations)
	err := migrator.Init(ctx)
	if err != nil {
		panic(err)
	}
	group, err := migrator.Migrate(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("migrated to %s\n", group)
}
