package org

import (
	"context"

	appErr "bundle-server/internal/errors"

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
		return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
	}

	return pids, nil
}
