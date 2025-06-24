package org

import (
	"context"
	"database/sql"
	"errors"

	appErr "github.com/jjhwan-h/bundle-server/internal/errors"

	_ "embed"

	"github.com/uptrace/bun"
)

//go:embed sql/list_gid_recursive.sql
var SQLListGidsRecursive string

type orgGroupRepo struct {
	db *bun.DB
}

func NewOrgGroupRepo(db *bun.DB) OrgGroupRepo {
	return &orgGroupRepo{
		db: db,
	}
}

func (gr *orgGroupRepo) ListGidsRecursive(c context.Context, rootGcodes []string) ([]string, error) {
	var pids []string

	err := gr.db.NewRaw(SQLListGidsRecursive, bun.In(rootGcodes)).Scan(c, &pids)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NewDBError(appErr.DB_NO_ROWS, "", err)
		} else {
			return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
		}
	}

	return pids, nil
}
