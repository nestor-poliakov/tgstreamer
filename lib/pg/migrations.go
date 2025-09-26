package pg

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"errors"
	"tgstreamer/lib/log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func Migrate(fs fs.FS, conf Config) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if conf.DropTables {
		log.Default().Warn("drop pg tables")
		conn := NewConn(ctx, conf.Dsn)
		defer conn.Close()
		var dropSql []string
		var dropTypes []string
		err := conn.SelectContext(ctx, &dropSql, `select 'drop table if exists "' || schemaname || '"."' || tablename || '" cascade;' as string from pg_tables where not schemaname in ('pg_catalog','information_schema');`)
		if err != nil {
			panic(fmt.Errorf("get drop tables query: %w", err))
		}
		err = conn.SelectContext(ctx, &dropTypes, `select 'drop type if exists "' || typname || '";' as str from pg_catalog.pg_type where typowner in (select oid from pg_catalog.pg_authid where rolname = current_role);`)
		if err != nil {
			panic(fmt.Errorf("get drop types query: %w", err))
		}
		fmt.Println(dropSql)
		_, err = conn.ExecContext(ctx, strings.Join(append(dropSql, dropTypes...), "\n"))
		if err != nil {
			panic(fmt.Errorf("drop tables: %w", err))
		}
	}

	d, err := iofs.New(fs, "migrations")
	if err != nil {
		panic(fmt.Errorf("new iofs: %w", err))
	}
	defer d.Close()
	m, err := migrate.NewWithSourceInstance("iofs", d, conf.Dsn)
	if err != nil {
		panic(fmt.Errorf("new migrator: %w", err))
	}
	defer m.Close()
	ver, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		err = nil
	}
	if err != nil {
		panic(fmt.Errorf("get version: %w", err))
	}
	if dirty {
		err = m.Force(int(ver))
		if err != nil {
			panic(fmt.Errorf("migrate dirty version: %w", err))
		}
	}
	for {
		err = m.Up()
		if err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				break
			}
			panic(fmt.Errorf("migrate up: %w", err))
		}
	}

	newVer, _, err := m.Version()
	if err != nil {
		panic(fmt.Errorf("get new version: %w", err))
	}
	log.Defaults().Infof("migrate pg from %d to %d", ver, newVer)
}
