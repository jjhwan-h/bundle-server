package category

import (
	"context"

	"github.com/jjhwan-h/bundle-server/domain/casb/policy"
	"github.com/jjhwan-h/bundle-server/domain/common/profile"

	"github.com/uptrace/bun"
)

type TCategorySummary struct {
	bun.BaseModel `bun:"table:common.t_saas_category"`

	CID    uint16         `bun:"cid,pk,autoincrement" json:"cid"`
	PID    uint16         `bun:"pid,notnull,default:0" json:"pid"`
	CName  string         `bun:"cname,notnull" json:"cname"`
	Action profile.Action `bun:"action" json:"action"`
}

type CategoryService struct {
	CID    uint16        `bun:"cid,pk,autoincrement" json:"cid"`
	Action policy.Action `bun:"action" json:"action"`
}

type CategoryRepo interface {
	ListCategorySummaries(c context.Context) ([]TCategorySummary, error)
	ListCategoryServices(c context.Context, pidCates []policy.Pid) ([]CategoryService, error)
}
