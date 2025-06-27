package casbusecase

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"

	"github.com/jjhwan-h/bundle-server/domain/casb/policy"
	"github.com/jjhwan-h/bundle-server/domain/integration/category"
	"github.com/jjhwan-h/bundle-server/domain/sse/org"
	"github.com/jjhwan-h/bundle-server/domain/sse/profile"
	appErr "github.com/jjhwan-h/bundle-server/internal/errors"
)

type CasbUsecase interface {
	BuildDataJson(c context.Context) (*Data, error)
	BuildPatchJson(oldData *Data, data *Data) (*Patch, error)
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

	Patch struct {
		Data []PatchData `json:"data"`
	}

	PatchData struct {
		Op    string      `json:"op"`
		Path  string      `json:"path"`
		Value interface{} `json:"value,omitempty"`
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

func (cu *casbUsecase) BuildPatchJson(oldData *Data, data *Data) (*Patch, error) {
	patchData := getCasbPatch(oldData, data)
	if len(patchData) == 0 {
		return nil, appErr.ErrNoChanges
	}
	return &Patch{
		Data: patchData,
	}, nil
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

// GET 업데이트된 CASB 정책 데이터
func getCasbPatch(oldData *Data, data *Data) (changes []PatchData) {
	// default_effect
	if oldData.DefaultEffect != data.DefaultEffect {
		changes = append(changes, PatchData{
			Op:    "replace",
			Path:  "/default_effect",
			Value: data.DefaultEffect,
		})
	}
	// policies
	for _, newPolicy := range data.Policies {
		isExist := false
		for idx, oldPolicy := range oldData.Policies {
			if newPolicy.PolicyID == oldPolicy.PolicyID {
				isExist = true
				changes = append(changes, compareCasbPolicies(oldPolicy, newPolicy, idx)...)
				break
			}
		}
		if !isExist {
			changes = append(changes, PatchData{
				Op:    "upsert",
				Path:  "/policies",
				Value: newPolicy,
			})
		}
	}

	for idx, oldPolicy := range oldData.Policies {
		isExist := false
		for _, newPolicy := range data.Policies {
			if oldPolicy.PolicyID == newPolicy.PolicyID {
				isExist = true
				break
			}
		}
		if !isExist {
			changes = append(changes, PatchData{Op: "remove",
				Path:  fmt.Sprintf("/policies/%d", idx),
				Value: nil,
			})
		}
	}
	return
}

func compareCasbPolicies(oldPolicy, newPolicy Policy, idx int) (changes []PatchData) {
	prefix := fmt.Sprintf("/policies/%d", idx)

	if newPolicy.PolicyID != oldPolicy.PolicyID {
		changes = append(changes, PatchData{"replace", prefix + "/id", newPolicy.PolicyID})
	}
	if newPolicy.Priority != oldPolicy.Priority {
		changes = append(changes, PatchData{"replace", prefix + "/priority", newPolicy.Priority})
	}
	if newPolicy.PolicyName != oldPolicy.PolicyName {
		changes = append(changes, PatchData{"replace", prefix + "/name", newPolicy.PolicyName})
	}
	if newPolicy.Effect != oldPolicy.Effect {
		changes = append(changes, PatchData{"replace", prefix + "/effect", newPolicy.Effect})
	}
	if !slices.Equal(newPolicy.Subject.Users, oldPolicy.Subject.Users) {
		changes = append(changes, PatchData{"replace", prefix + "/subject/users", newPolicy.Subject.Users})
	}
	if !slices.Equal(newPolicy.Subject.Groups, oldPolicy.Subject.Groups) {
		changes = append(changes, PatchData{"replace", prefix + "/subject/groups", newPolicy.Subject.Groups})
	}

	if !equalService(newPolicy.Services, oldPolicy.Services) {
		changes = append(changes, PatchData{"replace", prefix + "/services", newPolicy.Services})
	}

	return
}

// 내부 객체 값은 동일하지만 객체 순서가 바뀐 경우에도 다른 것으로 처리됨.
func equalService(a, b []category.CategoryService) bool {
	return reflect.DeepEqual(a, b)
}

func handleErr(action string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s failed: %w", action, err)
}
