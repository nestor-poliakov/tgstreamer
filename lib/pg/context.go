package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gocraft/dbr/v2"
	_ "github.com/lib/pq"
)

var (
	ErrNoRows            = errors.New("no rows")
	ErrNoPgConnInContext = errors.New("no pg connection in context")
)

type pgSessionCtx struct{}

type Scanner interface {
	Scan(dest ...any) error
}

type Loader interface {
	// placeholder: ?
	LoadContext(ctx context.Context, val any) error
	LoadOneContext(ctx context.Context, val any) error
	ExecContext(ctx context.Context) error
	// placeholder: $num
	QueryRowContext(ctx context.Context) Scanner
	QueryContext(ctx context.Context) (*sql.Rows, error)
}

type loader struct {
	sess  *dbr.Session
	query string
	args  []any
}

func (l *loader) LoadContext(ctx context.Context, val any) error {
	return l.sess.InsertBySql(l.query, l.args...).LoadContext(ctx, val)
}

func (l *loader) LoadOneContext(ctx context.Context, val any) error {
	return l.sess.SelectBySql(l.query, l.args...).LoadOneContext(ctx, val)
}

func (l *loader) ExecContext(ctx context.Context) error {
	_, err := l.sess.UpdateBySql(l.query, l.args...).ExecContext(ctx)
	return err
}

func (l *loader) QueryContext(ctx context.Context) (*sql.Rows, error) {
	return l.sess.DB.QueryContext(ctx, l.query, l.args...)
}

func (l *loader) QueryRowContext(ctx context.Context) Scanner {
	return l.sess.DB.QueryRowContext(ctx, l.query, l.args...)
}

type Queryer interface {
	Query(query string, vals ...any) Loader
}

type queryer struct {
	sess *dbr.Session
}

func (s *queryer) Query(query string, vals ...any) Loader {
	s.sess.EventKv("query pg", map[string]string{"sql": query})
	return &loader{
		sess:  s.sess,
		query: query,
		args:  vals,
	}
}

type nullQueryer struct{}

func (nullQueryer) Query(query string, vals ...any) Loader {
	return nullLoader{}
}

func NewContext(parent context.Context, conn *dbr.Connection) context.Context {
	return context.WithValue(parent, pgSessionCtx{}, conn)
}

func FromContext(ctx context.Context) Queryer {
	switch val := ctx.Value(pgSessionCtx{}).(type) {
	case *dbr.Connection:
		return &queryer{sess: val.NewSession(nil)}
	case *tx:
		return &txQueryer{tx: val.tx}
	default:
		return nullQueryer{}
	}
}

func NewConn(ctx context.Context, pg string) *dbr.Connection {
	dbr.ErrNotFound = ErrNoRows
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	conn, err := dbr.Open("postgres", pg, nil)
	if err != nil {
		panic(fmt.Errorf("open postgres conn: %w", err))
	}
	err = conn.PingContext(ctx)
	if err != nil {
		panic(fmt.Errorf("ping pg connection: %w", err))
	}
	return conn
}

func Begin(ctx context.Context) (context.Context, Tx, error) {
	switch val := ctx.Value(pgSessionCtx{}).(type) {
	case *dbr.Connection:
		dbrTx, err := val.NewSession(nil).Begin()
		if err != nil {
			return ctx, nil, fmt.Errorf("begin tx: %w", err)
		}
		t := &tx{
			tx: dbrTx,
		}
		return context.WithValue(ctx, pgSessionCtx{}, t), t, nil
	case *tx:
		t, err := val.Begin()
		if err != nil {
			return ctx, nil, fmt.Errorf("begin nested tx: %w", err)
		}
		return context.WithValue(ctx, pgSessionCtx{}, t), t, nil
	default:
		return nil, nil, fmt.Errorf("unknown value in context: %T", val)
	}
}

type txQueryer struct {
	tx *dbr.Tx
}

func (t *txQueryer) Query(query string, vals ...any) Loader {
	return &txLoader{
		tx:    t.tx,
		query: query,
		args:  vals,
	}
}

type txLoader struct {
	tx    *dbr.Tx
	query string
	args  []any
}

func (t *txLoader) LoadContext(ctx context.Context, val any) error {
	return t.tx.UpdateBySql(t.query, t.args...).LoadContext(ctx, val)
}

func (t *txLoader) LoadOneContext(ctx context.Context, val any) error {
	return t.tx.SelectBySql(t.query, t.args...).LoadOneContext(ctx, val)
}

func (t *txLoader) ExecContext(ctx context.Context) error {
	_, err := t.tx.UpdateBySql(t.query, t.args...).ExecContext(ctx)
	return err
}

func (t *txLoader) QueryRowContext(ctx context.Context) Scanner {
	return t.tx.QueryRowContext(ctx, t.query, t.args...)
}

func (t *txLoader) QueryContext(ctx context.Context) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, t.query, t.args)
}

type Tx interface {
	Commit() error
	RollbackUnlessCommitted()
	Rollback() error
}

type tx struct {
	tx       *dbr.Tx
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

type beginner interface {
	Begin() (*dbr.Tx, error)
}

type nullScanner struct{}

func (nullScanner) Scan(dest ...any) error {
	return ErrNoPgConnInContext
}

type nullLoader struct{}

func (l nullLoader) Begin() (*dbr.Tx, error) {
	return nil, ErrNoPgConnInContext
}

func (nullLoader) QueryRowContext(ctx context.Context) Scanner {
	return nullScanner{}
}

func (nullLoader) QueryContext(ctx context.Context) (*sql.Rows, error) {
	return nil, ErrNoPgConnInContext
}

func (nullLoader) LoadContext(ctx context.Context, val any) error {
	return ErrNoPgConnInContext
}

func (nullLoader) LoadOneContext(ctx context.Context, val any) error {
	return ErrNoPgConnInContext
}

func (nullLoader) ExecContext(ctx context.Context) error {
	return ErrNoPgConnInContext
}
