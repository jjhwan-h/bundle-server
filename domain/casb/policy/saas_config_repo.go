package policy

import (
	appErr "bundle-server/internal/errors"
	"context"

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
		return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
	}

	return config, err
}
