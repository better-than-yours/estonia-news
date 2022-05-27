package migrations

import "github.com/uptrace/bun/migrate"

// Migrations is migration object
var Migrations = migrate.NewMigrations()

func init() { //nolint:gocritic,gochecknoinits
	if err := Migrations.DiscoverCaller(); err != nil {
		panic(err)
	}
}
