package category

import (
	"bundle-server/domain/casb/policy"
	appErr "bundle-server/internal/errors"
	"context"
	"database/sql"
	_ "embed"
	"errors"

	"github.com/uptrace/bun"
)

//go:embed sql/list_category_summaries.sql
var SQLListSummaries string

//go:embed sql/list_category_cid_recursive.sql
var SQLListCidsRecursive string

type categoryRepo struct {
	db *bun.DB
}

func NewCategoryRepo(db *bun.DB) CategoryRepo {
	return &categoryRepo{
		db: db,
	}
}

func (cr *categoryRepo) ListCategorySummaries(c context.Context) ([]TCategorySummary, error) {
	var categories []TCategorySummary

	err := cr.db.NewRaw(SQLListSummaries).Scan(c, &categories)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NewDBError(appErr.DB_NO_ROWS, "", err)
		} else {
			return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
		}
	}

	return categories, nil
}

func (cr *categoryRepo) ListCategoryServices(c context.Context, pidCates []policy.Pid) ([]CategoryService, error) {
	var cidDescendants []CategoryService

	err := cr.db.NewRaw(SQLListCidsRecursive, bun.In(pidCates)).Scan(c, &cidDescendants)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NewDBError(appErr.DB_NO_ROWS, "", err)
		} else {
			return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
		}
	}

	return cidDescendants, nil
}
