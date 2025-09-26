package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var (
	ErrNoRows            = errors.New("no rows")
	ErrNoPgConnInContext = errors.New("no pg connection in context")
)

type pgConnCtx struct{}

type Loader interface {
	LoadContext(ctx context.Context, val any) error
	LoadOneContext(ctx context.Context, val any) error
	ExecContext(ctx context.Context) error
}

type Queryer interface {
	Query(query string, vals ...any) Loader
}

type Tx interface {
	Commit() error
	RollbackUnlessCommitted()
	Rollback() error
}

func NewContext(parent context.Context, conn *sqlx.DB) context.Context {
	return context.WithValue(parent, pgConnCtx{}, conn)
}

func FromContext(ctx context.Context) Queryer {
	switch val := ctx.Value(pgConnCtx{}).(type) {
	case *sqlx.DB:
		return &queryer{db: val}
	case *tx:
		return &txQueryer{tx: val.tx}
	default:
		return nullQueryer{}
	}
}

func NewConn(ctx context.Context, pg string) *sqlx.DB {
	sql.ErrNoRows = ErrNoRows
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	conn, err := sqlx.Connect("postgres", pg)
	if err != nil {
		panic(fmt.Errorf("open postgres conn: %w", err))
	}
	return conn
}

func Begin(ctx context.Context) (context.Context, Tx, error) {
	switch val := ctx.Value(pgConnCtx{}).(type) {
	case *sqlx.DB:
		sqlxTx, err := val.BeginTxx(ctx, &sql.TxOptions{})
		if err != nil {
			return ctx, nil, fmt.Errorf("begin tx: %w", err)
		}
		t := &tx{
			tx: sqlxTx,
		}
		return context.WithValue(ctx, pgConnCtx{}, t), t, nil
	case *tx:
		t, err := val.Begin()
		if err != nil {
			return ctx, nil, fmt.Errorf("begin nested tx: %w", err)
		}
		return context.WithValue(ctx, pgConnCtx{}, t), t, nil
	case nil:
		return nil, nil, ErrNoPgConnInContext
	default:
		return nil, nil, fmt.Errorf("unknown value in context: %T", val)
	}
}

type loader struct {
	db    *sqlx.DB
	query string
	args  []any
}

func (l *loader) LoadContext(ctx context.Context, val any) error {
	return l.db.SelectContext(ctx, val, l.query, l.args...)
}

func (l *loader) LoadOneContext(ctx context.Context, val any) error {
	return l.db.GetContext(ctx, val, l.query, l.args...)
}

func (l *loader) ExecContext(ctx context.Context) error {
	_, err := l.db.ExecContext(ctx, l.query, l.args...)
	return err
}

type queryer struct {
	db *sqlx.DB
}

func (s *queryer) Query(query string, vals ...any) Loader {
	return &loader{
		db:    s.db,
		query: query,
		args:  vals,
	}
}

type txQueryer struct {
	tx *sqlx.Tx
}

func (t *txQueryer) Query(query string, vals ...any) Loader {
	return &txLoader{
		tx:    t.tx,
		query: query,
		args:  vals,
	}
}

type txLoader struct {
	tx    *sqlx.Tx
	query string
	args  []any
}

func (t *txLoader) LoadContext(ctx context.Context, val any) error {
	return t.tx.SelectContext(ctx, val, t.query, t.args...)
}

func (t *txLoader) LoadOneContext(ctx context.Context, val any) error {
	return t.tx.GetContext(ctx, val, t.query, t.args...)
}

func (t *txLoader) ExecContext(ctx context.Context) error {
	_, err := t.tx.ExecContext(ctx, t.query, t.args...)
	return err
}

type tx struct {
	tx       *sqlx.Tx
	prev     *tx
	num      int
	commited bool
}

func (t *tx) Commit() error {
	if t.commited {
		return fmt.Errorf("already commited")
	}
	if t.prev == nil {
		return t.tx.Commit()
	}
	return nil
}

func (t *tx) RollbackUnlessCommitted() {
	if !t.commited {
		_ = t.Rollback()
	}
}

func (t *tx) Rollback() error {
	if t.prev == nil {
		return t.tx.Rollback()
	} else {
		_, err := t.tx.Exec("rollback to sp_" + strconv.Itoa(t.num))
		return err
	}
}

func (t *tx) Begin() (*tx, error) {
	_, err := t.tx.Exec("savepoint sp_" + strconv.Itoa(t.num))
	if err != nil {
		return nil, fmt.Errorf("add savepoint sp_%d: %w", t.num, err)
	}
	return &tx{
		tx:   t.tx,
		prev: t,
		num:  t.num + 1,
	}, nil
}

type nullQueryer struct{}

func (nullQueryer) Query(query string, vals ...any) Loader {
	return nullLoader{}
}

type nullLoader struct{}

func (nullLoader) LoadContext(ctx context.Context, val any) error {
	return ErrNoPgConnInContext
}

func (nullLoader) LoadOneContext(ctx context.Context, val any) error {
	return ErrNoPgConnInContext
}

func (nullLoader) ExecContext(ctx context.Context) error {
	return ErrNoPgConnInContext
}
