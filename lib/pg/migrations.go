package pg

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"tgstreamer/lib/log"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func Migrate(migrationsFS fs.FS, conf Config) {
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

	db, err := sql.Open("postgres", conf.Dsn)
	if err != nil {
		panic(fmt.Errorf("open postgres conn: %w", err))
	}
	defer db.Close()

	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		panic(fmt.Errorf("set goose dialect: %w", err))
	}

	ver, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		panic(fmt.Errorf("get version: %w", err))
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		panic(fmt.Errorf("migrate up: %w", err))
	}
	newVer, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		panic(fmt.Errorf("get new version: %w", err))
	}
	log.Defaults().Infof("migrate pg from %d to %d", ver, newVer)
}
