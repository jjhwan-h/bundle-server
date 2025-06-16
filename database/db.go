package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	appErr "bundle-server/internal/errors"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/extra/bundebug"
	"golang.org/x/sync/errgroup"
)

const (
	timeout      = 3
	readTimeout  = 5
	writeTimeout = 5
	parseTime    = true
)

var (
	dbMap = map[string]*bun.DB{}
	once  sync.Once
	err   error
)

func Init(dbs []string) error {
	var initErr error

	once.Do(func() {
		tmpMap := make(map[string]*bun.DB)
		for _, db := range dbs {
			conn, err := dbConn(db)
			if err != nil {
				for _, c := range tmpMap {
					_ = c.Close()
				}
				initErr = appErr.NewDBError(appErr.DB_CONN_FAIL, "", err)
				return
			}

			tmpMap[db] = conn
		}
		dbMap = tmpMap
	})

	return initErr
}

func GetDB(dbName string) *bun.DB {
	return dbMap[dbName]
}

func CloseAll() error {
	g := &errgroup.Group{}

	errCh := make(chan error, len(dbMap))

	for _, conn := range dbMap {
		conn := conn

		g.Go(func() error {
			err = conn.Close()
			if err != nil {
				errCh <- appErr.NewDBError(appErr.DB_CLOSE_FAIL, "", err)
			}

			return nil
		})
	}

	_ = g.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func dbConn(dbName string) (*bun.DB, error) {
	var db *bun.DB
	user := viper.GetString("DB_USER")
	password := viper.GetString("DB_PASSWORD")
	host := viper.GetString("DB_HOST")
	port := viper.GetString("DB_PORT")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=%t&timeout=%ds&readTimeout=%ds&writeTimeout=%ds",
		user,
		password,
		host,
		port,
		dbName,
		parseTime,
		timeout,
		readTimeout,
		writeTimeout,
	)
	log.Println(dsn)
	sqldb, openErr := sql.Open("mysql", dsn)
	if openErr != nil {
		return nil, err
	}

	db = bun.NewDB(sqldb, mysqldialect.New())

	if pingErr := CheckConn(db); pingErr != nil {
		return nil, pingErr
	}

	if viper.GetBool("DB_DEBUG") {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	return db, nil
}

func CheckConn(db *bun.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

	err = db.PingContext(ctx)
	defer cancel()

	if err != nil {
		return appErr.NewDBError(appErr.DB_CONN_FAIL, "", err)
	}
	return nil
}
