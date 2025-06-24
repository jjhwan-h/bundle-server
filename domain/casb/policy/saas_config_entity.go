package policy

import (
	"context"

	"github.com/uptrace/bun"
)

type PolicySaasConfig struct {
	bun.BaseModel `bun:"table:casb_policy_saas_config"`
	Effect        string `bun:"effect" json:"effect"`
	Action        string `bun:"action" json:"action"`
}

type PolicySaasConfigRepo interface {
	GetConfig(c context.Context) (*PolicySaasConfig, error)
}
