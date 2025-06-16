package policy

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type Action int

const (
	Allow int = 1
	Deny  int = 0
)

type TPolicySaas struct {
	bun.BaseModel `bun:"table:t_policy_saas"`
	RuleID        uint       `bun:"rule_id,pk,autoincrement" json:"rule_id"`
	BID           int16      `bun:"bid,notnull" json:"bid"`
	RuleName      string     `bun:"rule_name,notnull" json:"rule_name"`
	Action        Action     `bun:"action,notnull" json:"action"`
	PIDTime       uint       `bun:"pid_time,notnull" json:"pid_time"`
	Comment       *string    `bun:"comment,nullzero" json:"comment,omitempty"`
	Seq           int16      `bun:"seq,notnull" json:"seq"`
	Enable        string     `bun:"enable,notnull" json:"enable"`
	RegDate       time.Time  `bun:"reg_date,notnull,default:current_timestamp()" json:"reg_date"`
	ModDate       *time.Time `bun:"mod_date" json:"mod_date,omitempty"`
}

type GroupAttr struct {
	GType uint8  `bun:"gtype,pk"         json:"gtype"`
	GCode string `bun:"gcode,pk"         json:"gcode"`
}

type Pid uint

type PolicySaasRepo interface {
	ListPolicies(c context.Context) ([]TPolicySaas, error)
	ListGroupAttrs(c context.Context, ruleID uint) ([]GroupAttr, error)
	ListCatePids(c context.Context, ruleID uint) ([]Pid, error)
}
