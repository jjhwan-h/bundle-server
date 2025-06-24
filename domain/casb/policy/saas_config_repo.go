package policy

import (
	"context"
	"database/sql"
	"errors"

	appErr "github.com/jjhwan-h/bundle-server/internal/errors"

	"github.com/uptrace/bun"
)

type policySaasConfigRepo struct {
	db *bun.DB
}

func NewPolicySaasConfigRepo(db *bun.DB) PolicySaasConfigRepo {
	return &policySaasConfigRepo{
		db: db,
	}
}

func (pr *policySaasConfigRepo) GetConfig(c context.Context) (*PolicySaasConfig, error) {
	config := &PolicySaasConfig{}

	err := pr.db.NewSelect().
		Model(config).
		Scan(c)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NewDBError(appErr.DB_NO_ROWS, "", err)
		} else {
			return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
		}
	}

	return config, err
}
