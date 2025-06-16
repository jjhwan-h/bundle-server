package org

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type TOrgGroup struct {
	bun.BaseModel `bun:"table:t_org_group"`

	GID     string  `bun:"gid,pk"           json:"gid"`
	GName   string  `bun:"gname,notnull"    json:"gname"`
	Comment *string `bun:"comment,nullzero" json:"comment,omitempty"`
	PID     string  `bun:"pid,notnull"      json:"pid"`
	Seq     uint16  `bun:"seq,notnull"      json:"seq"`

	RegDate time.Time  `bun:"reg_date,notnull,default:current_timestamp()" json:"reg_date"`
	ModDate *time.Time `bun:"mod_date" json:"mod_date,omitempty"`
}

type OrgGroupRepo interface {
	ListGidsRecursive(c context.Context, rootGcodes []string) ([]string, error)
}
