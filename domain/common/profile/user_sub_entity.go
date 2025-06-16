package profile

import (
	"context"

	"github.com/uptrace/bun"
)

// t_profile_saas_cate_sub의 action
type Action string

const (
	None       Action = "1"
	Read       Action = "2"
	Write      Action = "4"
	ReadNWrite Action = "7"
)

type TProfileUserSub struct {
	bun.BaseModel `bun:"table:t_profile_user_sub"`
	PID           uint   `bun:"pid,pk"           json:"pid"`
	GType         uint8  `bun:"gtype,pk"         json:"gtype"`
	GCode         string `bun:"gcode,pk"         json:"gcode"`

	TimeFrom *string `bun:"time_from,nullzero" json:"time_from,omitempty"`
	TimeTo   *string `bun:"time_to,nullzero"   json:"time_to,omitempty"`
	Comment  *string `bun:"comment,nullzero"   json:"comment,omitempty"`

	UseSIP   *bool   `bun:"use_sip,nullzero"   json:"use_sip,omitempty"`
	StaticIP *string `bun:"static_ip,nullzero" json:"static_ip,omitempty"`

	IsAPI bool `bun:"is_api,notnull"     json:"is_api"`
}

type ProfileUserSubRepo interface {
	ListGcodes(c context.Context, pid uint, gtype uint8) ([]string, error)
}

var (
	_ = None
	_ = Read
	_ = Write
	_ = ReadNWrite
)
