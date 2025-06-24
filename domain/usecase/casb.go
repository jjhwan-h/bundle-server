package usecase

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jjhwan-h/bundle-server/domain/casb/policy"
	"github.com/jjhwan-h/bundle-server/domain/common/org"
	"github.com/jjhwan-h/bundle-server/domain/common/profile"
	"github.com/jjhwan-h/bundle-server/domain/integration/category"
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
	// [casb_policy_saas_config] effect 조회
	config, err := cu.policySaasConfigRepo.GetConfig(c)
	if err != nil {
		return handleErr("fetch casb_policy_saas_config", err)
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
	// [casb_policy_saas] rule_id, rule_name, seq, enable 조회
	policies, err := cu.policySaasRepo.ListPolicies(c)
	if err != nil {
		return handleErr("get casb_policy_saas", err)
	}

	for _, policy := range policies {
		// enable == 1인 경우에만 정책생성
		if policy.Enable != "1" {
			continue
		}

		tmpPolicy := Policy{}

		code, err := convertCode(int(policy.Action))
		if err != nil {
			return handleErr("convert 'enable' value to code", err)
		}

		tmpPolicy.Effect = code
		tmpPolicy.Priority = policy.Seq
		tmpPolicy.PolicyID = policy.RuleID
		tmpPolicy.PolicyName = policy.RuleName

		err = cu.setSubject(c, &tmpPolicy, policy)
		if err != nil {
			return err
		}

		err = cu.setServices(c, &tmpPolicy, policy)
		if err != nil {
			return err
		}
		data.Policies = append(data.Policies, tmpPolicy)
	}
	return nil
}

func (cu *casbUsecase) setSubject(c context.Context, data *Policy, policy policy.TPolicySaas) error {
	data.Subject = Subject{}
	data.Subject.Users = []string{}
	data.Subject.Groups = []string{}

	// [casb_profile_user_sub] gtype, gcode 조회
	groupAttrs, err := cu.policySaasRepo.ListGroupAttrs(c, policy.RuleID)
	if err != nil {
		return handleErr("query casb_profile_user_sub", err)
	}

	var groups []string
	for _, groupAttr := range groupAttrs {
		if groupAttr.GType == 2 {
			// gypte이 2(user)인 경우 all gcode append
			data.Subject.Users = append(data.Subject.Users, groupAttr.GCode)
		} else if groupAttr.GType == 1 {
			groups = append(groups, groupAttr.GCode)
		}
	}
	// [common_org_group] gtype이 1(그룹)인 경우 하위부서까지 모두 조회 및 append
	gcodes, err := cu.orgGroupRepo.ListGidsRecursive(c, groups)
	if err != nil {
		return handleErr("query common_org_group", err)
	}

	data.Subject.Groups = append(data.Subject.Groups, gcodes...)
	return nil
}

func (cu *casbUsecase) setServices(c context.Context, data *Policy, policy policy.TPolicySaas) error {
	data.Services = []category.CategoryService{}

	// [casb_policy_saas_cate_mapping] pid 조회
	pids, err := cu.policySaasRepo.ListCatePids(c, policy.RuleID)
	if err != nil {
		return handleErr("query casb_policy_saas_cate_mapping", err)
	}

	// [common_profile_saas_cate_sub, common.t_saas_category] cid, action 조회
	services, err := cu.categoryRepo.ListCategoryServices(c, pids)
	if err != nil {
		return handleErr("query common_profile_saas_cate_sub, common.t_saas_category", err)
	}

	data.Services = append(data.Services, services...)
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
