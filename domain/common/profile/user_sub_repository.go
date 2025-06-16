package profile

import (
	"context"

	appErr "bundle-server/internal/errors"

	"github.com/uptrace/bun"
)

type profileUserSubRepo struct {
	db *bun.DB
}

func NewProfileUserSubRepo(db *bun.DB) ProfileUserSubRepo {
	return &profileUserSubRepo{
		db: db,
	}
}

func (ur *profileUserSubRepo) ListGcodes(c context.Context, pid uint, gtype uint8) ([]string, error) {
	var gcodes []string

	err := ur.db.NewSelect().
		Model((*TProfileUserSub)(nil)).
		Column("gcode").
		Where("pid= ? AND gtype = ?", pid, gtype).
		Scan(c, &gcodes)

	if err != nil {
		return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
	}

	return gcodes, err
}
