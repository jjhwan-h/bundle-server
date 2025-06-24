package policy

import (
	"context"
	"database/sql"
	"errors"

	appErr "github.com/jjhwan-h/bundle-server/internal/errors"

	_ "embed"

	"github.com/uptrace/bun"
)

//go:embed sql/list_group_attrs.sql
var SQLListGroupAttrs string

//go:embed sql/list_cate_pid.sql
var SQLListCatePids string

type policySaasRepo struct {
	db *bun.DB
}

func NewPolicySaasRepo(db *bun.DB) PolicySaasRepo {
	return &policySaasRepo{
		db: db,
	}
}

func (sr *policySaasRepo) ListPolicies(c context.Context) ([]TPolicySaas, error) {
	var Policies []TPolicySaas

	err := sr.db.NewSelect().
		Model(&Policies).
		Column("rule_id", "rule_name", "seq", "action", "enable").
		Scan(c)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NewDBError(appErr.DB_NO_ROWS, "", err)
		} else {
			return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
		}
	}
	return Policies, err
}

func (sr *policySaasRepo) ListGroupAttrs(c context.Context, ruleID uint) ([]GroupAttr, error) {
	var groupAttrs []GroupAttr

	err := sr.db.NewRaw(SQLListGroupAttrs, ruleID).Scan(c, &groupAttrs)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NewDBError(appErr.DB_NO_ROWS, "", err)
		} else {
			return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
		}
	}
	return groupAttrs, nil
}

func (sr *policySaasRepo) ListCatePids(c context.Context, ruleID uint) ([]Pid, error) {
	var pids []Pid

	err := sr.db.NewRaw(SQLListCatePids, ruleID).Scan(c, &pids)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NewDBError(appErr.DB_NO_ROWS, "", err)
		} else {
			return nil, appErr.NewDBError(appErr.DB_QUERY_FAIL, "", err)
		}
	}
	return pids, nil
}
