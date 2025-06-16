package usecase

import (
	"bundle-server/domain/casb/policy"
	"bundle-server/domain/common/org"
	"bundle-server/domain/common/profile"
	"bundle-server/domain/integration/category"
	"context"
	"fmt"
	"strconv"
)

type CasbUsecase interface {
	BuildDataJson(c context.Context) (*Data, error)
}

type (
	Data struct {
		DefaultEffect string   `json:"default_effect"`
		Policies      []Policy `json:"policies"`
	}

	Policy struct {
		Priority   int16                      `json:"priority"`
		PolicyID   uint                       `json:"id"`
		PolicyName string                     `json:"name"`
		Subject    Subject                    `json:"subject"`
		Services   []category.CategoryService `json:"services"`
		Effect     string                     `json:"effect"`
	}

	Subject struct {
		Users  []string `json:"users"`
		Groups []string `json:"groups"`
	}

	casbUsecase struct {
		policySaasRepo       policy.PolicySaasRepo
		orgGroupRepo         org.OrgGroupRepo
		profileUserSubRepo   profile.ProfileUserSubRepo
		categoryRepo         category.CategoryRepo
		policySaasConfigRepo policy.PolicySaasConfigRepo
	}
)

func NewCasbUsecase(
	pr policy.PolicySaasRepo,
	or org.OrgGroupRepo,
	pur profile.ProfileUserSubRepo,
	cr category.CategoryRepo,
	pcr policy.PolicySaasConfigRepo,
) CasbUsecase {
	return &casbUsecase{
		policySaasRepo:       pr,
		orgGroupRepo:         or,
		profileUserSubRepo:   pur,
		categoryRepo:         cr,
		policySaasConfigRepo: pcr,
	}
}

func (cu *casbUsecase) BuildDataJson(c context.Context) (data *Data, err error) {
	data = &Data{
		DefaultEffect: "",
		Policies:      []Policy{},
	}

	err = cu.setDefaultEffect(c, data)
	if err != nil {
		return
	}

	err = cu.setPolicies(c, data)
	if err != nil {
		return
	}

	return
}

func (cu *casbUsecase) setDefaultEffect(c context.Context, data *Data) error {
	// [casb.t_policy_saas_config] effect 조회
	config, err := cu.policySaasConfigRepo.GetConfig(c)
	if err != nil {
		return handleErr("fetch t_policy_saas_config", err)
	}

	effect, err := strconv.Atoi(config.Effect)
	if err != nil {
		return handleErr("convert 'effect' value  to int", err)
	}

	code, err := convertCode(effect)
	if err != nil {
		return handleErr("convert 'effect' value to code", err)
	}

	data.DefaultEffect = code
	return nil
}

func (cu *casbUsecase) setPolicies(c context.Context, data *Data) error {
	// [casb.t_policy_saas] rule_id, rule_name, seq, enable 조회
	policies, err := cu.policySaasRepo.ListPolicies(c)
	if err != nil {
		return handleErr("get t_policy_saas", err)
	}
	data.Policies = make([]Policy, len(policies))

	for i, policy := range policies {
		data.Policies[i].Priority = policy.Seq
		data.Policies[i].PolicyID = policy.RuleID
		data.Policies[i].PolicyName = policy.RuleName

		enable, err := strconv.Atoi(policy.Enable)
		if err != nil {
			return handleErr("convert 'enable' value to int", err)
		}
		code, err := convertCode(enable)
		if err != nil {
			return handleErr("convert 'enable' value to code", err)
		}
		data.Policies[i].Effect = code

		err = cu.setSubject(c, data, policy, i)
		if err != nil {
			return err
		}

		err = cu.setServices(c, data, policy, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cu *casbUsecase) setSubject(c context.Context, data *Data, policy policy.TPolicySaas, idx int) error {
	data.Policies[idx].Subject = Subject{}
	data.Policies[idx].Subject.Users = []string{}
	data.Policies[idx].Subject.Groups = []string{}

	// [casb.t_profile_user_sub] gtype, gcode 조회
	groupAttrs, err := cu.policySaasRepo.ListGroupAttrs(c, policy.RuleID)
	if err != nil {
		return handleErr("query t_profile_user_sub", err)
	}

	var groups []string
	for _, groupAttr := range groupAttrs {
		if groupAttr.GType == 2 {
			// gypte이 2(user)인 경우 all gcode append
			data.Policies[idx].Subject.Users = append(data.Policies[idx].Subject.Users, groupAttr.GCode)
		} else if groupAttr.GType == 1 {
			groups = append(groups, groupAttr.GCode)
		}
	}
	// [common.t_org_group] gtype이 1(그룹)인 경우 하위부서까지 모두 조회 및 append
	gcodes, err := cu.orgGroupRepo.ListGidsRecursive(c, groups)
	if err != nil {
		return handleErr("query t_org_group", err)
	}

	data.Policies[idx].Subject.Groups = append(data.Policies[idx].Subject.Groups, gcodes...)
	return nil
}

func (cu *casbUsecase) setServices(c context.Context, data *Data, policy policy.TPolicySaas, idx int) error {
	data.Policies[idx].Services = []category.CategoryService{}

	// [casb.t_policy_saas_cate_mapping] pid 조회
	pids, err := cu.policySaasRepo.ListCatePids(c, policy.RuleID)
	if err != nil {
		return handleErr("query t_policy_saas_cate_mapping", err)
	}

	// [common.t_profile_saas_cate_sub, common.t_saas_category] cid, action 조회
	services, err := cu.categoryRepo.ListCategoryServices(c, pids)
	if err != nil {
		return handleErr("query t_profile_saas_cate_sub, common.t_saas_category", err)
	}

	data.Policies[idx].Services = append(data.Policies[idx].Services, services...)
	return nil
}

func convertCode(num int) (string, error) {
	switch num {
	case policy.Allow:
		return "allow", nil
	case policy.Deny:
		return "deny", nil
	default:
		return "", fmt.Errorf("invalid effect num: %d", num)
	}
}

func handleErr(action string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s failed: %w", action, err)
}
