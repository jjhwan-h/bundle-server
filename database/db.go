package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jjhwan-h/bundle-server/config"
	appErr "github.com/jjhwan-h/bundle-server/internal/errors"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/extra/bundebug"
	"golang.org/x/sync/errgroup"
)

var (
	dbMap = map[string]*bun.DB{} // 초기화 후 불변
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
	// 불변하므로 mutex 사용x
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
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	timeout := config.Cfg.DB.Timeout
	readTimeout := config.Cfg.DB.ReadTimeout
	writeTimeout := config.Cfg.DB.WriteTimeout
	parseTime := config.Cfg.DB.ParseTime

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
	sqldb, openErr := sql.Open("mysql", dsn)
	if openErr != nil {
		return nil, err
	}

	db = bun.NewDB(sqldb, mysqldialect.New())

	db.SetMaxOpenConns(config.Cfg.DB.MaxOpenConns)                                    // 최대 연결 수
	db.SetMaxIdleConns(config.Cfg.DB.MaxIdleConns)                                    // Idle 커넥션 수
	db.SetConnMaxLifetime(time.Duration(config.Cfg.DB.ConnMaxLifetime) * time.Minute) // 연결 생명주기
	db.SetConnMaxIdleTime(time.Duration(config.Cfg.DB.ConnMaxIdleTime) * time.Minute) // Idle 커넥션 유지 시간

	if pingErr := CheckConn(db); pingErr != nil {
		return nil, pingErr
	}

	if os.Getenv("DB_DEBUG") == "true" {
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
